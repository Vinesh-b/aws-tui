package serviceviews

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

func populateS3BucketsTable(table *tview.Table, data map[string]types.Bucket) {
	var tableData []core.TableRow
	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			*row.Name,
			row.CreationDate.Format(time.DateTime),
		})
	}

	core.InitSelectableTable(table, "Buckets",
		core.TableRow{
			"Name",
			"CreationDate",
		},
		tableData,
		[]int{0, 1},
	)
	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
}

func parentDir(s3ObjectPrefix string) string {
	var vals = strings.Split(s3ObjectPrefix, "/")
	if len(vals) <= 2 {
		return ""
	}

	var parent = strings.Join(vals[0:len(vals)-2], "/")
	return fmt.Sprintf("%s/", parent)
}

func appendToObjectsTable(
	table *tview.Table,
	title string,
	data []core.TableRow,
	rowOffset int,
	prefix string,
) {
	// Don't count the headings row in the title hence the -1
	var tableTitle = fmt.Sprintf("%s (%d)", title, len(data)+rowOffset-1)
	table.SetTitle(tableTitle)

	for rowIdx, rowData := range data {
		for colIdx, cellData := range rowData {
			var text = cellData
			if colIdx == 0 {
				text, _ = filepath.Rel(prefix, cellData)
			}
			table.SetCell(rowIdx+rowOffset, colIdx, tview.NewTableCell(text).
				SetReference(cellData).
				SetAlign(tview.AlignLeft),
			)
		}
	}
}

func populateS3ObjectsTable(
	table *tview.Table,
	data []types.Object,
	dirs []types.CommonPrefix,
	prefix string,
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

	for _, row := range data {
		tableData = append(tableData, core.TableRow{
			aws.ToString(row.Key),
			fmt.Sprintf("%.1f KB", float64(aws.ToInt64(row.Size))/1024.0),
			row.LastModified.Format(time.DateTime),
		})
	}

	var title = "Objects"
	if extend {
		var rowOffset = table.GetRowCount()
		appendToObjectsTable(table, title, tableData, rowOffset, prefix)
		return
	}

	var headings = core.TableRow{
		"Key",
		"Size",
		"LastModified",
	}

	table.
		Clear().
		SetBorders(false).
		SetFixed(1, len(headings)-1)
	table.
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(true)

	if len(tableData) > 0 {
		if len(headings) != len(tableData[0]) {
			log.Panicln("Table data and headings dimensions do not match")
		}
	}

	table.SetSelectable(true, false).SetSelectedStyle(
		tcell.Style{}.Background(core.MoreContrastBackgroundColor),
	)

	var rowOffset = 0
	for col, heading := range headings {
		table.SetCell(rowOffset, col, tview.NewTableCell(heading).
			SetAlign(tview.AlignLeft).
			SetTextColor(core.SecondaryTextColor).
			SetSelectable(false).
			SetBackgroundColor(core.ContrastBackgroundColor),
		)
	}
	rowOffset++

	var parentDirRow = core.TableRow{"../", "-", "-"}
	for colIdx, cellData := range parentDirRow {
		table.SetCell(rowOffset, colIdx, tview.NewTableCell(cellData).
			SetReference(parentDir(prefix)).
			SetAlign(tview.AlignLeft),
		)
	}
	rowOffset++

	appendToObjectsTable(table, title, tableData, rowOffset, prefix)

	table.GetCell(0, 0).SetExpansion(1)
	table.Select(0, 0)
}

type S3BucketsDetailsView struct {
	BucketsTable      *tview.Table
	ObjectsTable      *tview.Table
	ObjectInput       *tview.InputField
	RootView          *tview.Flex
	searchabelView    *core.SearchableView
	selctedBucketName string
	currentPrefix     string
	app               *tview.Application
	api               *awsapi.S3BucketsApi
}

func NewS3bucketsDetailsView(
	app *tview.Application,
	api *awsapi.S3BucketsApi,
	logger *log.Logger,
) *S3BucketsDetailsView {
	var bucketsTable = tview.NewTable()
	populateS3BucketsTable(bucketsTable, make(map[string]types.Bucket, 0))

	var objectsTable = tview.NewTable()
	populateS3ObjectsTable(objectsTable, nil, nil, "", false)

	var objectKeyInputField = core.CreateSearchInput("Object Path")

	const objectsTableSize = 4000
	const bucketsTableSize = 3000

	var mainPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(objectsTable, 0, objectsTableSize, false).
		AddItem(tview.NewFlex().
			AddItem(objectKeyInputField, 0, 1, false),
			3, 0, false,
		).
		AddItem(bucketsTable, 0, bucketsTableSize, true)

	var searchabelView = core.NewSearchableView(app, logger, mainPage)
	var serviceView = core.NewServiceView(app, logger)

	serviceView.RootView = searchabelView.RootView

	serviceView.SetResizableViews(
		objectsTable, bucketsTable,
		objectsTableSize, bucketsTableSize,
	)

	serviceView.InitViewNavigation(
		[]core.View{
			bucketsTable,
			objectKeyInputField,
			objectsTable,
		},
	)

	return &S3BucketsDetailsView{
		BucketsTable:      bucketsTable,
		ObjectsTable:      objectsTable,
		ObjectInput:       objectKeyInputField,
		RootView:          serviceView.RootView,
		searchabelView:    searchabelView,
		selctedBucketName: "",
		currentPrefix:     "",
		app:               app,
		api:               api,
	}
}

