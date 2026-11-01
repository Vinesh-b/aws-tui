package eventbridge

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
)

func validFormValues() scheduleFormValues {
	return scheduleFormValues{
		Name:       "nightly",
		Expression: "rate(1 day)",
		TargetArn:  lambdaArn,
		RoleArn:    roleArn,
		Enabled:    true,
	}
}

func TestValidScheduleValues(t *testing.T) {
	for name, tc := range map[string]struct {
		change   func(v *scheduleFormValues)
		creating bool
		wantErr  string
	}{
		"valid":                       {func(v *scheduleFormValues) {}, true, ""},
		"cron":                        {func(v *scheduleFormValues) { v.Expression = "cron(0 12 * * ? *)" }, true, ""},
		"at":                          {func(v *scheduleFormValues) { v.Expression = "at(2030-01-01T00:00:00)" }, true, ""},
		"window":                      {func(v *scheduleFormValues) { v.Window = "15" }, true, ""},
		"empty name":                  {func(v *scheduleFormValues) { v.Name = " " }, true, "Name is empty"},
		"empty name is fine for edit": {func(v *scheduleFormValues) { v.Name = "" }, false, ""},
		"bad name":                    {func(v *scheduleFormValues) { v.Name = "my schedule" }, true, "up to 64 letters"},
		"long name":                   {func(v *scheduleFormValues) { v.Name = strings.Repeat("a", 65) }, true, "up to 64 letters"},
		"bad group":                   {func(v *scheduleFormValues) { v.Group = "a/b" }, true, "Group names"},
		"empty expression":            {func(v *scheduleFormValues) { v.Expression = "" }, true, "Schedule is empty"},
		"bad expression":              {func(v *scheduleFormValues) { v.Expression = "every day" }, true, "rate(...)"},
		"unclosed expression":         {func(v *scheduleFormValues) { v.Expression = "rate(1 day" }, true, "rate(...)"},
		"empty target":                {func(v *scheduleFormValues) { v.TargetArn = "" }, true, "Target ARN is empty"},
		"empty role":                  {func(v *scheduleFormValues) { v.RoleArn = "" }, false, "Role ARN is empty"},
		"window not a number":         {func(v *scheduleFormValues) { v.Window = "soon" }, true, "Flexible window"},
		"window too long":             {func(v *scheduleFormValues) { v.Window = "1441" }, true, "Flexible window"},
		"negative window":             {func(v *scheduleFormValues) { v.Window = "-1" }, true, "Flexible window"},
	} {
		t.Run(name, func(t *testing.T) {
			var values = validFormValues()
			tc.change(&values)

			var _, err = validScheduleValues(values, tc.creating)

			if tc.wantErr == "" && err != nil {
				t.Errorf("unexpected error %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Errorf("error = %v, want one containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidScheduleValues__TrimsAndDefaultsTheGroup(t *testing.T) {
	var values = validFormValues()
	values.Name = " nightly "
	values.Expression = " rate(1 day) "

	var got, err = validScheduleValues(values, true)

	if err != nil || got.Name != "nightly" || got.Group != "default" || got.Expression != "rate(1 day)" {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestNewCreateScheduleInput__LeavesOutEmptyOptionalSettings(t *testing.T) {
	var values, _ = validScheduleValues(validFormValues(), true)
	values.Enabled = false

	var input = newCreateScheduleInput(values)

	if aws.ToString(input.GroupName) != "default" || input.State != types.ScheduleStateDisabled ||
		input.FlexibleTimeWindow.Mode != types.FlexibleTimeWindowModeOff {
		t.Errorf("input = %+v", input)
	}
	if input.Description != nil || input.ScheduleExpressionTimezone != nil || input.Target.Input != nil {
		t.Errorf("expected optional settings to be nil, got %+v %+v", input, input.Target)
	}
}

func TestNewEditScheduleInput__ChangesTheFormSettingsAndKeepsTheRest(t *testing.T) {
	var schedule = newSchedule("default", "nightly")
	var values = scheduleValuesOf(schedule)
	values.Expression = "rate(2 hours)"
	values.Window = "10"
	values.Enabled = false
	values.Input = ""

	var input = newEditScheduleInput(schedule, values)

	if aws.ToString(input.ScheduleExpression) != "rate(2 hours)" || input.State != types.ScheduleStateDisabled {
		t.Errorf("input = %+v", input)
	}
	if aws.ToInt32(input.FlexibleTimeWindow.MaximumWindowInMinutes) != 10 ||
		input.FlexibleTimeWindow.Mode != types.FlexibleTimeWindowModeFlexible {
		t.Errorf("window = %+v", input.FlexibleTimeWindow)
	}
	if input.Target.Input != nil {
		t.Errorf("target input = %q, want it cleared", aws.ToString(input.Target.Input))
	}
	if aws.ToInt32(input.Target.RetryPolicy.MaximumRetryAttempts) != 3 ||
		aws.ToString(input.Target.DeadLetterConfig.Arn) != queueArn || aws.ToString(input.KmsKeyArn) != kmsArn {
		t.Errorf("settings the form doesn't have were not kept: %+v %+v", input, input.Target)
	}
	if aws.ToString(schedule.Target.Input) != `{"job":"report"}` {
		t.Error("the schedule given was changed")
	}
}

func TestNewUpdateScheduleInput__CopiesEverySetting(t *testing.T) {
	var schedule = newSchedule("ops", "nightly")

	var input = newUpdateScheduleInput(schedule)

	if aws.ToString(input.Name) != "nightly" || aws.ToString(input.GroupName) != "ops" ||
		aws.ToString(input.ScheduleExpression) != "rate(1 day)" || aws.ToString(input.Description) != "Nightly report" ||
		aws.ToString(input.ScheduleExpressionTimezone) != "Europe/London" || input.State != types.ScheduleStateEnabled ||
		aws.ToString(input.Target.Arn) != lambdaArn || input.ActionAfterCompletion != types.ActionAfterCompletionNone {
		t.Errorf("input = %+v", input)
	}
	input.Target.Arn = aws.String("changed")
	if aws.ToString(schedule.Target.Arn) != lambdaArn {
		t.Error("changing the input changed the schedule")
	}
}

func TestScheduleValuesOf__ShowsTheWindowOnlyWhenFlexible(t *testing.T) {
	var schedule = newSchedule("default", "nightly")
	if got := scheduleValuesOf(schedule).Window; got != "5" {
		t.Errorf("window = %q, want 5", got)
	}

	schedule.FlexibleTimeWindow = &types.FlexibleTimeWindow{Mode: types.FlexibleTimeWindowModeOff}
	if got := scheduleValuesOf(schedule).Window; got != "" {
		t.Errorf("window = %q, want it empty", got)
	}
}

func TestExtractScheduleTargetJson__ShowsJsonInputAsJsonAndLeavesOutUnsetSettings(t *testing.T) {
	var got = extractScheduleTargetJson(newSchedule("default", "nightly"))

	for _, want := range []string{`"Input":{"job":"report"}`, `"MaximumRetryAttempts":3`, `"Arn":"` + queueArn} {
		if !strings.Contains(got, want) {
			t.Errorf("%s does not contain %s", got, want)
		}
	}
	if strings.Contains(got, "null") || strings.Contains(got, "EcsParameters") {
		t.Errorf("%s has unset settings", got)
	}

	if got := extractScheduleTargetJson(scheduler.GetScheduleOutput{}); got != "" {
		t.Errorf("no target gave %q", got)
	}
}
