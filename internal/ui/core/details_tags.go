package core

import (
	"maps"
	"slices"

	"github.com/rivo/tview"
)

// SortedRows are the keys and values of the map sorted by key
func SortedRows(values map[string]string) []TableRow {
	var rows []TableRow
	for _, key := range slices.Sorted(maps.Keys(values)) {
		rows = append(rows, TableRow{key, values[key]})
	}
	return rows
}

// TagRows are the tags sorted by key, with the key and value of each taken by
// keyValue as services have a tag type of their own.
func TagRows[T any](tags []T, keyValue func(T) (string, string)) []TableRow {
	var rows = make([]TableRow, 0, len(tags))
	for _, tag := range tags {
		var key, value = keyValue(tag)
		rows = append(rows, TableRow{key, value})
	}
	slices.SortStableFunc(rows, func(a, b TableRow) int {
		if a[0] < b[0] {
			return -1
		}
		if a[0] > b[0] {
			return 1
		}
		return 0
	})
	return rows
}

// DetailsTags are the tags of a resource, shown as a section of its details.
// They are loaded after the rest of the details, once for each resource.
type DetailsTags struct {
	rows []TableRow
	// key is the resource the rows are of, which is empty before any are loaded
	key string
}

// Section is the rows of the tags section if the tags are those of the
// resource with the key, and no rows otherwise.
func (inst *DetailsTags) Section(key string) []TableRow {
	if len(key) == 0 || inst.key != key {
		return nil
	}
	return DetailsSection("Tags", inst.rows...)
}

// Loaded says if the tags of the resource with the key are here
func (inst *DetailsTags) Loaded(key string) bool {
	return len(key) > 0 && inst.key == key
}

// Clear drops the tags, e.g. after the resource was deleted
func (inst *DetailsTags) Clear() {
	inst.rows = nil
	inst.key = ""
}

// Load gets the tags of the resource with the key in the background, unless
// they are here already, and then calls redraw. The tags are kept only if
// current still gives the key, as the selection may have moved on while they
// loaded. Failing to load them is reported with onError.
func (inst *DetailsTags) Load(
	app *tview.Application,
	view View,
	key string,
	current func() string,
	fetch func() ([]TableRow, error),
	onError func(text string, a ...any),
	redraw func(),
) {
	if len(key) == 0 || inst.Loaded(key) {
		return
	}

	var dataLoader = NewUiDataLoader(app, 10)
	var rows []TableRow
	var loadFailed = false

	dataLoader.AsyncLoadData(func() {
		var err error
		rows, err = fetch()
		if err != nil {
			loadFailed = true
			onError("%s", err.Error())
		}
	})

	dataLoader.AsyncUpdateView(view, func() {
		if !loadFailed && current() == key {
			inst.rows = rows
			inst.key = key
		}
		// The loader puts back the title from before it loaded, so the table
		// is drawn again either way
		redraw()
	})
}
