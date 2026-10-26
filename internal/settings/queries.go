package settings

import "path/filepath"

const queriesFileName = "queries.json"

// SavedQuery is a Logs Insights query the user has named to use again
type SavedQuery struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

// InsightsQueries are the Logs Insights queries kept between runs
type InsightsQueries struct {
	// Last is the query that was run last, which is put back in the query form
	Last  string       `json:"last,omitempty"`
	Saved []SavedQuery `json:"saved,omitempty"`
}

// QueriesPath is where the queries are kept, next to the settings file. It is
// empty if there is no settings file, in which case they are not kept.
func QueriesPath(settingsPath string) string {
	if settingsPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(settingsPath), queriesFileName)
}

// LoadQueries reads the queries file, which gives no queries and no error if it
// does not exist, and no queries and the error if it can't be read.
func LoadQueries(path string) (InsightsQueries, error) {
	var loaded InsightsQueries
	if err := readJSON(path, "queries", &loaded); err != nil {
		return InsightsQueries{}, err
	}

	return loaded, nil
}

// SaveQueries writes the queries file, in the same way as the settings
func SaveQueries(path string, queries InsightsQueries) error {
	return writeJSON(path, "queries", queries)
}
