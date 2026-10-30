package dynamodb

import (
	"testing"

	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func newScanView(projection string, filters ...filterFields) *DynamoDBScanInputView {
	var view = NewDynamoDBScanInputView(uitest.NewAppContext())
	view.projectedAttributesInput.SetText(projection)
	for i, fields := range filters {
		setFilterFields(view.filterInputViews[i], fields)
	}
	return view
}

func TestScanInputView__RequiresFirstFilter(t *testing.T) {
	var _, err = newScanView("").GenerateScanExpression()

	if err == nil || err.Error() != "MISSING_REQUIRED_INPUT: Attribute name not set" {
		t.Errorf("error = %v, want missing attribute name", err)
	}
}

func TestScanInputView__SingleFilter(t *testing.T) {
	var view = newScanView("", filterFields{"age", "number", "between", "1", "5"})

	var expr, err = view.GenerateScanExpression()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := *expr.Filter(); got != "#0 BETWEEN :0 AND :1" {
		t.Errorf("filter = %q", got)
	}
	if expr.Projection() != nil {
		t.Errorf("unexpected projection %q", *expr.Projection())
	}
}

func TestScanInputView__CombinesValidFiltersWithAnd(t *testing.T) {
	var view = newScanView("",
		filterFields{"age", "number", "gt", "18", ""},
		filterFields{"name", "string", "exists", "", ""},
	)

	var expr, err = view.GenerateScanExpression()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := *expr.Filter(); got != "(#0 > :0) AND (attribute_exists (#1))" {
		t.Errorf("filter = %q", got)
	}
}

func TestScanInputView__IgnoresInvalidAdditionalFilters(t *testing.T) {
	var view = newScanView("",
		filterFields{"age", "number", "gt", "18", ""},
		filterFields{"name", "string", "like", "x", ""},
	)

	var expr, err = view.GenerateScanExpression()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := *expr.Filter(); got != "#0 > :0" {
		t.Errorf("filter = %q, want %q", got, "#0 > :0")
	}
}

func TestScanInputView__ProjectsListedAttributes(t *testing.T) {
	var view = newScanView("id,name", filterFields{"age", "number", "gt", "18", ""})

	var expr, err = view.GenerateScanExpression()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := *expr.Projection(); got != "#1, #2" {
		t.Errorf("projection = %q, want %q", got, "#1, #2")
	}
	if names := expr.Names(); names["#1"] != "id" || names["#2"] != "name" {
		t.Errorf("names = %v, want #1=id and #2=name", names)
	}
}

func TestScanInputView__NavigatesIntoAndOutOfTheFilters(t *testing.T) {
	var view = NewDynamoDBScanInputView(uitest.NewAppContext())
	var first, second = view.filterInputViews[0], view.filterInputViews[1]
	var app = view.appCtx.App

	app.SetFocus(view.projectedAttributesInput)
	view.navigator.NavigateWithin(core.APP_KEY_BINDINGS.ViewFocusDown)
	if app.GetFocus() != first.AttributeNameInput {
		t.Fatalf("Expected focus on the attribute of the first filter")
	}

	view.navigator.NavigateWithin(core.APP_KEY_BINDINGS.ViewFocusRight)
	view.navigator.NavigateWithin(core.APP_KEY_BINDINGS.ViewFocusRight)
	if app.GetFocus() != first.Condition {
		t.Fatalf("Expected focus on the condition of the first filter")
	}

	view.navigator.NavigateWithin(core.APP_KEY_BINDINGS.ViewFocusDown)
	if app.GetFocus() != second.AttributeNameInput {
		t.Fatalf("Expected focus on the attribute of the second filter")
	}
}
