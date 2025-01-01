package servicetables

import (
	"aws-tui/internal/pkg/errors"
	"aws-tui/internal/pkg/ui/core"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DynamoDBQueryInputView struct {
	QueryDoneButton   *tview.Button
	QueryCancelButton *tview.Button
	RootView          *tview.Flex

	logger              *log.Logger
	filterView          *FilterInputView
	pkInput             *tview.InputField
	skInput             *tview.InputField
	skComparatorInput   *tview.InputField
	projectedAttributes []string
	selectedIndex       string
	tableName           string
	indexes             []string
	pkName              string
	skName              string
	tabNavigator        *core.ViewNavigation
}

func NewDynamoDBQueryInputView(app *tview.Application, logger *log.Logger) *DynamoDBQueryInputView {
	var pkInput = tview.NewInputField().SetLabel("PK ").SetFieldWidth(0)
	var skInput = tview.NewInputField().SetLabel("SK ").SetFieldWidth(0)
	var skComparitorInput = tview.NewInputField().SetLabel("Comparator ").SetFieldWidth(8)
	var filterInputView = NewFilterInputView(app, logger)
	var doneButton = tview.NewButton("Done")
	var cancelButton = tview.NewButton("Cancel")

	var wrapper = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pkInput, 1, 0, true).
		AddItem(tview.NewBox(), 1, 0, true).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(skComparitorInput, 19, 0, true).
				AddItem(tview.NewBox(), 1, 0, true).
				AddItem(skInput, 0, 1, true),
			0, 1, true,
		).
		AddItem(filterInputView.RootView, 2, 0, true).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(doneButton, 0, 1, true).
				AddItem(tview.NewBox(), 1, 0, true).
				AddItem(cancelButton, 0, 1, true),
			1, 0, true,
		)

	var tabNavigator = core.NewViewNavigation(wrapper,
		[]core.View{
			pkInput,
			skComparitorInput,
			skInput,
			filterInputView.AttributeNameInput,
			filterInputView.AttributeTypeInput,
			filterInputView.Condition,
			doneButton,
			cancelButton,
		},
		app,
	)

	return &DynamoDBQueryInputView{
		QueryDoneButton:   doneButton,
		QueryCancelButton: cancelButton,
		RootView:          wrapper,

		logger:            logger,
		pkInput:           pkInput,
		skInput:           skInput,
		skComparatorInput: skComparitorInput,
		filterView:        filterInputView,
		tabNavigator:      tabNavigator,
	}
}

func (inst *DynamoDBQueryInputView) GenerateQueryExpression() (expression.Expression, error) {
	var pk = strings.TrimSpace(inst.pkInput.GetText())
	var sk = strings.TrimSpace(inst.skInput.GetText())
	var comp = strings.TrimSpace(strings.ToLower(inst.skComparatorInput.GetText()))

	if len(pk) == 0 {
		return expression.Expression{}, errors.NewDDBViewError(
			errors.MissingRequiredInput,
			"Partition Key value not provided",
		)
	}

	var keyCond = expression.
		Key(inst.pkName).Equal(expression.Value(pk))

	if len(sk) > 0 && len(inst.skName) > 0 && len(comp) > 0 {
		switch comp {
		case "eq":
			keyCond = keyCond.And(expression.
				Key(inst.skName).
				Equal(expression.Value(sk)),
			)
		case "lt":
			keyCond = keyCond.And(expression.
				Key(inst.skName).
				LessThan(expression.Value(sk)),
			)
		case "gt":
			keyCond = keyCond.And(expression.
				Key(inst.skName).
				GreaterThan(expression.Value(sk)),
			)
		case "lte":
			keyCond = keyCond.And(expression.
				Key(inst.skName).
				LessThanEqual(expression.Value(sk)),
			)
		case "gte":
			keyCond = keyCond.And(expression.
				Key(inst.skName).
				GreaterThanEqual(expression.Value(sk)),
			)
		case "begins":
			keyCond = keyCond.And(expression.
				Key(inst.skName).
				BeginsWith(sk),
			)
		default:
			return expression.Expression{}, errors.NewDDBViewError(
				errors.InvalidOption,
				"Invalid condition",
			)
		}
	}

	var exprBuilder = expression.NewBuilder()

	var filterCond, filtErr = inst.filterView.GenerateFilterCondition()
	if filtErr != nil {
		return expression.Expression{}, filtErr
	}

	if filterCond.IsSet() {
		exprBuilder.WithFilter(filterCond)
	}

	var expr, err = exprBuilder.
		WithKeyCondition(keyCond).
		Build()
	if err != nil {
		inst.logger.Printf("Failed to build expression for query: %v\n", err)
	}

	return expr, err
}

