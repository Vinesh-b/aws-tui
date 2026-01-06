package utils

import (
	"encoding/json"
	"sort"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

type StringSet map[string]struct{}

func FilterSlice[S ~[]E, E any](s S, f func(E) bool) []E {
	var result = []E{}
	for i := range s {
		if f(s[i]) {
			result = append(result, s[i])
		}
	}
	return result
}

func ClampStringLen(input *string, maxLen int) string {
	if len(*input) < maxLen {
		return *input
	}
	return (*input)[0:maxLen-1] + "…"
}

func TryFormatToJson(text string) (string, bool) {
	var anyJson map[string]any
	var err = json.Unmarshal([]byte(text), &anyJson)
	if err != nil {
		return text, false
	}

	var jsonBytes, _ = json.MarshalIndent(anyJson, "", "  ")

	return string(jsonBytes), true
}

func FuzzySearch[T any](search string, values []T, handler func(val T) string) []T {
	if len(values) == 0 {
		return nil
	}

	if len(search) == 0 {
		return values
	}

	var names = make([]string, 0, len(values))
	for _, v := range values {
		names = append(names, handler(v))
	}

	var matches = fuzzy.RankFindFold(search, names)
	sort.Sort(matches)

	var results = make([]int, 0, len(matches))
	for _, m := range matches {
		results = append(results, m.OriginalIndex)
	}

	var found = []T{}
	for _, matchIdx := range results {
		found = append(found, values[matchIdx])
	}

	return found
}
