package cloudwatch

import (
	"errors"
	"slices"
	"strings"
	"time"

	"aws-tui/internal/settings"
	"aws-tui/internal/ui/core"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type InsightsQuery struct {
	Query     string
	StartTime time.Time
	EndTime   time.Time
}

// insightsRange is a time range to pick, ending now. Custom has no span, and
// leaves the times as they are entered.
type insightsRange struct {
	label string
	span  time.Duration
}

var insightsRanges = []insightsRange{
	{"Last 15 minutes", 15 * time.Minute},
	{"Last hour", time.Hour},
	{"Last 3 hours", 3 * time.Hour},
	{"Last 24 hours", 24 * time.Hour},
	{"Last 7 days", 7 * 24 * time.Hour},
	{"Custom", 0},
}

const (
	defaultInsightsRange = 2
	customInsightsRange  = 5
)

const defaultInsightsQuery = "fields @timestamp, @message, @log\n" +
	"| sort @timestamp desc\n" +
	"| limit 1000\n"

type InsightsQueryInputView struct {
	*tview.Flex
	RunButton  *core.Button
	StopButton *core.Button
	// ErrorMessageCallback reports what fails, such as saving the queries
	ErrorMessageCallback func(text string, a ...any)

	appCtx         *core.AppContext
	queryTextArea  *tview.TextArea
	rangeDropDown  *core.DropDown
	startDateInput *core.DateTimeInputField
	endDateInput   *core.DateTimeInputField
	nameInput      *core.InputField
	saveButton     *core.Button
	savedTable     *core.SelectableTable[string]
	queries        settings.InsightsQueries
	// settingRange is set while the times are put in by a range, which is not
	// them being entered
	settingRange bool
}

func NewInsightsQueryInputView(appContext *core.AppContext) *InsightsQueryInputView {
	var flex = tview.NewFlex().SetDirection(tview.FlexRow)
	var view = &InsightsQueryInputView{
		Flex:                 flex,
		RunButton:            core.NewButton("Run", appContext.Theme),
		StopButton:           core.NewButton("Stop", appContext.Theme),
		ErrorMessageCallback: func(text string, a ...any) {},

		appCtx:         appContext,
		queryTextArea:  tview.NewTextArea(),
		rangeDropDown:  core.NewDropDown(appContext.Theme),
		startDateInput: core.NewDateTimeInputField(appContext.Theme),
		endDateInput:   core.NewDateTimeInputField(appContext.Theme),
		nameInput:      core.NewInputField(appContext.Theme),
		saveButton:     core.NewButton("Save", appContext.Theme),
		savedTable: core.NewSelectableTable[string](
			"Saved Queries", core.TableRow{"Name"}, appContext,
		),
	}

	var separator = tview.NewBox()

	view.
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.queryTextArea, 0, 2, true).
				AddItem(view.savedTable, 0, 1, false),
			0, 1, true,
		).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.rangeDropDown, 0, 22, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.startDateInput, 0, 26, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.endDateInput, 0, 24, false),
			1, 0, false,
		).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(view.nameInput, 0, 2, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.saveButton, 0, 1, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.RunButton, 0, 1, false).
				AddItem(separator, 1, 0, false).
				AddItem(view.StopButton, 0, 1, false),
			1, 0, false,
		)

	view.queryTextArea.SetClipboard(
		func(s string) { clipboard.WriteAll(s) },
		func() string {
			var res, _ = clipboard.ReadAll()
			return res
		},
	)
	view.queryTextArea.SetSelectedStyle(appContext.Theme.GetFocusFormItemStyle())
	view.queryTextArea.
		SetTitle("Query").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	view.startDateInput.SetLabel("Start ")
	view.endDateInput.SetLabel("End ")
	view.nameInput.SetLabel("Name ")
	view.initRanges()
	view.initSavedQueries()

	return view
}

// initRanges adds the ranges to pick from, which put their times in as they are
// picked and when the query is run. Entering a time moves to Custom.
func (inst *InsightsQueryInputView) initRanges() {
	inst.rangeDropDown.SetLabel("Range ")
	for _, option := range insightsRanges {
		inst.rangeDropDown.AddOption(option.label, nil)
	}

	var enteredTime = func(string) {
		if !inst.settingRange {
			inst.rangeDropDown.SetCurrentOption(customInsightsRange)
		}
	}
	inst.startDateInput.SetChangedFunc(enteredTime)
	inst.endDateInput.SetChangedFunc(enteredTime)

	inst.rangeDropDown.SetSelectedFunc(func(text string, index int) {
		inst.applyRange(index, time.Now())
	})
	inst.rangeDropDown.SetCurrentOption(defaultInsightsRange)
}

// applyRange puts the times of the range ending at now in. Custom leaves the
// times that were entered.
func (inst *InsightsQueryInputView) applyRange(index int, now time.Time) {
	var span = insightsRanges[index].span
	if span == 0 {
		return
	}

	inst.settingRange = true
	defer func() { inst.settingRange = false }()
	inst.startDateInput.SetTextTime(now.Add(-span))
	inst.endDateInput.SetTextTime(now)
}