func (inst *DynamoDBQueryInputView) SetSelectedTable(tableName string) {
	inst.tableName = tableName
}

func (inst *DynamoDBQueryInputView) SetPartitionKeyName(pk string) {
	inst.pkName = pk
}

func (inst *DynamoDBQueryInputView) SetSortKeyName(sk string) {
	inst.skName = sk
}

func (inst *DynamoDBQueryInputView) SetTableIndexes(indexes []string) {
	inst.indexes = indexes
}

type FilterInputView struct {
	AttributeNameInput *tview.InputField
	AttributeTypeInput *tview.InputField
	Condition          *tview.InputField
	Value1             *tview.InputField
	Value2             *tview.InputField
	RootView           *tview.Flex

	tabNavigator *core.ViewNavigation
	logger       *log.Logger
}

func NewFilterInputView(app *tview.Application, logger *log.Logger) *FilterInputView {
	var attrNameInput = tview.NewInputField().SetLabel("Attribute ")
	var attrTypeInput = tview.NewInputField().SetLabel("Type ")
	var conditionInput = tview.NewInputField().SetLabel("Condition ")
	var value1Input = tview.NewInputField().SetLabel("Value ")
	var value2Input = tview.NewInputField().SetLabel("Value ")

	var spacerView = tview.NewBox()
	var line2View = tview.NewFlex().SetDirection(tview.FlexColumn)
	var line1View = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(attrNameInput, 0, 1, true).
		AddItem(spacerView, 1, 0, true).
		AddItem(attrTypeInput, 13, 0, true).
		AddItem(spacerView, 1, 0, true).
		AddItem(conditionInput, 18, 0, true)

	var wrapper = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(line1View, 1, 0, true).
		AddItem(line2View, 1, 0, true)

	var tabNavigator = core.NewViewNavigation(wrapper,
		[]core.View{
			attrNameInput,
			attrTypeInput,
			conditionInput,
		},
		app,
	)

	conditionInput.SetDoneFunc(func(key tcell.Key) {
		var condValue = conditionInput.GetText()
		line2View.Clear()
		tabNavigator.UpdateOrderedViews([]core.View{
			attrNameInput,
			attrTypeInput,
			conditionInput,
		}, 2)

		switch condValue {
		case "exist", "notexist":
		case "between":
			line2View.
				AddItem(value1Input, 0, 1, false).
				AddItem(tview.NewBox(), 1, 0, false).
				AddItem(value2Input, 0, 1, false)
			tabNavigator.UpdateOrderedViews([]core.View{
				attrNameInput,
				attrTypeInput,
				conditionInput,
				value1Input,
				value2Input,
			}, 2)
		default:
			tabNavigator.UpdateOrderedViews([]core.View{
				attrNameInput,
				attrTypeInput,
				conditionInput,
				value1Input,
			}, 2)
			line2View.AddItem(value1Input, 0, 1, false)
		}
	})

	return &FilterInputView{
		AttributeNameInput: attrNameInput,
		AttributeTypeInput: attrTypeInput,
		Condition:          conditionInput,
		Value1:             value1Input,
		Value2:             value2Input,
		RootView:           wrapper,

		logger:       logger,
		tabNavigator: tabNavigator,
	}
}

func (inst *FilterInputView) isConditionAllowed(attrType string, condition string) bool {
	var allowedSet = StringSet{}
	var allowedCond = []string{}

	switch attrType {
	case "null":
		allowedCond = []string{"exists", "nexists"}
	case "bool":
		allowedCond = []string{"eq", "neq", "exists", "nexists"}
	case "number":
		allowedCond = []string{
			"eq", "neq", "lt", "lte", "gt", "gte",
			"exists", "nexists", "between",
		}
	case "string", "binary":
		allowedCond = []string{
			"eq", "neq", "lt", "lte", "gt", "gte",
			"exists", "nexists", "between", "contains", "begins",
		}
	}

	for _, c := range allowedCond {
		allowedSet[c] = struct{}{}
	}

	var _, found = allowedSet[condition]
	return found
}

func (inst *FilterInputView) parseValue(value string, dataType string) (any, error) {
	var parsedValue any
	var err error = nil
	switch dataType {
	case "bool":
		parsedValue, err = strconv.ParseBool(value)
	case "number":
		parsedValue, err = strconv.ParseFloat(value, 64)
	default:
		parsedValue = value
	}

	return parsedValue, err
}

