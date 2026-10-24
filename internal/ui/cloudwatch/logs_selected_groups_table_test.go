package cloudwatch

import (
	"slices"
	"testing"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/uitest"
)

func newSelectedGroupsTable(groups ...string) *SelectedGroupsTable {
	var table = NewSelectedGroupsTable(uitest.NewServiceContext[awsapi.CloudWatchLogsApi]())
	for _, group := range groups {
		table.AddLogGroup(group)
	}
	table.populateSelectedGroupsTable()
	return table
}

func TestSelectedGroupsTable__ShowsGroupsSortedByName(t *testing.T) {
	var table = newSelectedGroupsTable("/b", "/c", "/a")

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}, {"/a"}, {"/b"}, {"/c"}})
}

func TestSelectedGroupsTable__EmptyShowsOnlyHeading(t *testing.T) {
	var table = newSelectedGroupsTable()

	uitest.AssertTableRows(t, table, []core.TableRow{{"Name"}})
}

func TestSelectedGroupsTable__AddingSameGroupTwiceKeepsOneEntry(t *testing.T) {
	var table = newSelectedGroupsTable("/a", "/a")

	if got := table.GetAllLogGroups(); !slices.Equal(got, []string{"/a"}) {
		t.Errorf("groups = %v, want [/a]", got)
	}
}

func TestSelectedGroupsTable__EmptyGroupNameIsIgnored(t *testing.T) {
	var table = newSelectedGroupsTable("")

	if got := table.GetAllLogGroups(); len(got) != 0 {
		t.Errorf("groups = %v, want none", got)
	}
}

func TestSelectedGroupsTable__RemovingGroup(t *testing.T) {
	var table = newSelectedGroupsTable("/a", "/b")

	table.RemoveLogGroup("/a")

	if got := table.GetAllLogGroups(); !slices.Equal(got, []string{"/b"}) {
		t.Errorf("groups = %v, want [/b]", got)
	}
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
