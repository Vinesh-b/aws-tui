package core

import (
	"reflect"
	"testing"
)

func TestSortedRows__SortsByKey(t *testing.T) {
	var got = SortedRows(map[string]string{"team": "orders", "env": "prod"})

	var want = []TableRow{{"env", "prod"}, {"team", "orders"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestTagRows__SortsByKeyWhateverTheTagType(t *testing.T) {
	type tag struct{ k, v string }

	var got = TagRows([]tag{{"team", "orders"}, {"env", "prod"}}, func(t tag) (string, string) { return t.k, t.v })

	var want = []TableRow{{"env", "prod"}, {"team", "orders"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestDetailsTags__SectionIsOnlyOfTheResourceTheTagsAreOf(t *testing.T) {
	var tags = DetailsTags{rows: []TableRow{{"env", "prod"}}, key: "orders"}

	if got := tags.Section("orders"); !reflect.DeepEqual(got, []TableRow{{"Tags"}, {"env", "prod"}}) {
		t.Errorf("section = %v", got)
	}
	if got := tags.Section("billing"); got != nil {
		t.Errorf("section of another resource = %v, want none", got)
	}
	if got := (&DetailsTags{}).Section(""); got != nil {
		t.Errorf("section before any are loaded = %v, want none", got)
	}

	tags.Clear()
	if tags.Loaded("orders") || tags.Section("orders") != nil {
		t.Error("tags were kept after being cleared")
	}
}