func (inst *FilterInputView) GenerateFilterCondition() (expression.ConditionBuilder, error) {
	var attrName = strings.TrimSpace(inst.AttributeNameInput.GetText())
	var attrType = strings.ToLower(inst.AttributeTypeInput.GetText())
	var attrValue1 = strings.TrimSpace(inst.Value1.GetText())
	var attrValue2 = strings.TrimSpace(inst.Value2.GetText())
	var cond = strings.TrimSpace(strings.ToLower(inst.Condition.GetText()))

	var filterCond = expression.ConditionBuilder{}

	if len(attrName) == 0 || len(attrType) == 0 || len(cond) == 0 {
		return filterCond, nil
	}

	if !inst.isConditionAllowed(attrType, cond) {
		return filterCond, errors.NewDDBViewError(
			errors.InvalidOption,
			fmt.Sprintf("`%v` does not support condition `%v`", attrType, cond),
		)
	}

	switch cond {
	case "exists":
		filterCond = expression.Name(attrName).AttributeExists()
		return filterCond, nil
	case "nexists":
		filterCond = expression.Name(attrName).AttributeNotExists()
		return filterCond, nil
	}

	var parsedValue1, val1Err = inst.parseValue(attrValue1, attrType)
	if val1Err != nil {
		return filterCond, errors.NewDDBViewError(
			errors.InvalidOption,
			fmt.Sprintf("Value 1 conversion failed %v", val1Err),
		)
	}

	switch cond {
	case "eq":
		filterCond = expression.Name(attrName).Equal(expression.Value(parsedValue1))
	case "neq":
		filterCond = expression.Name(attrName).NotEqual(expression.Value(parsedValue1))
	case "lt":
		filterCond = expression.Name(attrName).LessThan(expression.Value(parsedValue1))
	case "lte":
		filterCond = expression.Name(attrName).LessThanEqual(expression.Value(parsedValue1))
	case "gt":
		filterCond = expression.Name(attrName).GreaterThan(expression.Value(parsedValue1))
	case "gte":
		filterCond = expression.Name(attrName).GreaterThanEqual(expression.Value(parsedValue1))
	case "contains":
		filterCond = expression.Name(attrName).Contains(parsedValue1)
	case "begins":
		filterCond = expression.Name(attrName).BeginsWith(parsedValue1.(string))
	case "between":
		var parsedValue2, val2Err = inst.parseValue(attrValue2, attrType)
		if val2Err != nil {
			return filterCond, errors.NewDDBViewError(
				errors.InvalidOption,
				fmt.Sprintf("Value 2 conversion failed %v", val2Err),
			)
		}

		filterCond = expression.Name(attrName).Between(
			expression.Value(parsedValue1),
			expression.Value(parsedValue2),
		)

	default:
		return filterCond, errors.NewDDBViewError(
			errors.InvalidOption,
			"Unsupported condition given",
		)
	}

	return filterCond, nil
}

type DynamoDBScanInputView struct {
	ScanDoneButton   *tview.Button
	ScanCancelButton *tview.Button
	RootView         *tview.Flex

	logger                   *log.Logger
	filterInputViews         [3]*FilterInputView
	projectedAttributesInput *tview.InputField
	projectedAttributes      []string
	tableName                string
	indexes                  []string
	selectedIndex            string
}

func NewDynamoDBScanInputView(app *tview.Application, logger *log.Logger) *DynamoDBScanInputView {
	var filterInputViews = [3]*FilterInputView{
		NewFilterInputView(app, logger),
		NewFilterInputView(app, logger),
		NewFilterInputView(app, logger),
	}

	var separater = tview.NewBox()
	var doneButton = tview.NewButton("Done")
	var cancelButton = tview.NewButton("Cancel")
	var projAttrInput = tview.NewInputField().
		SetLabel("Attribute Projection ").
		SetPlaceholder("id,timestamp,name")

	var wrapper = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(projAttrInput, 0, 1, false).
		AddItem(separater, 1, 0, true)

	for _, view := range filterInputViews {
		wrapper.AddItem(view.RootView, 3, 0, true)
	}

	wrapper.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(doneButton, 0, 1, true).
			AddItem(separater, 1, 0, true).
			AddItem(cancelButton, 0, 1, true),
		1, 0, true,
	)

	return &DynamoDBScanInputView{
		RootView:         wrapper,
		ScanDoneButton:   doneButton,
		ScanCancelButton: cancelButton,

		logger:                   logger,
		filterInputViews:         filterInputViews,
		projectedAttributesInput: projAttrInput,
		projectedAttributes:      nil,
		tableName:                "",
		indexes:                  nil,
		selectedIndex:            "",
	}
}

