package servicetables

import (
	"aws-tui/internal/pkg/errors"
	"aws-tui/internal/pkg/ui/core"
	"fmt"
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
	QueryDoneButton   *core.Button
	QueryCancelButton *core.Button

	appCtx              *core.AppContext
	filterView          *FilterInputView
	pkInput             *core.InputField
	skInput             *core.InputField
	skComparatorInput   *core.InputField
	projectedAttributes []string
	selectedIndex       string
	tableName           string
	indexes             []string
	pkName              string
	skName              string
	tabNavigator        *core.ViewNavigation1D
}

func NewDynamoDBQueryInputView(appContext *core.AppContext) *DynamoDBQueryInputView {
	var pkInput = core.NewInputField()
	var skInput = core.NewInputField()
	var skComparitorInput = core.NewInputField()
	var filterInputView = NewFilterInputView(appContext)
	var doneButton = core.NewButton("Done")
	var cancelButton = core.NewButton("Cancel")

	pkInput.SetLabel("PK ").SetFieldWidth(0)
	skInput.SetLabel("SK ").SetFieldWidth(0)
	skComparitorInput.SetLabel("Comparator ").SetFieldWidth(8)

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
		appContext.App,
	)

	return &DynamoDBQueryInputView{
		Flex:              wrapper,
		QueryDoneButton:   doneButton,
		QueryCancelButton: cancelButton,

		appCtx:            appContext,
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
		inst.appCtx.Logger.Printf("Failed to build expression for query: %v\n", err)
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

type FloatingDDBQueryInputView struct {
	*tview.Flex
	Input *DynamoDBQueryInputView
}

func NewFloatingDDBQueryInputView(appContext *core.AppContext) *FloatingDDBQueryInputView {
	var queryView = NewDynamoDBQueryInputView(appContext)
	return &FloatingDDBQueryInputView{
		Flex:  core.FloatingView("Query", queryView, 70, 10),
		Input: queryView,
	}
}

func (inst *FloatingDDBQueryInputView) GetLastFocusedView() tview.Primitive {
	return inst.Input.tabNavigator.GetLastFocusedView()
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
	AttributeNameInput *core.InputField
	AttributeTypeInput *core.InputField
	Condition          *core.InputField
	Value1             *core.InputField
	Value2             *core.InputField

	filterInput  FilterInput
	tabNavigator *core.ViewNavigation1D
	appCtx       *core.AppContext
}

func NewFilterInputView(appContext *core.AppContext) *FilterInputView {
	var attrNameInput = core.NewInputField()
	var attrTypeInput = core.NewInputField()
	var conditionInput = core.NewInputField()
	var value1Input = core.NewInputField()
	var value2Input = core.NewInputField()

	attrNameInput.SetLabel("Attribute ")
	attrTypeInput.SetLabel("Type ")
	conditionInput.SetLabel("Condition ")
	value1Input.SetLabel("Value ")
	value2Input.SetLabel("Value ")

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
		appContext.App,
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

		appCtx:       appContext,
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
	ScanDoneButton   *core.Button
	ScanCancelButton *core.Button

	appCtx                   *core.AppContext
	filterInputViews         [3]*FilterInputView
	projectedAttributesInput *core.InputField
	projectedAttributes      []string
	tableName                string
	indexes                  []string
	selectedIndex            string
	tabNavigator             *core.ViewNavigation1D
}

func NewDynamoDBScanInputView(appContext *core.AppContext) *DynamoDBScanInputView {
	var filterInputViews = [3]*FilterInputView{
		NewFilterInputView(appContext),
		NewFilterInputView(appContext),
		NewFilterInputView(appContext),
	}

	var separater = tview.NewBox()
	var doneButton = core.NewButton("Done")
	var cancelButton = core.NewButton("Cancel")
	var projAttrInput = core.NewInputField()
	projAttrInput.
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

	var fiv0 = filterInputViews[0].tabNavigator.GetOrderedViews()
	var fiv1 = filterInputViews[1].tabNavigator.GetOrderedViews()
	var fiv2 = filterInputViews[2].tabNavigator.GetOrderedViews()

	var orderdViews = []core.View{projAttrInput}
	orderdViews = append(orderdViews, fiv0...)
	orderdViews = append(orderdViews, fiv1...)
	orderdViews = append(orderdViews, fiv2...)
	orderdViews = append(orderdViews,
		[]core.View{
			doneButton,
			cancelButton,
		}...,
	)

	var tabNavigator = core.NewViewNavigation1D(wrapper,
		orderdViews,
		appContext.App,
	)

	return &DynamoDBScanInputView{
		Flex:             wrapper,
		ScanDoneButton:   doneButton,
		ScanCancelButton: cancelButton,

		appCtx:                   appContext,
		filterInputViews:         filterInputViews,
		projectedAttributesInput: projAttrInput,
		projectedAttributes:      nil,
		tableName:                "",
		indexes:                  nil,
		selectedIndex:            "",
		tabNavigator:             tabNavigator,
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

type FloatingDDBScanInputView struct {
	*tview.Flex
	Input *DynamoDBScanInputView
}

func NewFloatingDDBScanInputView(appContext *core.AppContext) *FloatingDDBScanInputView {
	var scanView = NewDynamoDBScanInputView(appContext)
	return &FloatingDDBScanInputView{
		Flex:  core.FloatingView("Scan", scanView, 70, 10),
		Input: scanView,
	}
}

func (inst *FloatingDDBScanInputView) GetLastFocusedView() tview.Primitive {
	return inst.Input.tabNavigator.GetLastFocusedView()
}
