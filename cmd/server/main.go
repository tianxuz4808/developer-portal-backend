package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tianxuz4808/developer-portal-backend/internal/server"

	iDynamo "github.com/tianxuz4808/developer-portal-backend/internal/dynamodb"
	"go.uber.org/zap"
)

func main() {
	// setting up logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("finished setting up the logger")

	dynamodbEndpoint := os.Getenv("DYNAMODB_ENDPOINT")
	awsRegion := os.Getenv("AWS_REGION")
	logger.Info("the aws credentials are: ",
		zap.String("Dynamodb Endpoint", dynamodbEndpoint),
		zap.String("Aws Region", awsRegion),
	)

	mainCtx := context.TODO()

	// loading aws credentials
	cfg, err := config.LoadDefaultConfig(mainCtx, config.WithRegion(awsRegion))
	if err != nil {
		logger.Fatal("unable to load the aws sdk config",
			zap.Error(err),
		)
	}

	svc := setUpDynamoConnection(cfg, dynamodbEndpoint)

	resp, err := svc.ListTables(context.TODO(), &dynamodb.ListTablesInput{
		Limit: aws.Int32(5),
	})

	if err != nil {
		logger.Fatal("failed to list tables",
			zap.Error(err),
		)
	}

	for i, tableName := range resp.TableNames {
		logger.Info("Tables:",
			zap.String(fmt.Sprintf("%d", i), tableName),
		)
	}

	serviceRepo := iDynamo.NewDynamoSericeRepo(iDynamo.DynamoClient{
		Client: svc,
		Logger: logger,
	})

	dependencyRepo := iDynamo.NewDynamoDependencyClient(svc, logger)

	server := server.NewServer(server.RepoClients{
		ServiceRepo:    serviceRepo,
		DependencyRepo: dependencyRepo,
	})
	server.Run(mainCtx, *logger)

	log.Println("added a service")
}

func setUpDynamoConnection(awsCfg aws.Config, dynamoEndpoint string) *dynamodb.Client {
	svc := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(dynamoEndpoint)
	})

	return svc
}
