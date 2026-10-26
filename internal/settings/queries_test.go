package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestQueriesPath__IsNextToTheSettingsFile(t *testing.T) {
	if got := QueriesPath(filepath.Join("config", "aws-tui", "settings.json")); got != filepath.Join("config", "aws-tui", "queries.json") {
		t.Errorf("QueriesPath = %q", got)
	}
	if got := QueriesPath(""); got != "" {
		t.Errorf("QueriesPath without settings = %q, want none", got)
	}
}

func TestLoadQueries__MissingFileGivesNoQueries(t *testing.T) {
	var loaded, err = LoadQueries(filepath.Join(t.TempDir(), "queries.json"))

	if err != nil || !reflect.DeepEqual(loaded, InsightsQueries{}) {
		t.Errorf("LoadQueries = %+v, %v, want none and no error", loaded, err)
	}
}

func TestSaveQueries__ThenLoadGivesTheSameQueries(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "a", "queries.json")
	var queries = InsightsQueries{
		Last:  "fields @message",
		Saved: []SavedQuery{{Name: "errors", Query: "fields @message | filter @message like /ERROR/"}},
	}

	if err := SaveQueries(path, queries); err != nil {
		t.Fatal(err)
	}
	var loaded, err = LoadQueries(path)

	if err != nil || !reflect.DeepEqual(loaded, queries) {
		t.Errorf("LoadQueries = %+v, %v, want %+v", loaded, err, queries)
	}
}

func TestLoadQueries__UnreadableFileIsReportedWithItsPath(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "queries.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	var loaded, err = LoadQueries(path)

	if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "queries") {
		t.Errorf("error = %v, want one that names the queries and the file", err)
	}
	if !reflect.DeepEqual(loaded, InsightsQueries{}) {
		t.Errorf("loaded = %+v, want none", loaded)
	}
}