func (inst *S3BucketsDetailsView) RefreshBuckets(search string, force bool) {
	var data map[string]types.Bucket
	var resultChannel = make(chan struct{})

	go func() {
		if len(search) > 0 {
			data = inst.api.FilterByName(search)
		} else {
			data = inst.api.ListBuckets(force)
		}
		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.BucketsTable.Box, resultChannel, func() {
		populateS3BucketsTable(inst.BucketsTable, data)
	})
}

func (inst *S3BucketsDetailsView) RefreshObjects(bucketName string, prefix string, force bool) {
	var data []types.Object
	var dirs []types.CommonPrefix
	var resultChannel = make(chan struct{})

	go func() {
		var objects, commonPrefixes = inst.api.ListObjects(bucketName, prefix, force)
		var filterObjs = objects
		// the current dir is retured in the objects list and we don't want that
		for idx, val := range objects {
			if aws.ToString(val.Key) == prefix {
				filterObjs = slices.Delete(objects, idx, idx+1)
				break
			}
		}
		dirs = commonPrefixes
		data = filterObjs

		resultChannel <- struct{}{}
	}()

	go core.LoadData(inst.app, inst.ObjectsTable.Box, resultChannel, func() {
		populateS3ObjectsTable(inst.ObjectsTable, data, dirs, prefix, !force)
	})
}

func (inst *S3BucketsDetailsView) InitInputCapture() {
	inst.searchabelView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshBuckets(inst.searchabelView.GetText(), false)
		case tcell.KeyEsc:
			inst.searchabelView.SetText("")
		default:
			return
		}
	})

	inst.ObjectInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			inst.RefreshObjects(inst.selctedBucketName, inst.ObjectInput.GetText(), true)
		}

	})

	inst.BucketsTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selctedBucketName = inst.BucketsTable.GetCell(row, 0).Text
		inst.RefreshObjects(inst.BucketsTable.GetCell(row, 0).Text, "", true)
		inst.app.SetFocus(inst.ObjectsTable)
	})

	inst.BucketsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.RefreshBuckets(inst.searchabelView.GetText(), true)
		}
		return event
	})

	inst.ObjectsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			inst.currentPrefix = ""
			inst.RefreshObjects(inst.selctedBucketName, "", true)
		case tcell.KeyCtrlN:
			inst.RefreshObjects(inst.selctedBucketName, inst.currentPrefix, false)
		}
		return event
	})

	inst.ObjectsTable.SetSelectedFunc(func(row, column int) {
		var reference = inst.ObjectsTable.GetCell(row, 0).GetReference()
		if reference == nil || inst.selctedBucketName == "" {
			return
		}

		inst.currentPrefix = reference.(string)
		var isDir = inst.currentPrefix == "" || inst.currentPrefix[len(inst.currentPrefix)-1] == '/'

		if isDir {
			// Load and show files in this directory.
			inst.RefreshObjects(inst.selctedBucketName, inst.currentPrefix, true)
		} else {
			inst.api.DownloadFile(inst.selctedBucketName, inst.currentPrefix, filepath.Base(inst.currentPrefix))
		}
	})
}

func CreateS3bucketsHomeView(
	app *tview.Application,
	config aws.Config,
	logger *log.Logger,
) tview.Primitive {
	core.ChangeColourScheme(tcell.NewHexColor(0x005500))
	defer core.ResetGlobalStyle()

	var (
		api           = awsapi.NewS3BucketsApi(config, logger)
		s3DetailsView = NewS3bucketsDetailsView(app, api, logger)
	)

	var pages = tview.NewPages().
		AddAndSwitchToPage("S3Buckets", s3DetailsView.RootView, true)

	var orderedPages = []string{
		"S3Buckets",
	}

	var serviceRootView = core.NewServiceRootView(
		app, string(S3BUCKETS), pages, orderedPages).Init()

	s3DetailsView.InitInputCapture()

	return serviceRootView.RootView
}
