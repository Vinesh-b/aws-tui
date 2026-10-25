package awsapi

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBApi interface {
	ListTables(ctx context.Context, force bool) ([]string, error)
	DescribeTable(ctx context.Context, tableName string) (*types.TableDescription, error)
	ScanTable(
		ctx context.Context,
		tableName string,
		scanExpression expression.Expression,
		indexName string,
		force bool,
	) ([]map[string]any, error)
	QueryTable(
		ctx context.Context,
		tableName string,
		queryExpression expression.Expression,
		indexName string,
		force bool,
	) ([]map[string]any, error)
	ListTags(ctx context.Context, force bool, resourceArn string) ([]types.Tag, error)
	// PutItem writes the item to the table, replacing any item that has the
	// same key.
	PutItem(ctx context.Context, tableName string, item map[string]any) error
	// UpdateItem changes only the given attributes of the item with the key:
	// those in set are written and those in remove are deleted, the rest of the
	// item is left as it is. It fails if the item no longer exists. Attribute
	// names can't have '.', '[' or ']' as these are read as paths.
	UpdateItem(
		ctx context.Context, tableName string, key map[string]any, set map[string]any, remove []string,
	) error
	// DeleteItem deletes the item with the key. It fails if there is no such
	// item, so a key that doesn't match can't look like a delete that worked.
	DeleteItem(ctx context.Context, tableName string, key map[string]any) error
}

type dynamoDBApi struct {
	baseApi
	allTables      []string
	queryPaginator *dynamodb.QueryPaginator
	scanPaginator  *dynamodb.ScanPaginator
}

func NewDynamoDBApi(
	logger *log.Logger,
) DynamoDBApi {

	return &dynamoDBApi{
		baseApi: baseApi{logger: logger},
	}
}

func (inst *dynamoDBApi) ListTables(ctx context.Context, force bool) ([]string, error) {
	var client = GetAwsApiClients().dynamodb
	if len(inst.allTables) > 0 && !force {
		return inst.allTables, nil
	}

	inst.allTables = nil
	var paginator = dynamodb.NewListTablesPaginator(
		client, &dynamodb.ListTablesInput{},
	)

	var apiErr error = nil
	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(ctx)
		if err != nil {
			inst.logger.Printf("Couldn't list tables: %v\n", err)
			apiErr = err
			break
		}

		inst.allTables = append(inst.allTables, output.TableNames...)
	}

	sort.Strings(inst.allTables)

	return inst.allTables, apiErr
}

func (inst *dynamoDBApi) DescribeTable(ctx context.Context, tableName string) (*types.TableDescription, error) {
	if err := requireArg("table name", tableName); err != nil {
		return nil, err
	}

	var client = GetAwsApiClients().dynamodb
	var output, err = client.DescribeTable(ctx,
		&dynamodb.DescribeTableInput{TableName: &tableName},
	)
	if err != nil {
		inst.logger.Printf("Failed to describe table: %v\n", err)
		return nil, err
	}
	return output.Table, nil
}

func (inst *dynamoDBApi) ScanTable(
	ctx context.Context,
	tableName string,
	scanExpression expression.Expression,
	indexName string,
	force bool,
) ([]map[string]any, error) {
	var items []map[string]any

	if err := requireArg("table name", tableName); err != nil {
		return items, err
	}

	var index *string = nil
	if len(indexName) > 0 {
		index = aws.String(indexName)
	}
	var client = GetAwsApiClients().dynamodb

	if force || inst.scanPaginator == nil {
		inst.scanPaginator = dynamodb.NewScanPaginator(client, &dynamodb.ScanInput{
			TableName:                 aws.String(tableName),
			Limit:                     aws.Int32(20),
			FilterExpression:          scanExpression.Filter(),
			ExpressionAttributeNames:  scanExpression.Names(),
			ExpressionAttributeValues: scanExpression.Values(),
			ProjectionExpression:      scanExpression.Projection(),
			IndexName:                 index,
		})
	}

	if !inst.scanPaginator.HasMorePages() {
		return items, fmt.Errorf("No more pages found")
	}

	var output, err = inst.scanPaginator.NextPage(ctx)
	if err != nil {
		inst.logger.Printf("Scan failed: %s\n", err.Error())
	} else {
		var temp []map[string]any
		attributevalue.UnmarshalListOfMaps(output.Items, &temp)
		items = append(items, temp...)
	}
	return items, err
}

