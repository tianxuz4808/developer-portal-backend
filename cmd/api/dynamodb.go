package api

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tianxuz4808/developer-portal-backend/internal/local"
	"github.com/tianxuz4808/developer-portal-backend/internal/logging"
	services "github.com/tianxuz4808/developer-portal-backend/internal/service"
	"go.uber.org/zap"
)

type Clients struct {
	DynamoClient *dynamodb.Client
}

const TABLE_NAME = "services"

func (clients *Clients) WriteService(ctx context.Context, logger zap.Logger, service services.Service) error {
	item, err := attributevalue.MarshalMap(service.DynamoItemService())
	if err != nil {
		return err
	}

	_, err = clients.DynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TABLE_NAME),
		Item:      item,
	})
	if err != nil {
		return err
	}

	log.Printf("finished adding: %v\n", service)

	return nil
}

// ListServices will use the dynamodb scan() method to get the entire list from dynamo.
func (clients *Clients) ListServices(ctx context.Context, logger zap.Logger) ([]services.Service, error) {
	startTime := time.Now()

	// TODO: add parallel processing for large amounts of services
	op, err := clients.DynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName:              aws.String(TABLE_NAME),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityTotal,
	})

	for i, service := range op.Items {
		logger.Info("",
			zap.Int("record", i),
			zap.Any("service", service),
		)
	}

	if err != nil {
		return nil, err
	}
	endTime := time.Now()
	logging.LogTimeTaken(logger, startTime, endTime)
	return nil, nil
}

func (clients *Clients) CreateBatchServices(ctx context.Context, logger zap.Logger, batchServices []services.Service) error {

	logger.Info("the batch services from the request is: ",
		zap.Any("request-batch-services", batchServices),
	)
	requests := []types.WriteRequest{}
	if len(batchServices) == 0 {
		// this is simply for testing. if the size is zero, i'll generate a shit ton of services to try stress the process out
		for i := 0; i < 20; i++ {
			randPart := local.GenerateRandomString(5)
			serviceName := fmt.Sprintf("%s-service", randPart)

			newRandService := services.NewService(serviceName, local.GenerateRandomString(5)+"-owner")

			requestService, err := attributevalue.MarshalMap(newRandService.DynamoItemService())
			if err != nil {
				return err
			}
			requests = append(requests, types.WriteRequest{
				PutRequest: &types.PutRequest{
					Item: requestService,
				},
			})
		}
	} else {
		// this is the real logical block when i want i remove the if above
		for _, service := range batchServices {
			newService := services.NewService(service.Name, service.Owner)

			requestService, err := attributevalue.MarshalMap(newService.DynamoItemService())
			if err != nil {
				return err
			}

			requests = append(requests, types.WriteRequest{
				PutRequest: &types.PutRequest{
					Item: requestService,
				},
			})
		}
	}

	startTime := time.Now()

	_, err := clients.DynamoClient.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			TABLE_NAME: requests,
		},
	})

	if err != nil {
		return err
	}

	logger.Info("finished writing batch items: ",
		zap.Any("batch-services", requests),
	)

	endTime := time.Now()
	logging.LogTimeTaken(logger, startTime, endTime)

	return nil
}
