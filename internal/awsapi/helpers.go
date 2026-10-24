package awsapi

import (
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// baseApi holds what every service api shares. Embed it in new service apis.
type baseApi struct {
	logger *log.Logger
}

func (b baseApi) logErr(err error) error {
	if err != nil {
		b.logger.Println(err)
	}
	return err
}

func requireArg(name string, value string) error {
	if len(value) == 0 {
		return fmt.Errorf("%s not set", name)
	}
	return nil
}

// collectPaginator drains an SDK paginator. Items fetched before an error are
// returned alongside it.
func collectPaginator[T any](hasMorePages func() bool, nextPage func() ([]T, error)) ([]T, error) {
	var result = []T{}
	for hasMorePages() {
		var items, err = nextPage()
		if err != nil {
			return result, err
		}
		result = append(result, items...)
	}
	return result, nil
}

// collectTokenPages drains an API that pages with a next token and has no SDK
// paginator. Items fetched before an error are returned alongside it.
func collectTokenPages[T any](fetch func(token *string) ([]T, *string, error)) ([]T, error) {
	var result = []T{}
	var token *string = nil
	for {
		var items, next, err = fetch(token)
		if err != nil {
			return result, err
		}
		result = append(result, items...)
		if next == nil {
			return result, nil
		}
		token = next
	}
}

// sortByKey sorts items ascending by a string pointer key.
func sortByKey[T any](items []T, key func(T) *string) {
	sort.Slice(items, func(i, j int) bool {
		return aws.ToString(key(items[i])) < aws.ToString(key(items[j]))
	})
}
