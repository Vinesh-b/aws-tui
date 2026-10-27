package cloudwatch

import (
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func newSelectedGroupsTable(groups ...string) *SelectedGroupsTable {
	var marks = NewLogGroupMarks()
	for _, group := range groups {
		marks.Add(group)
	}
	return NewSelectedGroupsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi](), marks)
}

func TestSelectedGroupsTable__ShowsGroupsSortedByName(t *testing.T) {
	var table = newSelectedGroupsTable("/b", "/c", "/a")

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}, {"/a"}, {"/b"}, {"/c"}})
}

func TestSelectedGroupsTable__EmptyShowsOnlyHeading(t *testing.T) {
	var table = newSelectedGroupsTable()

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}})
}

func TestSelectedGroupsTable__MarkingSameGroupTwiceKeepsOneEntry(t *testing.T) {
	var table = newSelectedGroupsTable("/a", "/a")

	if got := table.marks.All(); !slices.Equal(got, []string{"/a"}) {
		t.Errorf("groups = %v, want [/a]", got)
	}
}

func TestSelectedGroupsTable__EmptyGroupNameIsIgnored(t *testing.T) {
	var table = newSelectedGroupsTable("")

	if got := table.marks.All(); len(got) != 0 {
		t.Errorf("groups = %v, want none", got)
	}
}

func TestSelectedGroupsTable__MarkingAGroupListsIt(t *testing.T) {
	var table = newSelectedGroupsTable("/b")

	table.marks.Add("/a")

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}, {"/a"}, {"/b"}})
}

func TestSelectedGroupsTable__UnmarkKeyUnmarksTheSelectedGroup(t *testing.T) {
	var table = newSelectedGroupsTable("/a", "/b")
	table.Select(1, 0)

	uitest.PressTableRune(table, 'u')

	if got := table.marks.All(); !slices.Equal(got, []string{"/b"}) {
		t.Errorf("groups = %v, want [/b]", got)
	}
	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}, {"/b"}})
}

func TestSelectedGroupsTable__ResetKeyUnmarksAllGroups(t *testing.T) {
	var table = newSelectedGroupsTable("/a", "/b")

	uitest.PressTableRune(table, core.APP_KEY_BINDINGS.Reset)

	if got := table.marks.Len(); got != 0 {
		t.Errorf("groups = %v, want none", table.marks.All())
	}
	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}})
}

func TestSelectedGroupsTable__FirstGroupIsSelectedAfterPopulate(t *testing.T) {
	var table = newSelectedGroupsTable("/b", "/a")

	if got := table.GetSelectedLogGroup(); got != "/a" {
		t.Errorf("selected group = %q, want %q", got, "/a")
	}
}

func TestSelectedGroupsTable__SelectingRowTracksSelectedGroup(t *testing.T) {
	var table = newSelectedGroupsTable("/a", "/b")

	table.Select(2, 0)

	if got := table.GetSelectedLogGroup(); got != "/b" {
		t.Errorf("selected group = %q, want %q", got, "/b")
	}
}
