package awsapi

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBApi struct {
	logger         *log.Logger
	config         aws.Config
	client         *dynamodb.Client
	allTables      []string
	queryPaginator *dynamodb.QueryPaginator
	scanPaginator  *dynamodb.ScanPaginator
}

func NewDynamoDBApi(
	config aws.Config,
	logger *log.Logger,
) *DynamoDBApi {
	return &DynamoDBApi{
		config: config,
		logger: logger,
		client: dynamodb.NewFromConfig(config),
	}
}

func (inst *DynamoDBApi) ListTables(force bool) ([]string, error) {
	if len(inst.allTables) > 0 && !force {
		return inst.allTables, nil
	}

	inst.allTables = nil
	var paginator = dynamodb.NewListTablesPaginator(
		inst.client, &dynamodb.ListTablesInput{},
	)

	var apiErr error = nil
	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Printf("Couldn't list tables: %v\n", err)
			apiErr = err
			break
		}

		inst.allTables = append(inst.allTables, output.TableNames...)
	}
	return inst.allTables, apiErr
}

func (inst *DynamoDBApi) FilterByName(name string) []string {

	if len(inst.allTables) < 1 {
		inst.ListTables(true)
	}

	var foundTables []string

	for _, tableName := range inst.allTables {
		found := strings.Contains(tableName, name)
		if found {
			foundTables = append(foundTables, tableName)
		}
	}
	return foundTables
}
func (inst *DynamoDBApi) DescribeTable(tableName string) (*types.TableDescription, error) {
	if len(tableName) == 0 {
		return nil, fmt.Errorf("Table name not set")
	}

	var output, err = inst.client.DescribeTable(context.TODO(),
		&dynamodb.DescribeTableInput{TableName: &tableName},
	)
	if err != nil {
		inst.logger.Printf("Failed to describe table: %v\n", err)
		return nil, err
	}
	return output.Table, nil
}

func (inst *DynamoDBApi) ScanTable(
	tableName string,
	scanExpression expression.Expression,
	indexName string,
	force bool,
) ([]map[string]interface{}, error) {
	var items []map[string]interface{}

	if len(tableName) == 0 {
		return items, fmt.Errorf("Table name not set")
	}

	var index *string = nil
	if len(indexName) > 0 {
		index = aws.String(indexName)
	}

	if force || inst.scanPaginator == nil {
		inst.scanPaginator = dynamodb.NewScanPaginator(inst.client, &dynamodb.ScanInput{
			TableName:                 aws.String(tableName),
			Limit:                     aws.Int32(20),
			FilterExpression:          scanExpression.Filter(),
			ExpressionAttributeNames:  scanExpression.Names(),
			ExpressionAttributeValues: scanExpression.Values(),
			ProjectionExpression:      scanExpression.Projection(),
			IndexName:                 index,
		})
	}

	var output, err = inst.scanPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Printf("Scan failed: %v\n", err)
	} else {
		var temp []map[string]interface{}
		attributevalue.UnmarshalListOfMaps(output.Items, &temp)
		items = append(items, temp...)
	}
	return items, err
}

func (inst *DynamoDBApi) QueryTable(
	tableName string,
	queryExpression expression.Expression,
	indexName string,
	force bool,
) ([]map[string]interface{}, error) {
	var items []map[string]interface{}

	if len(tableName) == 0 {
		return items, fmt.Errorf("Table name not set")
	}

	if inst.queryPaginator == nil || force {
		var index *string = nil
		if len(indexName) > 0 {
			index = aws.String(indexName)
		}
		inst.queryPaginator = dynamodb.NewQueryPaginator(inst.client, &dynamodb.QueryInput{
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
		return items, nil
	}

	var output, err = inst.queryPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return items, err
	}

	var temp []map[string]interface{}
	err = attributevalue.UnmarshalListOfMaps(output.Items, &temp)
	if err != nil {
		inst.logger.Println(err)
		return items, err
	}

	items = append(items, temp...)
	return items, nil
}
