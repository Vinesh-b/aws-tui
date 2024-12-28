package serviceviews

import (
	"aws-tui/internal/pkg/ui/core"
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
	pkInput             *tview.InputField
	skInput             *tview.InputField
	skComparatorInput   *tview.InputField
	projectedAttributes []string
	selectedIndex       string
	tableName           string
	indexes             []string
	pkName              string
	skName              string
}

func NewDynamoDBQueryInputView(app *tview.Application, logger *log.Logger) *DynamoDBQueryInputView {
	var pkInput = tview.NewInputField().SetLabel("PK ").SetFieldWidth(0)
	var skInput = tview.NewInputField().SetLabel("SK ").SetFieldWidth(0)
	var skComparitorInput = tview.NewInputField().SetLabel("Comparator ").SetFieldWidth(8)
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
		AddItem(tview.NewBox(), 1, 0, true).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(doneButton, 0, 1, true).
				AddItem(tview.NewBox(), 1, 0, true).
				AddItem(cancelButton, 0, 1, true),
			1, 0, true,
		)

	core.InitViewTabNavigation(wrapper,
		[]core.View{
			pkInput,
			skComparitorInput,
			skInput,
			doneButton,
			cancelButton,
		},
		app,
	)

	return &DynamoDBQueryInputView{
		pkInput:           pkInput,
		skInput:           skInput,
		skComparatorInput: skComparitorInput,
		QueryDoneButton:   doneButton,
		QueryCancelButton: cancelButton,
		RootView:          wrapper,

		logger: logger,
	}
}

func (inst *DynamoDBQueryInputView) GenerateQueryExpression() expression.Expression {
	var pk = inst.pkInput.GetText()
	var sk = inst.skInput.GetText()
	var comp = inst.skComparatorInput.GetText()

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
			inst.logger.Println("Invalid operator")
		}
	}

	var expr, err = expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		inst.logger.Printf("Failed to build expression for query: %v\n", err)
	}

	return expr
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

type FilterConditionError struct {
	error
}

type FilterInputView struct {
	AttributeNameInput *tview.InputField
	AttributeTypeInput *tview.InputField
	Condition          *tview.InputField
	Value1             *tview.InputField
	Value2             *tview.InputField
	RootView           *tview.Flex

	tabNavigator *core.ViewTabNavigation
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

	var tabNavigator = core.NewViewTabNavigation(wrapper,
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

	if !inst.isConditionAllowed(attrType, cond) {
		inst.logger.Println("The attribute type does not support the given condition")
		return filterCond, FilterConditionError{}
	}

	var parsedValue1, val1Err = inst.parseValue(attrValue1, attrType)
	var parsedValue2, val2Err = inst.parseValue(attrValue2, attrType)

	if val1Err != nil {
		inst.logger.Printf("Value1 convertion failed %v\n", val1Err)
		return filterCond, FilterConditionError{}
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
	case "exists":
		filterCond = expression.Name(attrName).AttributeExists()
	case "nexists":
		filterCond = expression.Name(attrName).AttributeNotExists()
	case "contains":
		filterCond = expression.Name(attrName).Contains(parsedValue1)
	case "begins":
		filterCond = expression.Name(attrName).BeginsWith(parsedValue1.(string))
	case "between":
		if val2Err == nil {
			filterCond = expression.Name(attrName).Between(
				expression.Value(parsedValue1),
				expression.Value(parsedValue2),
			)
		} else {
			inst.logger.Printf("Value2 convertion failed %v\n", val2Err)
			return filterCond, FilterConditionError{}
		}
	}

	return filterCond, nil
}

const (
	QUERY_PAGE_NAME = "QUERY"
	MAIN_PAGE_NAME  = "MAIN_PAGE"
)

type DynamoDBTableSearchView struct {
	*DynamoDBQueryInputView
	RootView *tview.Flex
	MainPage tview.Primitive

	showSearch bool
	pages      *tview.Pages
	app        *tview.Application
	Logger     *log.Logger
}

func NewDynamoDBTableSearchView(
	mainPage tview.Primitive,
	app *tview.Application,
	logger *log.Logger,
) *DynamoDBTableSearchView {
	var queryView = NewDynamoDBQueryInputView(app, logger)
	var floatingSearch = core.FloatingView("Query", queryView.RootView, 70, 7)

	var pages = tview.NewPages().
		AddPage("MAIN_PAGE", mainPage, true, true).
		AddPage(QUERY_PAGE_NAME, floatingSearch, true, false)

	var view = &DynamoDBTableSearchView{
		DynamoDBQueryInputView: queryView,
		RootView:               tview.NewFlex().AddItem(pages, 0, 1, true),
		MainPage:               mainPage,

		showSearch: true,
		pages:      pages,
	}

	view.CancelButton.SetSelectedFunc(func() {
		pages.HidePage(QUERY_PAGE_NAME)
		view.showSearch = true
	})

	view.pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlF:
			if view.showSearch {
				pages.ShowPage(QUERY_PAGE_NAME)
			} else {
				pages.HidePage(QUERY_PAGE_NAME)
			}
			view.showSearch = !view.showSearch
			return nil
		}
		return event
	})

	return view
}