func (inst *DynamoDBScanInputView) GenerateScanExpression() (expression.Expression, error) {
	//	var filterCond expression.ConditionBuilder
	//	for _, filterView := range inst.filterInputViews {
	//		var cond, err = filterView.GenerateFilterCondition()
	//		if err == nil {
	//			filterCond.And(cond)
	//		}
	//	}
	var exprBuilder = expression.NewBuilder()
	var filterCond, filtErr = inst.filterInputViews[0].GenerateFilterCondition()
	if filtErr != nil {
		return expression.Expression{}, filtErr
	}

	if filterCond.IsSet() {
		exprBuilder.WithFilter(filterCond)
	}

	var projectionText = strings.TrimSpace(inst.projectedAttributesInput.GetText())
	var atterStrings = strings.Split(projectionText, ",")

	var names = []expression.NameBuilder{}
	for _, attr := range atterStrings {
		inst.logger.Printf("Adding name: %v\n", attr)
		names = append(names, expression.Name(attr))
	}
	if len(names) > 0 {
        // FAILING: Build error: unset parameter: Builder
		var projection = expression.NamesList(names[0], names[1:]...)
		exprBuilder.WithProjection(projection)
	}

	var expr, err = exprBuilder.Build()
	if err != nil {
		return expression.Expression{}, errors.WrapDynamoDBSearchError(
			err, errors.FailedToBuildExpression, "Failed to build Scan expression",
		)
	}

	return expr, nil
}

func (inst *DynamoDBScanInputView) SetSelectedTable(tableName string) {
	inst.tableName = tableName
}

func (inst *DynamoDBScanInputView) SetTableIndexes(indexes []string) {
	inst.indexes = indexes
}

const (
	QUERY_PAGE_NAME = "QUERY"
	SCAN_PAGE_NAME  = "SCAN"
	MAIN_PAGE_NAME  = "MAIN_PAGE"
)

type DynamoDBTableSearchView struct {
	*DynamoDBQueryInputView
	*DynamoDBScanInputView
	RootView *tview.Flex
	MainPage tview.Primitive

	queryViewHidden bool
	scanViewHidden  bool
	pages           *tview.Pages
	app             *tview.Application
	Logger          *log.Logger
}

func NewDynamoDBTableSearchView(
	mainPage tview.Primitive,
	app *tview.Application,
	logger *log.Logger,
) *DynamoDBTableSearchView {
	var queryView = NewDynamoDBQueryInputView(app, logger)
	var floatingQuery = core.FloatingView("Query", queryView.RootView, 70, 10)
	var scanView = NewDynamoDBScanInputView(app, logger)
	var floatingScan = core.FloatingView("Scan", scanView.RootView, 70, 14)

	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage(QUERY_PAGE_NAME, floatingQuery, true, false).
		AddPage(SCAN_PAGE_NAME, floatingScan, true, false)

	var view = &DynamoDBTableSearchView{
		DynamoDBQueryInputView: queryView,
		DynamoDBScanInputView:  scanView,
		RootView:               tview.NewFlex().AddItem(pages, 0, 1, true),
		MainPage:               mainPage,

		queryViewHidden: true,
		scanViewHidden:  true,
		pages:           pages,
	}

	view.QueryCancelButton.SetSelectedFunc(func() {
		pages.HidePage(QUERY_PAGE_NAME)
		view.queryViewHidden = true
	})

	view.pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlQ:
			if view.queryViewHidden {
				pages.ShowPage(QUERY_PAGE_NAME)
				pages.HidePage(SCAN_PAGE_NAME)
				view.scanViewHidden = true
			} else {
				pages.HidePage(QUERY_PAGE_NAME)
			}
			view.queryViewHidden = !view.queryViewHidden
			return nil
		case tcell.KeyCtrlS:
			if view.scanViewHidden {
				pages.HidePage(QUERY_PAGE_NAME)
				pages.ShowPage(SCAN_PAGE_NAME)
				view.queryViewHidden = true
			} else {
				pages.HidePage(SCAN_PAGE_NAME)
			}
			view.scanViewHidden = !view.scanViewHidden
			return nil
		}
		return event
	})

	return view
}
