package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tianxuz4808/developer-portal-backend/cmd/api"
	"github.com/tianxuz4808/developer-portal-backend/internal/server"
	services "github.com/tianxuz4808/developer-portal-backend/internal/service"
	"go.uber.org/zap"
)

func main() {
	dynamodbEndpoint := os.Getenv("DYNAMODB_ENDPOINT")
	awsRegion := os.Getenv("AWS_REGION")
	log.Println("the dynamo endpoint from the env var is: ", dynamodbEndpoint)
	log.Println("the aws region is: ", awsRegion)

	log.Println("creating a new service...")
	// setting up logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	mainCtx := context.TODO()

	// loading aws credentials
	cfg, err := config.LoadDefaultConfig(mainCtx, config.WithRegion(awsRegion))
	if err != nil {
		log.Fatalf("unable to load aws sdk config, %v", err)
	}

	svc := setUpDynamoConnection(cfg, dynamodbEndpoint)

	resp, err := svc.ListTables(context.TODO(), &dynamodb.ListTablesInput{
		Limit: aws.Int32(5),
	})

	if err != nil {
		log.Fatalf("failed to list tables, %v", err)
	}

	fmt.Println("Tables:")
	for _, tableName := range resp.TableNames {
		fmt.Println(tableName)
	}

	myService := services.NewService("my-service", "matthew")
	clients := api.Clients{
		DynamoClient: svc,
	}

	err = clients.WriteService(mainCtx, myService)
	if err != nil {
		log.Fatalf("error is: %v", err)
	}

	server := server.NewServer()
	server.Run()

	log.Println("added a service")
}

func setUpDynamoConnection(awsCfg aws.Config, dynamoEndpoint string) *dynamodb.Client {
	svc := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(dynamoEndpoint)
		log.Println("DYNAMO ENDPOINT:", *o.BaseEndpoint)
	})

	return svc
}
