package s3

import (
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/gdamore/tcell/v2"
)

type BucketObjectsTable struct {
	*core.SelectableTable[types.Object]
	ErrorMessageCallback func(text string, a ...any)
	selectedObject       types.Object
	selectedBucket       string
	selectedDir          string
	data                 []types.Object
	dirs                 []types.CommonPrefix
	serviceCtx           *core.ServiceContext[awsapi.S3BucketsApi]
}

func NewBucketObjectsTable(
	serviceViewCtx *core.ServiceContext[awsapi.S3BucketsApi],
) *BucketObjectsTable {
	var view = &BucketObjectsTable{
		SelectableTable: core.NewSelectableTable[types.Object](
			"Objects",
			core.TableRow{
				"Key",
				"Size",
				"LastModified",
			},
			serviceViewCtx.AppContext,
		).SetFilterColumns(0).SetPinnedFunc(isParentDirRow),
		ErrorMessageCallback: func(text string, a ...any) {},
		selectedObject:       types.Object{},
		selectedBucket:       "",
		selectedDir:          "",
		data:                 nil,
		dirs:                 nil,
		serviceCtx:           serviceViewCtx,
	}

	view.populateS3ObjectsTable(nil, nil)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	view.SetSelectionChangedFunc(func(row, column int) {})
	view.SetSelectedFunc(func(row, column int) {})

	return view
}

// parentDirRow is the text of the row that goes up a directory
const parentDirRow = "../"

// isParentDirRow is true for the row that goes up a directory, which is kept at
// the top whatever the filter and the sort are
func isParentDirRow(row core.TableRow, _ types.Object) bool {
	return row[0] == parentDirRow
}

func (inst *BucketObjectsTable) populateS3ObjectsTable(
	data []types.Object, dirs []types.CommonPrefix,
) {
	var cleanedPrefix = filepath.Clean(inst.selectedDir)
	var parentDir = filepath.Dir(cleanedPrefix) + "/"
	if parentDir == "./" {
		parentDir = ""
	}

	var tableData = []core.TableRow{
		{parentDirRow, "-", "-"},
	}
	var privateData = []types.Object{
		{Key: aws.String(parentDir)},
	}

	for _, row := range dirs {
		tableData = append(tableData, core.TableRow{
			filepath.Base(aws.ToString(row.Prefix)) + "/", "-", "-",
		})
		privateData = append(privateData, types.Object{Key: row.Prefix})
	}

	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			filepath.Base(aws.ToString(row.Key)),
			fmt.Sprintf("%.1f KB", float64(aws.ToInt64(row.Size))/1024.0),
			aws.ToTime(row.LastModified).Format(time.DateTime),
		})
	}
	privateData = append(privateData, data...)

	inst.SetTitleExtra(cleanedPrefix)
	inst.SetData(tableData, privateData, 0)
	inst.GetTable().ScrollToBeginning()
}

func (inst *BucketObjectsTable) RefreshObjects(force bool) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var bucket, dir = inst.selectedBucket, inst.selectedDir
	var objects []types.Object
	var commonPrefixes []types.CommonPrefix
	var loadFailed = false

	dataLoader.AsyncLoadData(func() {
		var err error
		objects, commonPrefixes, err = inst.serviceCtx.Api.ListObjects(
			inst.serviceCtx.Ctx, bucket, dir, force,
		)

		if err != nil {
			loadFailed = true
			inst.ErrorMessageCallback("%s", err.Error())
			return
		}

		// the current dir is retured in the objects list and we don't want that
		for idx, val := range objects {
			if aws.ToString(val.Key) == dir {
				objects = slices.Delete(objects, idx, idx+1)
				break
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		if !loadFailed {
			if !force {
				inst.dirs = append(inst.dirs, commonPrefixes...)
				inst.data = append(inst.data, objects...)
			} else {
				inst.dirs = commonPrefixes
				inst.data = objects
			}
		}
		inst.populateS3ObjectsTable(inst.data, inst.dirs)
	})
}

func (inst *BucketObjectsTable) SetSelectedFunc(handler func(row, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		var prefix = aws.ToString(inst.selectedObject.Key)
		var isDir = prefix == "" || prefix[len(prefix)-1] == '/'

		if isDir {
			// Load and show files in currently selected directory.
			inst.selectedDir = prefix
			inst.ClearFilter()
			inst.RefreshObjects(true)
		} else {
			//inst.serviceCtx.Api.DownloadFile(inst.serviceCtx.Ctx, inst.selectedBucket, prefix, filepath.Base(prefix))
		}

		handler(row, column)
	})
}

func (inst *BucketObjectsTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		inst.selectedObject = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *BucketObjectsTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SetLoadMoreFunc(func() { inst.RefreshObjects(false) })
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshObjects(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *BucketObjectsTable) SetSelectedBucket(name string) {
	inst.selectedBucket = name
	inst.selectedDir = ""
	inst.selectedObject = types.Object{}
	inst.ClearFilter()
}

// GetSelectedBucket is the bucket the listed objects belong to. It does not
// follow the cursor of the bucket list.
func (inst *BucketObjectsTable) GetSelectedBucket() string {
	return inst.selectedBucket
}

func (inst *BucketObjectsTable) GetSelectedPrefix() string {
	return aws.ToString(inst.selectedObject.Key)
}
