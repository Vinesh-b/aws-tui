package servicetables

import (
	"aws-tui/internal/pkg/errors"
	"aws-tui/internal/pkg/ui/core"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DynamoDBDataType int

const (
	Number DynamoDBDataType = iota
	String
	Boolean
	Binary
	Null
)

func DynamoDBDataTypeString(val string) (DynamoDBDataType, error) {
	var stringEnumMap = map[string]DynamoDBDataType{
		"number": Number,
		"string": String,
		"bool":   Boolean,
		"binary": Binary,
		"null":   Null,
	}

	var cond, found = stringEnumMap[val]

	if !found {
		return -1, errors.NewDDBViewError(
			errors.InvalidOption,
			"Invalid data type set",
		)
	}

	return cond, nil
}

type DynamoDBCondition int

const (
	Equals DynamoDBCondition = iota
	NotEquals
	LessThan
	GreaterThan
	LessThanOrEqual
	GreaterThanOrEqual
	Exists
	NotExists
	Between
	Contains
	BeginsWith
)

func DynamoDBConditionFromString(val string) (DynamoDBCondition, error) {
	var stringEnumMap = map[string]DynamoDBCondition{
		"eq":       Equals,
		"neq":      NotEquals,
		"lt":       LessThan,
		"gt":       GreaterThan,
		"lte":      LessThanOrEqual,
		"gte":      GreaterThanOrEqual,
		"exists":   Exists,
		"nexists":  NotExists,
		"between":  Between,
		"contains": Contains,
		"begins":   BeginsWith,
	}

	var cond, found = stringEnumMap[val]

	if !found {
		return -1, errors.NewDDBViewError(
			errors.InvalidOption,
			"Invalid condition set",
		)
	}

	return cond, nil
}

func DynamoDBTypeOpMap() map[DynamoDBDataType][]DynamoDBCondition {
	return map[DynamoDBDataType][]DynamoDBCondition{
		String: {
			Equals, NotEquals, LessThan, GreaterThan, LessThanOrEqual, GreaterThanOrEqual,
			Exists, NotExists, Between, Contains, BeginsWith,
		},
		Binary: {
			Equals, NotEquals, LessThan, GreaterThan, LessThanOrEqual, GreaterThanOrEqual,
			Exists, NotExists, Between, Contains, BeginsWith,
		},
		Number: {
			Equals, NotEquals, LessThan, GreaterThan, LessThanOrEqual, GreaterThanOrEqual,
			Exists, NotExists, Between,
		},
		Boolean: {
			Equals, NotEquals, Exists, NotExists,
		},
		Null: {
			Exists, NotExists,
		},
	}
}

type DynamoDBQueryInputView struct {
	*tview.Flex
	QueryDoneButton   *tview.Button
	QueryCancelButton *tview.Button

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
	tabNavigator        *core.ViewNavigation1D
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
		AddItem(filterInputView, 2, 0, true).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(doneButton, 0, 1, true).
				AddItem(tview.NewBox(), 1, 0, true).
				AddItem(cancelButton, 0, 1, true),
			1, 0, true,
		)

	var tabNavigator = core.NewViewNavigation1D(wrapper,
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
		Flex:              wrapper,
		QueryDoneButton:   doneButton,
		QueryCancelButton: cancelButton,

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

	var filterCond, _ = inst.filterView.GenerateFilterCondition()

	if filterCond.IsSet() {
		exprBuilder = exprBuilder.WithFilter(filterCond)
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

type FilterInput struct {
	AttributeName string
	AttributeType DynamoDBDataType
	Condition     DynamoDBCondition
	Value1        any
	Value2        any
}

type FilterInputView struct {
	*tview.Flex
	AttributeNameInput *tview.InputField
	AttributeTypeInput *tview.InputField
	Condition          *tview.InputField
	Value1             *tview.InputField
	Value2             *tview.InputField

	filterInput  FilterInput
	tabNavigator *core.ViewNavigation1D
	logger       *log.Logger
}

func NewFilterInputView(app *tview.Application, logger *log.Logger) *FilterInputView {
	var attrNameInput = tview.NewInputField().SetLabel("Attribute ")
	var attrTypeInput = tview.NewInputField().SetLabel("Type ")
	var conditionInput = tview.NewInputField().SetLabel("Condition ")
	var value1Input = tview.NewInputField().SetLabel("Value ")
	var value2Input = tview.NewInputField().SetLabel("Value ")

	var spacerView = tview.NewBox()
	var line2View = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(spacerView, 0, 1, false)
	var line1View = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(attrNameInput, 0, 1, true).
		AddItem(spacerView, 1, 0, false).
		AddItem(attrTypeInput, 13, 0, true).
		AddItem(spacerView, 1, 0, false).
		AddItem(conditionInput, 18, 0, true)

	var wrapper = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(line1View, 1, 0, true).
		AddItem(line2View, 1, 0, true)

	var tabNavigator = core.NewViewNavigation1D(wrapper,
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
			line2View.AddItem(spacerView, 0, 1, false)
		case "between":
			line2View.
				AddItem(value1Input, 0, 1, false).
				AddItem(spacerView, 1, 0, false).
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
		Flex:               wrapper,
		AttributeNameInput: attrNameInput,
		AttributeTypeInput: attrTypeInput,
		Condition:          conditionInput,
		Value1:             value1Input,
		Value2:             value2Input,

		logger:       logger,
		tabNavigator: tabNavigator,
	}
}

func (inst *FilterInputView) isConditionAllowed(attrType DynamoDBDataType, condition DynamoDBCondition) bool {
	var typeOpMapping = DynamoDBTypeOpMap()
	var conditions, _ = typeOpMapping[attrType]
	var res = slices.Index(conditions, condition)

	return res != -1
}

func (inst *FilterInputView) parseValue(value string, dataType DynamoDBDataType) (any, error) {
	var parsedValue any
	var err error = nil
	switch dataType {
	case Boolean:
		parsedValue, err = strconv.ParseBool(value)
	case Number:
		parsedValue, err = strconv.ParseFloat(value, 64)
	default:
		parsedValue = value
	}

	return parsedValue, err
}

func (inst *FilterInputView) parseInputFields() (FilterInput, error) {
	var attrName = strings.TrimSpace(inst.AttributeNameInput.GetText())
	var attrType = strings.ToLower(inst.AttributeTypeInput.GetText())
	var attrValue1 = strings.TrimSpace(inst.Value1.GetText())
	var attrValue2 = strings.TrimSpace(inst.Value2.GetText())
	var cond = strings.TrimSpace(strings.ToLower(inst.Condition.GetText()))

	var err error = nil
	var filterInput = FilterInput{}

	if filterInput.AttributeName = attrName; len(attrName) == 0 {
		return filterInput, errors.NewDDBViewError(
			errors.MissingRequiredInput,
			"Attribute name not set",
		)
	}

	if filterInput.AttributeType, err = DynamoDBDataTypeString(attrType); err != nil {
		return filterInput, err
	}

	if filterInput.Condition, err = DynamoDBConditionFromString(cond); err != nil {
		return filterInput, err
	}

	if !inst.isConditionAllowed(filterInput.AttributeType, filterInput.Condition) {
		return filterInput, errors.NewDDBViewError(
			errors.InvalidOption,
			"Attribute type does not support given condition",
		)
	}

	if filterInput.Condition != Exists && filterInput.Condition != NotExists {
		if len(attrValue1) == 0 {
			return filterInput, errors.NewDDBViewError(
				errors.MissingRequiredInput,
				"Attribute value not set",
			)
		}
		if filterInput.Value1, err = inst.parseValue(attrValue1, filterInput.AttributeType); err != nil {
			return filterInput, errors.NewDDBViewError(
				errors.InvalidOption,
				fmt.Sprintf("Value 1 conversion failed %v", err),
			)
		}
	}

	if filterInput.Condition == Between {
		if len(attrValue2) == 0 {
			return filterInput, errors.NewDDBViewError(
				errors.MissingRequiredInput,
				"Second attribute value not set",
			)
		}
		if filterInput.Value2, err = inst.parseValue(attrValue2, filterInput.AttributeType); err != nil {
			return filterInput, errors.NewDDBViewError(
				errors.InvalidOption,
				fmt.Sprintf("Value 2 conversion failed %v", err),
			)
		}
	}

	return filterInput, nil
}

func (inst *FilterInputView) GenerateFilterCondition() (expression.ConditionBuilder, error) {
	var filterInput, err = inst.parseInputFields()
	var filterCond = expression.ConditionBuilder{}

	if err != nil {
		return filterCond, err
	}

	var exprName = expression.Name(filterInput.AttributeName)
	var exprVal1 = expression.Value(filterInput.Value1)
	var exprVal2 = expression.Value(filterInput.Value2)

	switch filterInput.Condition {
	case Equals:
		filterCond = exprName.Equal(exprVal1)
	case NotEquals:
		filterCond = exprName.NotEqual(exprVal1)
	case LessThan:
		filterCond = exprName.LessThan(exprVal1)
	case LessThanOrEqual:
		filterCond = exprName.LessThanEqual(exprVal1)
	case GreaterThan:
		filterCond = exprName.GreaterThan(exprVal1)
	case GreaterThanOrEqual:
		filterCond = exprName.GreaterThanEqual(exprVal1)
	case Contains:
		filterCond = exprName.Contains(filterInput.Value1)
	case BeginsWith:
		filterCond = exprName.BeginsWith(filterInput.Value1.(string))
	case Exists:
		filterCond = exprName.AttributeExists()
		return filterCond, nil
	case NotExists:
		filterCond = exprName.AttributeNotExists()
		return filterCond, nil
	case Between:
		filterCond = exprName.Between(exprVal1, exprVal2)
	default:
		return filterCond, errors.NewDDBViewError(
			errors.InvalidOption,
			"Unsupported condition given",
		)
	}

	return filterCond, nil
}

type DynamoDBScanInputView struct {
	*tview.Flex
	ScanDoneButton   *tview.Button
	ScanCancelButton *tview.Button

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
		AddItem(separater, 0, 1, false)

	for _, view := range filterInputViews {
		wrapper.
			AddItem(view, 2, 0, true).
			AddItem(separater, 0, 1, false)
	}

	wrapper.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(doneButton, 0, 1, true).
			AddItem(separater, 1, 0, false).
			AddItem(cancelButton, 0, 1, true),
		1, 0, true,
	)

	return &DynamoDBScanInputView{
		Flex:             wrapper,
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
	var filterCond, filtErr = inst.filterInputViews[0].GenerateFilterCondition()
	if filtErr != nil {
		return expression.Expression{}, filtErr
	}

	for _, filterView := range inst.filterInputViews[1:] {
		var cond, err = filterView.GenerateFilterCondition()
		if err == nil {
			filterCond = filterCond.And(cond)
		}
	}

	var exprBuilder = expression.NewBuilder()

	if filterCond.IsSet() {
		exprBuilder = exprBuilder.WithFilter(filterCond)
	} else {
		return expression.Expression{}, errors.NewDDBViewError(
			errors.InvalidFilterCondition,
			"Filter Condition not set",
		)
	}

	var projectionText = strings.TrimSpace(inst.projectedAttributesInput.GetText())
	if atterStrings := strings.Split(projectionText, ","); len(atterStrings[0]) > 0 {
		var names = []expression.NameBuilder{}
		for _, attr := range atterStrings {
			names = append(names, expression.Name(attr))
		}
		var projection = expression.NamesList(names[0], names[1:]...)
		exprBuilder = exprBuilder.WithProjection(projection)
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
	*tview.Flex
	*DynamoDBQueryInputView
	*DynamoDBScanInputView
	MainPage tview.Primitive

	queryViewHidden bool
	scanViewHidden  bool
	pages           *tview.Pages
	app             *tview.Application
	logger          *log.Logger
}

func NewDynamoDBTableSearchView(
	mainPage tview.Primitive,
	app *tview.Application,
	logger *log.Logger,
) *DynamoDBTableSearchView {
	var queryView = NewDynamoDBQueryInputView(app, logger)
	var floatingQuery = core.FloatingView("Query", queryView, 70, 10)
	var scanView = NewDynamoDBScanInputView(app, logger)
	var floatingScan = core.FloatingView("Scan", scanView, 70, 14)

	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage(QUERY_PAGE_NAME, floatingQuery, true, false).
		AddPage(SCAN_PAGE_NAME, floatingScan, true, false)

	var view = &DynamoDBTableSearchView{
		Flex:                   tview.NewFlex().AddItem(pages, 0, 1, true),
		DynamoDBQueryInputView: queryView,
		DynamoDBScanInputView:  scanView,
		MainPage:               mainPage,

		queryViewHidden: true,
		scanViewHidden:  true,
		pages:           pages,
		app:             app,
	}

	view.QueryCancelButton.SetSelectedFunc(func() {
		pages.HidePage(QUERY_PAGE_NAME)
		view.queryViewHidden = true
	})

	view.pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Escape:
			pages.HidePage(QUERY_PAGE_NAME)
			pages.HidePage(SCAN_PAGE_NAME)
			view.queryViewHidden = true
			view.scanViewHidden = true
		case core.APP_KEY_BINDINGS.TableQuery:
			if view.queryViewHidden {
				pages.ShowPage(QUERY_PAGE_NAME)
				pages.HidePage(SCAN_PAGE_NAME)
				view.app.SetFocus(queryView)
				view.scanViewHidden = true
			} else {
				pages.HidePage(QUERY_PAGE_NAME)
			}
			view.queryViewHidden = !view.queryViewHidden
			return nil
		case core.APP_KEY_BINDINGS.TableScan:
			if view.scanViewHidden {
				pages.HidePage(QUERY_PAGE_NAME)
				pages.ShowPage(SCAN_PAGE_NAME)
				view.app.SetFocus(scanView)
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
