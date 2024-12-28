package awsapi

import (
	"context"
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

func (inst *DynamoDBApi) ListTables(force bool) []string {
	if len(inst.allTables) > 0 && !force {
		return inst.allTables
	}

	inst.allTables = nil
	var paginator = dynamodb.NewListTablesPaginator(
		inst.client, &dynamodb.ListTablesInput{},
	)

	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			inst.logger.Printf("Couldn't list tables: %v\n", err)
			break
		}

		inst.allTables = append(inst.allTables, output.TableNames...)
	}
	return inst.allTables
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
func (inst *DynamoDBApi) DescribeTable(tableName string) *types.TableDescription {

	var output, err = inst.client.DescribeTable(context.TODO(),
		&dynamodb.DescribeTableInput{TableName: &tableName},
	)
	if err != nil {
		inst.logger.Printf("Failed to describe table: %v\n", err)
		return nil
	}
	return output.Table
}

func (inst *DynamoDBApi) ScanTable(
	tableName string,
	scanExpression expression.Expression,
	indexName string,
	force bool,
) []map[string]interface{} {
	var items []map[string]interface{}

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
	return items
}

func (inst *DynamoDBApi) QueryTable(
	tableName string,
	queryExpression expression.Expression,
	indexName string,
	force bool,
) []map[string]interface{} {
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

	var items = make([]map[string]interface{}, 0)

	if !inst.queryPaginator.HasMorePages() {
		return items
	}

	var output, err = inst.queryPaginator.NextPage(context.TODO())
	if err != nil {
		inst.logger.Println(err)
		return items
	}

	var temp []map[string]interface{}
	err = attributevalue.UnmarshalListOfMaps(output.Items, &temp)
	if err != nil {
		inst.logger.Println(err)
		return items
	}

	items = append(items, temp...)
	return items
}
