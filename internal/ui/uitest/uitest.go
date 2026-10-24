package uitest

import (
	"context"
	"testing"

	"aws-tui/internal/ui/core"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TableView interface {
	GetCellText(row int, column int) string
	GetTable() *tview.Table
}

func NewAppContext() *core.AppContext {
	return core.NewAppContext(
		context.Background(), tview.NewApplication(), nil, nil, &core.AppTheme{},
	)
}

func NewServiceContext[Api any]() *core.ServiceContext[Api] {
	return core.NewServiceViewContext[Api](NewAppContext(), nil)
}

func AssertDetailsRows(t *testing.T, table *core.DetailsTable, want []core.TableRow) {
	t.Helper()
	for row, wantRow := range want {
		for col, wantText := range wantRow {
			if got := table.GetCellText(row, col); got != wantText {
				t.Errorf("cell (%d,%d) = %q, want %q", row, col, got, wantText)
			}
		}
	}
	if extra := table.GetCellText(len(want), 0); extra != "" {
		t.Errorf("unexpected extra row starting with %q", extra)
	}
}

func AssertTableRows(t *testing.T, table TableView, want []core.TableRow) {
	t.Helper()
	for row, wantRow := range want {
		for col, wantText := range wantRow {
			if got := table.GetCellText(row, col); got != wantText {
				t.Errorf("cell (%d,%d) = %q, want %q", row, col, got, wantText)
			}
		}
	}
	if got := table.GetTable().GetRowCount(); got != len(want) {
		t.Errorf("row count = %d, want %d", got, len(want))
	}
}

func PressEnter(table TableView) {
	var handler = table.GetTable().InputHandler()
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
}