// initSavedQueries loads the queries that were kept, and puts the one that was
// run last in the query form
func (inst *InsightsQueryInputView) initSavedQueries() {
	if path := inst.appCtx.QueriesPath; len(path) > 0 {
		var loaded, err = settings.LoadQueries(path)
		if err != nil {
			inst.appCtx.Logger.Print(err)
		}
		inst.queries = loaded
	}

	if len(inst.queries.Last) > 0 {
		inst.queryTextArea.SetText(inst.queries.Last, false)
	} else {
		inst.queryTextArea.SetText(defaultInsightsQuery, false)
	}

	inst.savedTable.SetSelectionChangedFunc(func(row, column int) {})
	inst.savedTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.queryTextArea.SetText(inst.savedTable.GetPrivateData(row, 0), true)
		inst.nameInput.SetText(inst.savedTable.GetCellText(row, 0))
		inst.appCtx.App.SetFocus(inst.queryTextArea)
	})
	inst.savedTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == core.APP_KEY_BINDINGS.RemoveItem {
			inst.deleteSelectedQuery()
			return nil
		}
		return event
	})
	inst.saveButton.SetSelectedFunc(func() { inst.SaveQuery() })
	inst.showSavedQueries()
}

func (inst *InsightsQueryInputView) showSavedQueries() {
	var rows []core.TableRow
	var queries []string
	for _, saved := range inst.queries.Saved {
		rows = append(rows, core.TableRow{saved.Name})
		queries = append(queries, saved.Query)
	}

	inst.savedTable.SetData(rows, queries, 0)
	inst.savedTable.GetCell(0, 0).SetExpansion(1)
	inst.savedTable.Select(1, 0)
}

// persist keeps the queries, which is reported if it fails
func (inst *InsightsQueryInputView) persist() {
	var path = inst.appCtx.QueriesPath
	if len(path) == 0 {
		return
	}
	if err := settings.SaveQueries(path, inst.queries); err != nil {
		inst.ErrorMessageCallback("%s", err.Error())
	}
}

// SaveQuery keeps the query that is entered under the name that is entered,
// replacing the one that has the name
func (inst *InsightsQueryInputView) SaveQuery() {
	var name = strings.TrimSpace(inst.nameInput.GetText())
	var query = strings.TrimSpace(inst.queryTextArea.GetText())
	if len(name) == 0 {
		inst.ErrorMessageCallback("Enter a name to save the query as")
		return
	}
	if len(query) == 0 {
		inst.ErrorMessageCallback("The query is empty")
		return
	}

	var saved = settings.SavedQuery{Name: name, Query: query}
	var idx = slices.IndexFunc(inst.queries.Saved, func(q settings.SavedQuery) bool { return q.Name == name })
	if idx >= 0 {
		inst.queries.Saved[idx] = saved
	} else {
		inst.queries.Saved = append(inst.queries.Saved, saved)
	}
	slices.SortFunc(inst.queries.Saved, func(a, b settings.SavedQuery) int { return strings.Compare(a.Name, b.Name) })

	inst.persist()
	inst.showSavedQueries()
}

func (inst *InsightsQueryInputView) deleteSelectedQuery() {
	var row, _ = inst.savedTable.GetTable().GetSelection()
	if row < 1 || row > len(inst.queries.Saved) {
		return
	}

	inst.queries.Saved = slices.Delete(inst.queries.Saved, row-1, row)
	inst.persist()
	inst.showSavedQueries()
}

// RememberQuery keeps the query as the one run last, to be put back next time
func (inst *InsightsQueryInputView) RememberQuery(query InsightsQuery) {
	if inst.queries.Last == query.Query {
		return
	}
	inst.queries.Last = query.Query
	inst.persist()
}

func (inst *InsightsQueryInputView) rangeIndex() int {
	var index, _ = inst.rangeDropDown.GetCurrentOption()
	return index
}

// NavigationRows are the views to move between in the order they are laid out
func (inst *InsightsQueryInputView) NavigationRows() [][]core.View {
	return [][]core.View{
		{inst.queryTextArea, inst.savedTable},
		{inst.rangeDropDown, inst.startDateInput, inst.endDateInput},
		{inst.nameInput, inst.saveButton, inst.RunButton, inst.StopButton},
	}
}

// GenerateQuery is the query as it is entered, which is empty with an error if
// it can't be run
func (inst *InsightsQueryInputView) GenerateQuery() (InsightsQuery, error) {
	var empty = InsightsQuery{}
	// A range that is picked ends when the query is run, not when it was picked
	inst.applyRange(inst.rangeIndex(), time.Now())
	var query = InsightsQuery{Query: strings.TrimSpace(inst.queryTextArea.GetText())}

	var err error
	if query.StartTime, err = inst.startDateInput.ValidateInput(); err != nil {
		return empty, err
	}
	if query.EndTime, err = inst.endDateInput.ValidateInput(); err != nil {
		return empty, err
	}

	if len(query.Query) == 0 {
		return empty, errors.New("The query is empty")
	}
	if query.EndTime.Before(query.StartTime) {
		return empty, errors.New("The end time is before the start time")
	}

	return query, nil
}
