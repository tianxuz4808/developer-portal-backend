package api

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	services "github.com/tianxuz4808/developer-portal-backend/internal/service"
)

type Clients struct {
	DynamoClient *dynamodb.Client
}

func (clients *Clients) WriteService(ctx context.Context, service services.Service) error {
	item, err := attributevalue.MarshalMap(map[string]any{
		"id": service.ID.String(),
		"name": service.Name,
		"owner": service.Owner,
		"created_at": service.CreatedAt,
	})
	if err != nil {
		return err
	}

	_, err = clients.DynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("services"),
		Item:      item,
	})
	if err != nil {
		return err
	}

	log.Printf("finished adding: %v\n", service)

	return nil
}
