package api

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tianxuz4808/developer-portal-backend/internal/logging"
	services "github.com/tianxuz4808/developer-portal-backend/internal/service"
	"go.uber.org/zap"
)

type Clients struct {
	DynamoClient *dynamodb.Client
}

const TABLE_NAME = "services"

func (clients *Clients) WriteService(ctx context.Context, service services.Service) error {
	item, err := attributevalue.MarshalMap(map[string]any{
		"id":         service.ID.String(),
		"name":       service.Name,
		"owner":      service.Owner,
		"created_at": service.CreatedAt,
	})
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
	log.Println("listing out the services...")
	endTime := time.Now()
	logging.LogTimeTaken(logger, startTime, endTime)
	return nil, nil
}
