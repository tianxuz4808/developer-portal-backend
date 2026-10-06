package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tianxuz4808/developer-portal-backend/cmd/api"
	services "github.com/tianxuz4808/developer-portal-backend/internal/service"
	"go.uber.org/zap"
)

func main() {
	log.Println("creating a new service...")
	// setting up logger
	logger, _ := zap.NewProduction()
	defer logger.Sync() 

	mainCtx := context.TODO()

	// loading aws credentials
	cfg, err := config.LoadDefaultConfig(mainCtx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("unable to load aws sdk config, %v", err)
	}

	svc := setUpDynamoConnection(cfg, true)

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

	log.Println("added a service")
}

func setUpDynamoConnection(awsCfg aws.Config, localhostOpt bool) *dynamodb.Client {

	svc := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		if localhostOpt == true {
			o.BaseEndpoint = aws.String("http://localhost:8000")
		}
	})

	return svc

}
