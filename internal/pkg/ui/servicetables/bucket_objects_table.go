package servicetables

import (
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"aws-tui/internal/pkg/awsapi"
	"aws-tui/internal/pkg/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func parentDir(s3ObjectPrefix string) string {
	var vals = strings.Split(s3ObjectPrefix, "/")
	if len(vals) <= 2 {
		return ""
	}

	var parent = strings.Join(vals[0:len(vals)-2], "/")
	return fmt.Sprintf("%s/", parent)
}

type BucketObjectsTable struct {
	*tview.Table
	ErrorMessageCallback func(text string, a ...any)
	selectedBucket      string
	selectedPrefix      string
	data                []types.Object
	logger              *log.Logger
	app                 *tview.Application
	api                 *awsapi.S3BucketsApi
}

func NewBucketObjectsTable(
	app *tview.Application,
	api *awsapi.S3BucketsApi,
	logger *log.Logger,
) *BucketObjectsTable {
	var view = &BucketObjectsTable{
		Table:               tview.NewTable(),
		ErrorMessageCallback: func(text string, a ...any) {},
		selectedBucket:      "",
		selectedPrefix:      "",
		data:                nil,
		logger:              logger,
		app:                 app,
		api:                 api,
	}
	view.populateS3ObjectsTable(nil, false)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SetSelectedFunc(func(row, column int) {})

	return view
}

func (inst *BucketObjectsTable) appendToObjectsTable(
	title string,
	data []core.TableRow,
	rowOffset int,
) {
	// Don't count the headings row in the title hence the -1
	var tableTitle = fmt.Sprintf("%s (%d)", title, len(data)+rowOffset-1)
	inst.Table.SetTitle(tableTitle)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			var text = cellData
			if colIdx == 0 {
				text, _ = filepath.Rel(inst.selectedPrefix, cellData)
			}
			inst.Table.SetCell(rowIdx+rowOffset, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}
}

func (inst *BucketObjectsTable) populateS3ObjectsTable(
	dirs []types.CommonPrefix,
	extend bool,
) {
	var tableData []core.TableRow
	for _, row := range dirs {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Prefix),
			"-",
			"-",
		})
	}

	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Key),
			fmt.Sprintf("%.1f KB", float64(aws.ToInt64(row.Size))/1024.0),
			row.LastModified.Format(time.DateTime),
		})
	}

	var title = "Objects"
	if extend {
		var rowOffset = inst.Table.GetRowCount()
		inst.appendToObjectsTable(title, tableData, rowOffset)
		return
	}

	var headings = core.TableRow{
		"Key",
		"Size",
		"LastModified",
	}

	inst.Table.
		Clear().
		SetBorders(false).
		SetFixed(1, len(headings)-1)
	inst.Table.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	if len(tableData) > 0 {
		if len(headings) != len(tableData[0]) {
			log.Panicln("Table data and headings dimensions do not match")
		}
	}

	inst.Table.SetSelectable(true, false).SetSelectedStyle(
		tcell.Style{}.Background(core.MoreContrastBackgroundColor),
	)

	var rowOffset = 0
	for col, heading := range headings {
		inst.Table.SetCell(rowOffset, col, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(core.SecondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(core.ContrastBackgroundColor),
		)
	}
	rowOffset++

	var parentDirRow = core.TableRow{"../", "-", "-"}
	for colIdx, cellData := range parentDirRow {
		inst.Table.SetCell(rowOffset, colIdx, tview.NewTableCell(cellData).
			SetReference(parentDir(inst.selectedPrefix)).
			SetAlign(tview.AlignLeft),
		)
	}
	rowOffset++

	inst.appendToObjectsTable(title, tableData, rowOffset)

	inst.Table.GetCell(0, 0).SetExpansion(1)
	inst.Table.Select(1, 0)
}

func (inst *BucketObjectsTable) RefreshObjects(force bool) {
	var dirs []types.CommonPrefix
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var objects, commonPrefixes, err = inst.api.ListObjects(
			inst.selectedBucket, inst.selectedPrefix, force,
		)

		if err != nil {
			inst.ErrorMessageCallback(err.Error())
			return
		}

		var filterObjs = objects
		// the current dir is retured in the objects list and we don't want that
		for idx, val := range objects {
			if aws.ToString(val.Key) == inst.selectedPrefix {
				filterObjs = slices.Delete(objects, idx, idx+1)
				break
			}
		}
		dirs = commonPrefixes
		inst.data = filterObjs
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateS3ObjectsTable(dirs, !force)
	})
}

func (inst *BucketObjectsTable) SetSelectedFunc(handler func(row, column int)) {
	inst.Table.SetSelectedFunc(func(row, column int) {
		var isDir = inst.selectedPrefix == "" || inst.selectedPrefix[len(inst.selectedPrefix)-1] == '/'

		if isDir {
			// Load and show files in currently selected directory.
			inst.RefreshObjects(true)
		} else {
			inst.api.DownloadFile(
				inst.selectedBucket,
				inst.selectedPrefix,
				filepath.Base(inst.selectedPrefix),
			)
		}

		handler(row, column)
	})
}

func (inst *BucketObjectsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.Table.SetSelectionChangedFunc(func(row, column int) {
		var reference = inst.Table.GetCell(row, 0).GetReference()
		if reference == nil || row < 1 {
			return
		}

		inst.selectedPrefix = reference.(string)
		handler(row, column)
	})
}

func (inst *BucketObjectsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshObjects(true)
		case core.APP_KEY_BINDINGS.NextPage:
			inst.RefreshObjects(false)
		}
		return capture(event)
	})
}

func (inst *BucketObjectsTable) SetSelectedBucket(name string) {
	inst.selectedBucket = name
	inst.selectedPrefix = ""
}

func (inst *BucketObjectsTable) GetSelectedPrefix() string {
	return inst.selectedPrefix
}
