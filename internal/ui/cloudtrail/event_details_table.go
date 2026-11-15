package cloudtrail

import (
	"encoding/json"
	"fmt"

	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
)

// eventRecord is the full record of an event, which the api gives as json text
type eventRecord map[string]any

// parseEventRecord is the record of the event, which is nil for text that is
// not a json object
func parseEventRecord(text string) eventRecord {
	var record eventRecord
	if err := json.Unmarshal([]byte(text), &record); err != nil {
		return nil
	}
	return record
}

// text is the value at the path of keys, such as "userIdentity", "arn", which
// is empty when it is not set
func (inst eventRecord) text(path ...string) string {
	var value any = map[string]any(inst)
	for _, key := range path {
		var object, ok = value.(map[string]any)
		if !ok {
			return ""
		}
		value = object[key]
	}

	switch value := value.(type) {
	case nil, map[string]any, []any:
		return ""
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}

// EventDetailsTable shows an event of the history. The history holds all of
// each event, so showing one needs no request.
type EventDetailsTable struct {
	*core.DetailsTable
	event types.Event
}

func NewEventDetailsTable(appCtx *core.AppContext) *EventDetailsTable {
	var table = &EventDetailsTable{
		DetailsTable: core.NewDetailsTable("Event Details", appCtx),
	}
	table.populateDetailsTable()
	return table
}

func (inst *EventDetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	var rawData any

	if inst.event.EventId != nil {
		var event = inst.event
		var record = parseEventRecord(aws.ToString(event.CloudTrailEvent))

		tableData = []core.TableRow{
			{"Event Name", aws.ToString(event.EventName)},
			{"Event Time", core.FormatTime(event.EventTime)},
			{"Event Source", aws.ToString(event.EventSource)},
			{"User Name", aws.ToString(event.Username)},
			{"Access Key", aws.ToString(event.AccessKeyId)},
			{"Read Only", aws.ToString(event.ReadOnly)},
			{"Region", record.text("awsRegion")},
			{"Source IP", record.text("sourceIPAddress")},
			{"User Agent", record.text("userAgent")},
			{"Error Code", record.text("errorCode")},
			{"Error Message", record.text("errorMessage")},
			{"Event ID", aws.ToString(event.EventId)},
		}

		tableData = append(tableData, core.DetailsSection("Identity",
			core.TableRow{"Type", record.text("userIdentity", "type")},
			core.TableRow{"ARN", record.text("userIdentity", "arn")},
			core.TableRow{"Account", record.text("userIdentity", "accountId")},
		)...)

		var resources []core.TableRow
		for _, resource := range event.Resources {
			resources = append(resources, core.TableRow{
				aws.ToString(resource.ResourceType), aws.ToString(resource.ResourceName),
			})
		}
		tableData = append(tableData, core.DetailsSection("Resources", resources...)...)

		// The raw view shows the whole record, which has far more than the rows
		rawData = event
		if record != nil {
			rawData = record
		}
	}

	inst.SetRawData(rawData)
	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetEvent shows the event
func (inst *EventDetailsTable) SetEvent(event types.Event) {
	inst.event = event
	inst.populateDetailsTable()
}
