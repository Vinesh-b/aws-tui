package awsapi

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
)

// readPrefixes start the names of the operations that only read. An operation
// that starts with none of them is taken to change something, so one that is
// new to the app is refused in read-only mode until it is known to only read.
var readPrefixes = []string{
	"Get", "List", "Describe", "Head", "BatchGet", "Query", "Scan", "Filter", "Search", "Lookup",
}

// readOperations only read, though their names do not start as the others do.
// Starting and stopping a Logs Insights query changes nothing that is kept.
var readOperations = []string{"StartQuery", "StopQuery"}

// IsReadOperation is whether the operation of the SDK only reads, going by its
// name, such as ListTopics
func IsReadOperation(operation string) bool {
	if slices.Contains(readOperations, operation) {
		return true
	}
	return slices.ContainsFunc(readPrefixes, func(prefix string) bool {
		return strings.HasPrefix(operation, prefix)
	})
}

// ReadOnlyError is the error of an operation that was not sent, as the clients
// are read-only and it could change something
type ReadOnlyError struct {
	Operation string
}

func (e *ReadOnlyError) Error() string {
	if e.Operation == "" {
		return "read-only mode: the request was not sent, as what it does is not known"
	}
	return fmt.Sprintf("read-only mode: %s was not sent, as it could change something", e.Operation)
}

// WithReadOnly is the config with every operation that does not only read
// refused before it is sent. It is checked for each request of each client
// built from the config, below the service apis, so nothing the app does can
// get past it.
func WithReadOnly(cfg aws.Config) aws.Config {
	cfg.APIOptions = append(slices.Clone(cfg.APIOptions), func(stack *middleware.Stack) error {
		// After the middleware of the SDK that names the operation
		return stack.Initialize.Add(middleware.InitializeMiddlewareFunc(
			"ReadOnly",
			func(
				ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler,
			) (middleware.InitializeOutput, middleware.Metadata, error) {
				var operation = awsmiddleware.GetOperationName(ctx)
				if !IsReadOperation(operation) {
					return middleware.InitializeOutput{}, middleware.Metadata{}, &ReadOnlyError{Operation: operation}
				}
				return next.HandleInitialize(ctx, in)
			},
		), middleware.After)
	})
	return cfg
}
