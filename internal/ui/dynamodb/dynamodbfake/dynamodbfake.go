// Package dynamodbfake is an in-memory awsapi.DynamoDBApi for tests.
//
// All state is private and guarded by one mutex. Configure it with options when
// constructing, or with the Set methods afterwards. UI code calls the fake from
// background goroutines, so fields must never be touched directly.
package dynamodbfake

import (
	"context"
	"slices"
	"sync"

	"aws-tui/internal/awsapi"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ItemRef identifies an item by table and key.
type ItemRef struct {
	TableName string
	Key       map[string]any
}

type Put struct {
	TableName string
	Item      map[string]any
}

type Update struct {
	TableName string
	Key       map[string]any
	Set       map[string]any
	Remove    []string
}

type Api struct {
	awsapi.DynamoDBApi

	mtx          sync.Mutex
	tables       []string
	description  *types.TableDescription
	items        []map[string]any
	tags         []types.Tag
	err          error
	queryFunc    func(ctx context.Context, call int) ([]map[string]any, error)
	putErr       error
	updateErr    error
	deleteErr    error
	queryIndexes []string
	puts         []Put
	updates      []Update
	deleted      []ItemRef
}

type Option func(*Api)

func WithTables(tables []string) Option { return func(f *Api) { f.tables = tables } }
func WithDescription(description *types.TableDescription) Option {
	return func(f *Api) { f.description = description }
}
func WithItems(items []map[string]any) Option { return func(f *Api) { f.items = items } }
func WithTags(tags []types.Tag) Option        { return func(f *Api) { f.tags = tags } }
func WithErr(err error) Option                { return func(f *Api) { f.err = err } }
func WithPutErr(err error) Option             { return func(f *Api) { f.putErr = err } }
func WithUpdateErr(err error) Option          { return func(f *Api) { f.updateErr = err } }
func WithDeleteErr(err error) Option          { return func(f *Api) { f.deleteErr = err } }

// WithQueryFunc gives the result of each query by its call number, starting at 0.
func WithQueryFunc(fn func(ctx context.Context, call int) ([]map[string]any, error)) Option {
	return func(f *Api) { f.queryFunc = fn }
}

func New(options ...Option) *Api {
	var f = &Api{}
	for _, option := range options {
		option(f)
	}
	return f
}

func (f *Api) SetItems(items []map[string]any) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.items = items
}

func (f *Api) Items() []map[string]any {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.items
}

func (f *Api) SetDescription(description *types.TableDescription) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.description = description
}

// QueriedIndexes returns the index name of every query made so far.
func (f *Api) QueriedIndexes() []string {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.queryIndexes)
}

func (f *Api) Puts() []Put {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.puts)
}

func (f *Api) Updates() []Update {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.updates)
}

func (f *Api) Deleted() []ItemRef {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return slices.Clone(f.deleted)
}

func (f *Api) ListTables(ctx context.Context, force bool) ([]string, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.tables, f.err
}

func (f *Api) DescribeTable(ctx context.Context, tableName string) (*types.TableDescription, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.description, f.err
}

func (f *Api) ListTags(ctx context.Context, force bool, resourceArn string) ([]types.Tag, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.tags, f.err
}

func (f *Api) ScanTable(
	ctx context.Context, tableName string, scanExpression expression.Expression, indexName string, force bool,
) ([]map[string]any, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.items, f.err
}

func (f *Api) QueryTable(
	ctx context.Context, tableName string, queryExpression expression.Expression, indexName string, force bool,
) ([]map[string]any, error) {
	f.mtx.Lock()
	f.queryIndexes = append(f.queryIndexes, indexName)
	var call = len(f.queryIndexes) - 1
	var queryFunc = f.queryFunc
	var items, err = f.items, f.err
	f.mtx.Unlock()
	if queryFunc != nil {
		return queryFunc(ctx, call)
	}
	return items, err
}

func (f *Api) PutItem(ctx context.Context, tableName string, item map[string]any) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.puts = append(f.puts, Put{TableName: tableName, Item: item})
	return f.putErr
}

func (f *Api) UpdateItem(
	ctx context.Context, tableName string, key map[string]any, set map[string]any, remove []string,
) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.updates = append(f.updates, Update{TableName: tableName, Key: key, Set: set, Remove: remove})
	return f.updateErr
}

func (f *Api) DeleteItem(ctx context.Context, tableName string, key map[string]any) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.deleted = append(f.deleted, ItemRef{TableName: tableName, Key: key})
	return f.deleteErr
}
