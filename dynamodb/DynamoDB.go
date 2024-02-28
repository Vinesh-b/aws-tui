package dynamodb

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBApi struct {
	logger    *log.Logger
	config    aws.Config
	client    *dynamodb.Client
	allTables []string
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

	var paginator = dynamodb.NewListTablesPaginator(
		inst.client, &dynamodb.ListTablesInput{},
	)

	for paginator.HasMorePages() {
		var output, err = paginator.NextPage(context.TODO())
		if err != nil {
			log.Printf("Couldn't list tables: %v\n", err)
			break
		}

		inst.allTables = append(inst.allTables, output.TableNames...)
	}
	return inst.allTables
}

func (inst *DynamoDBApi) DescribeTable(tableName string) *types.TableDescription {

	var output, err = inst.client.DescribeTable(context.TODO(),
		&dynamodb.DescribeTableInput{TableName: &tableName},
	)
	if err != nil {
		log.Printf("Failed to describe table: %v\n", err)
		return nil
	}
	return output.Table
}

//func (inst *DynamoDBApi) ScanTable(tableName string) []map[string]interface{} {
//	var items []map[string]interface{}
//	var err error
//	var response *dynamodb.ScanOutput
//	scanPaginator := dynamodb.NewScanPaginator(inst.client, &dynamodb.ScanInput{
//		TableName: aws.String(tableName),
//		Limit:     aws.Int32(20),
//	})
//	for scanPaginator.HasMorePages() {
//		response, err = scanPaginator.NextPage(context.TODO())
//		if err != nil {
//			log.Printf("Scan failed: %v\n", err)
//			break
//		} else {
//			var temp map[string]interface{}
//			var marshalled = attributevalue.UnmarshalListOfMaps(response.Items, &temp)
//			items = append(items, marshalled...)
//		}
//	}
//	return items
//}