func (inst *dynamoDBApi) QueryTable(
	ctx context.Context,
	tableName string,
	queryExpression expression.Expression,
	indexName string,
	force bool,
) ([]map[string]any, error) {
	var items []map[string]any

	if err := requireArg("table name", tableName); err != nil {
		return items, err
	}

	if inst.queryPaginator == nil || force {
		var index *string = nil
		if len(indexName) > 0 {
			index = aws.String(indexName)
		}
		var client = GetAwsApiClients().dynamodb
		inst.queryPaginator = dynamodb.NewQueryPaginator(client, &dynamodb.QueryInput{
			TableName:                 aws.String(tableName),
			Limit:                     aws.Int32(100),
			FilterExpression:          queryExpression.Filter(),
			ExpressionAttributeNames:  queryExpression.Names(),
			ExpressionAttributeValues: queryExpression.Values(),
			KeyConditionExpression:    queryExpression.KeyCondition(),
			ProjectionExpression:      queryExpression.Projection(),
			IndexName:                 index,
		})
	}

	if !inst.queryPaginator.HasMorePages() {
		return items, fmt.Errorf("No more pages found")
	}

	var output, err = inst.queryPaginator.NextPage(ctx)
	if err != nil {
		inst.logger.Printf("Query failed: %s\n", err.Error())
		return items, err
	}

	var temp []map[string]any
	err = attributevalue.UnmarshalListOfMaps(output.Items, &temp)
	if err != nil {
		return items, inst.logErr(err)
	}

	items = append(items, temp...)
	return items, nil
}

func (inst *dynamoDBApi) ListTags(ctx context.Context, force bool, resourceArn string) ([]types.Tag, error) {
	var result, err = collectTokenPages(func(token *string) ([]types.Tag, *string, error) {
		var output, err = GetAwsApiClients().dynamodb.ListTagsOfResource(ctx,
			&dynamodb.ListTagsOfResourceInput{
				ResourceArn: aws.String(resourceArn),
				NextToken:   token,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		return output.Tags, output.NextToken, nil
	})

	return result, inst.logErr(err)
}

func (inst *dynamoDBApi) PutItem(ctx context.Context, tableName string, item map[string]any) error {
	if err := requireArg("table name", tableName); err != nil {
		return err
	}
	if len(item) == 0 {
		return fmt.Errorf("item has no attributes")
	}

	var attributes, err = attributevalue.MarshalMap(item)
	if err != nil {
		return inst.logErr(err)
	}

	_, err = GetAwsApiClients().dynamodb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      attributes,
	})
	return inst.logErr(err)
}

func (inst *dynamoDBApi) UpdateItem(
	ctx context.Context, tableName string, key map[string]any, set map[string]any, remove []string,
) error {
	if err := requireArg("table name", tableName); err != nil {
		return err
	}
	if len(key) == 0 {
		return fmt.Errorf("item key not set")
	}
	if len(set) == 0 && len(remove) == 0 {
		return fmt.Errorf("no attributes to update")
	}

	if err := rejectPathCharacters(append(sortedKeys(set), remove...)); err != nil {
		return err
	}

	var update expression.UpdateBuilder
	for _, name := range sortedKeys(set) {
		update = update.Set(expression.Name(name), expression.Value(set[name]))
	}
	for _, name := range remove {
		update = update.Remove(expression.Name(name))
	}

	// Without the condition updating an item that was deleted would create it
	var condition = expression.AttributeExists(expression.Name(sortedKeys(key)[0]))
	var built, err = expression.NewBuilder().WithUpdate(update).WithCondition(condition).Build()
	if err != nil {
		return inst.logErr(err)
	}

	var keyAttributes map[string]types.AttributeValue
	if keyAttributes, err = attributevalue.MarshalMap(key); err != nil {
		return inst.logErr(err)
	}

	_, err = GetAwsApiClients().dynamodb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(tableName),
		Key:                       keyAttributes,
		UpdateExpression:          built.Update(),
		ConditionExpression:       built.Condition(),
		ExpressionAttributeNames:  built.Names(),
		ExpressionAttributeValues: built.Values(),
	})
	return inst.logErr(err)
}

func sortedKeys[V any](m map[string]V) []string {
	var keys = make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// rejectPathCharacters fails for names the expression builder would read as a
// path to a nested attribute.
func rejectPathCharacters(names []string) error {
	for _, name := range names {
		if strings.ContainsAny(name, ".[]") {
			return fmt.Errorf("attribute %q can't be used as its name has '.', '[' or ']'", name)
		}
	}
	return nil
}

func (inst *dynamoDBApi) DeleteItem(ctx context.Context, tableName string, key map[string]any) error {
	if err := requireArg("table name", tableName); err != nil {
		return err
	}
	if len(key) == 0 {
		return fmt.Errorf("item key not set")
	}
	if err := rejectPathCharacters(sortedKeys(key)); err != nil {
		return err
	}

	var condition = expression.AttributeExists(expression.Name(sortedKeys(key)[0]))
	var built, err = expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return inst.logErr(err)
	}

	var keyAttributes map[string]types.AttributeValue
	if keyAttributes, err = attributevalue.MarshalMap(key); err != nil {
		return inst.logErr(err)
	}

	_, err = GetAwsApiClients().dynamodb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:                aws.String(tableName),
		Key:                      keyAttributes,
		ConditionExpression:      built.Condition(),
		ExpressionAttributeNames: built.Names(),
	})
	return inst.logErr(err)
}
