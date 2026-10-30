package awsapi

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"aws-tui/internal/safetest"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestRequireArg(t *testing.T) {
	if err := requireArg("table name", "abc"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var err = requireArg("table name", "")
	if err == nil || err.Error() != "table name not set" {
		t.Fatalf(`expected "table name not set", got %v`, err)
	}
}

func TestLogErr(t *testing.T) {
	var buf safetest.LogCapture
	var api = baseApi{logger: buf.Logger()}

	if err := api.logErr(nil); err != nil || buf.String() != "" {
		t.Fatalf("nil error should not be logged, err=%v log=%q", err, buf.String())
	}

	var want = errors.New("boom")
	if got := api.logErr(want); got != want {
		t.Fatalf("expected the same error back, got %v", got)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Fatalf("expected error to be logged, got %q", buf.String())
	}
}

func TestCollectPaginator(t *testing.T) {
	var pages = [][]int{{1, 2}, {3}, {4, 5}}

	t.Run("collects every page", func(t *testing.T) {
		var idx = 0
		var result, err = collectPaginator(
			func() bool { return idx < len(pages) },
			func() ([]int, error) {
				idx++
				return pages[idx-1], nil
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !slices.Equal(result, []int{1, 2, 3, 4, 5}) {
			t.Fatalf("unexpected result %v", result)
		}
	})

	t.Run("no pages gives empty non-nil slice", func(t *testing.T) {
		var result, err = collectPaginator(
			func() bool { return false },
			func() ([]int, error) { return nil, nil },
		)
		if err != nil || result == nil || len(result) != 0 {
			t.Fatalf("expected empty non-nil slice, got %v, %v", result, err)
		}
	})

	t.Run("error keeps earlier pages and stops", func(t *testing.T) {
		var idx = 0
		var want = errors.New("page failed")
		var result, err = collectPaginator(
			func() bool { return true },
			func() ([]int, error) {
				idx++
				if idx == 2 {
					return nil, want
				}
				return pages[idx-1], nil
			},
		)
		if err != want {
			t.Fatalf("expected %v, got %v", want, err)
		}
		if !slices.Equal(result, []int{1, 2}) {
			t.Fatalf("expected first page only, got %v", result)
		}
		if idx != 2 {
			t.Fatalf("expected to stop after the error, fetched %d pages", idx)
		}
	})
}

func TestCollectTokenPages(t *testing.T) {
	t.Run("follows tokens until nil", func(t *testing.T) {
		var tokensSeen []*string
		var result, err = collectTokenPages(func(token *string) ([]string, *string, error) {
			tokensSeen = append(tokensSeen, token)
			switch aws.ToString(token) {
			case "":
				return []string{"a"}, aws.String("t1"), nil
			case "t1":
				return []string{"b"}, aws.String("t2"), nil
			default:
				return []string{"c"}, nil, nil
			}
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !slices.Equal(result, []string{"a", "b", "c"}) {
			t.Fatalf("unexpected result %v", result)
		}
		if len(tokensSeen) != 3 || tokensSeen[0] != nil ||
			aws.ToString(tokensSeen[1]) != "t1" || aws.ToString(tokensSeen[2]) != "t2" {
			t.Fatalf("tokens were not passed through correctly: %v", tokensSeen)
		}
	})

	t.Run("error keeps earlier pages", func(t *testing.T) {
		var want = errors.New("fetch failed")
		var result, err = collectTokenPages(func(token *string) ([]string, *string, error) {
			if token == nil {
				return []string{"a"}, aws.String("t1"), nil
			}
			return nil, nil, want
		})
		if err != want || !slices.Equal(result, []string{"a"}) {
			t.Fatalf("expected [a] and %v, got %v and %v", want, result, err)
		}
	})
}

func TestSortByKey(t *testing.T) {
	type item struct{ name *string }

	var items = []item{
		{aws.String("charlie")}, {nil}, {aws.String("alpha")}, {aws.String("bravo")},
	}
	sortByKey(items, func(i item) *string { return i.name })

	var got []string
	for _, i := range items {
		got = append(got, aws.ToString(i.name))
	}
	// nil keys sort as the empty string, i.e. first
	if !slices.Equal(got, []string{"", "alpha", "bravo", "charlie"}) {
		t.Fatalf("unexpected order %v", got)
	}
}
