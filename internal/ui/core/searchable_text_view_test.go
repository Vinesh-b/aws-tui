package core

import (
	"regexp"
	"testing"
)

func TestMatchLines__MatchesCountRuneForEveryMatch(t *testing.T) {
	var texts = []string{
		"",
		"foo",
		"foo\nbar foo\n\nfoo",
		"héllo\nwörld foo\n日本語 foo\nfoo foo",
		"\nfoo\n",
	}
	var expr = regexp.MustCompile("foo")

	for _, text := range texts {
		var positions = expr.FindAllStringIndex(text, -1)
		var lines = matchLines(text, positions)

		if len(lines) != len(positions) {
			t.Fatalf("%q: got %d lines for %d matches", text, len(lines), len(positions))
		}
		for i, pos := range positions {
			var want = countRune(text, '\n', pos[0])
			if lines[i] != want {
				t.Errorf("%q match %d at %d: got %d, want %d", text, i, pos[0], lines[i], want)
			}
		}
	}
}
