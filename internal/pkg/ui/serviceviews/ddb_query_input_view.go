package serviceviews

import (
	"aws-tui/internal/pkg/ui/core"
	"log"

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
