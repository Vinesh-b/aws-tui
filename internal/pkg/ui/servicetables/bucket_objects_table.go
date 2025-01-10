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
	*core.SelectableTable[types.Object]
	ErrorMessageCallback func(text string, a ...any)
	selectedObject       types.Object
	selectedBucket       string
	selectedDir          string
	data                 []types.Object
	filtered             []types.Object
	dirs                 []types.CommonPrefix
	filteredDirs         []types.CommonPrefix
	logger               *log.Logger
	app                  *tview.Application
	api                  *awsapi.S3BucketsApi
}

func NewBucketObjectsTable(
	app *tview.Application,
	api *awsapi.S3BucketsApi,
	logger *log.Logger,
) *BucketObjectsTable {
	var view = &BucketObjectsTable{
		SelectableTable: core.NewSelectableTable[types.Object](
			"Objects",
			core.TableRow{
				"Key",
				"Size",
				"LastModified",
			},
			app,
		),
		ErrorMessageCallback: func(text string, a ...any) {},
		selectedObject:       types.Object{},
		selectedBucket:       "",
		selectedDir:          "",
		data:                 nil,
		filtered:             nil,
		dirs:                 nil,
		filteredDirs:         nil,
		logger:               logger,
		app:                  app,
		api:                  api,
	}

	view.populateS3ObjectsTable(nil, nil)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SetSelectedFunc(func(row, column int) {})

	view.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			var searchText = view.GetSearchText()
			view.FilterByName(searchText)
		}
	})

	view.SetSearchChangedFunc(func(text string) {
		view.FilterByName(text)
	})

	return view
}

func (inst *BucketObjectsTable) populateS3ObjectsTable(
	data []types.Object,
	dirs []types.CommonPrefix,
) {
	var tableData = []core.TableRow{
		{"../", "-", "-"},
	}
	var privateData = []types.Object{
		{Key: aws.String(parentDir(aws.ToString(inst.selectedObject.Key)))},
	}

	for _, row := range dirs {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Prefix), "-", "-",
		})
		privateData = append(privateData, types.Object{Key: row.Prefix})
	}

	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Key),
			fmt.Sprintf("%.1f KB", float64(aws.ToInt64(row.Size))/1024.0),
			row.LastModified.Format(time.DateTime),
		})
	}
	privateData = append(privateData, data...)

	var table = inst.GetTable()
	var rowCount = table.GetRowCount() - 1

	inst.SetData(tableData, privateData, 0)
	table.Select(rowCount, 0)
}

func (inst *BucketObjectsTable) FilterByName(name string) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		inst.filtered = core.FuzzySearch(
			name,
			inst.data,
			func(f types.Object) string {
				return aws.ToString(f.Key)
			},
		)
		inst.filteredDirs = core.FuzzySearch(
			name,
			inst.dirs,
			func(f types.CommonPrefix) string {
				return aws.ToString(f.Prefix)
			},
		)
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateS3ObjectsTable(inst.filtered, inst.filteredDirs)
	})
}

func (inst *BucketObjectsTable) RefreshObjects(force bool) {
	var dataLoader = core.NewUiDataLoader(inst.app, 10)

	dataLoader.AsyncLoadData(func() {
		var objects, commonPrefixes, err = inst.api.ListObjects(
			inst.selectedBucket, aws.ToString(inst.selectedObject.Key), force,
		)

		if err != nil {
			inst.ErrorMessageCallback(err.Error())
			return
		}

		var filterObjs = objects
		// the current dir is retured in the objects list and we don't want that
		for idx, val := range objects {
			if aws.ToString(val.Key) == aws.ToString(inst.selectedObject.Key) {
				filterObjs = slices.Delete(objects, idx, idx+1)
				break
			}
		}
		inst.dirs = commonPrefixes
		if !force {
			inst.data = append(inst.data, filterObjs...)
		} else {
			inst.data = filterObjs
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.populateS3ObjectsTable(inst.data, inst.dirs)
	})
}

func (inst *BucketObjectsTable) SetSelectedFunc(handler func(row, column int)) {
	inst.GetTable().SetSelectedFunc(func(row, column int) {
		var prefix = aws.ToString(inst.selectedObject.Key)
		var isDir = prefix == "" || prefix[len(prefix)-1] == '/'

		if isDir {
			// Load and show files in currently selected directory.
			inst.selectedDir = prefix
			inst.RefreshObjects(true)
		} else {
			inst.api.DownloadFile(inst.selectedBucket, prefix, filepath.Base(prefix))
		}

		handler(row, column)
	})
}

func (inst *BucketObjectsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.GetTable().SetSelectionChangedFunc(func(row, column int) {
		inst.selectedObject = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *BucketObjectsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.GetTable().SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
}

func (inst *BucketObjectsTable) GetSelectedPrefix() string {
	return aws.ToString(inst.selectedObject.Key)
}
