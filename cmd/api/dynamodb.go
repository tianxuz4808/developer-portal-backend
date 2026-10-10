package api

import (
	"context"
	"fmt"
	"log"
	"sync"
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

	// Starting to write the fan-out fan-in method to grab large amounts of services in go.

	// TODO: add parallel processing for large amounts of services
	op, err := clients.DynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName:              aws.String(TABLE_NAME),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityTotal,
	})

	allServices := []services.Service{}
	for i, unmarshaledServices := range op.Items {
		var service services.Service
		attributevalue.UnmarshalMap(unmarshaledServices, &service)
		allServices = append(allServices, service)
		logger.Info("",
			zap.Int("record", i),
			zap.Any("service", unmarshaledServices),
		)
	}

	if err != nil {
		return nil, err
	}

	endTime := time.Now()
	logging.LogTimeTaken(logger, startTime, endTime)
	return allServices, nil
}

// the true batch limit for the BatchWriteItem() function is 25, but i'm using 20 just to be safe
const AWS_DYNAMODB_BATCH_LIMIT = 20

func (clients *Clients) CreateBatchServices(ctx context.Context, logger zap.Logger, batchServices []services.Service) error {

	logger.Info("the batch services from the request is: ",
		zap.Any("request-batch-services", batchServices),
	)
	// this is just for testing...
	if len(batchServices) == 0 {
		// this is simply for testing. if the size is zero, i'll generate a shit ton of services to try stress the process out
		for i := 0; i < 10000; i++ {
			randPart := local.GenerateRandomString(5)
			serviceName := fmt.Sprintf("%s-service", randPart)

			newRandService := services.NewService(serviceName, local.GenerateRandomString(5)+"-owner")

			batchServices = append(batchServices, newRandService)
		}
	}

	requests := []types.WriteRequest{}

	// this is the real logical block when i want i remove the if above
	startTime := time.Now()

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
	logger.Info("the requests are",
		zap.Any("requests", requests),
	)

	// work batch writes to dynamodb. The request takes in jobs or requests to write to dynamo and 
	// returns a slice of unprocessed items along with an error if there is any
	work := func(ctx context.Context, logger zap.Logger, jobs []types.WriteRequest) ([]types.WriteRequest, error) {
		if len(jobs) == 0 {
			return nil, nil
		}

		// need to do this bc jobs can contain WriteRequests's with null values which is not valid
		// TODO: BatchWriteItem returns a list of items that were not written. I need to be able to get those items and retry.
		// There can also be the case of throttling in which i need some kind of backoff / exponential back off.
		opt, err := clients.DynamoClient.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				TABLE_NAME: jobs,
			},
		})

		if err != nil {
			logger.Error("error writing batch items to dynamo",
				zap.Error(err),
			)

			// errsCh <- err
			return nil, err
		}

		return opt.UnprocessedItems[TABLE_NAME], nil
	}

	var waitGroup sync.WaitGroup
	job := []types.WriteRequest{}
	const MAX_DYNAMODB_CLIENTS = 5
	sem := make(chan struct{}, MAX_DYNAMODB_CLIENTS)
	numTasks := (len(requests) + AWS_DYNAMODB_BATCH_LIMIT - 1) / AWS_DYNAMODB_BATCH_LIMIT
	allErrors := make(chan error, numTasks)
	// splitting up the batchServices into jobs of size AWS_DYNAMODB_BATCH_LIMIT
	for i, request := range requests {
		job = append(job, request)
		if len(job) == AWS_DYNAMODB_BATCH_LIMIT || i == len(requests)-1 {
			batch := job

			// just practice for adding a job NOT using the waitGroup.Go()
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				sem <- struct{}{}
				defer func() {
					<-sem
				}()
				unprocessedItems, err := work(ctx, logger, batch)
				if err != nil {
					allErrors <- err
				} else {
					allErrors <- nil
				}
				logger.Info("", zap.Any("unprocessedItems", unprocessedItems))

			}()

			// waitGroup.Go(func() {
			// 	// defer waitGroup.Done()
			// 	work(ctx, logger, batch, sem) // allErrors

			// 	<-sem
			// })
			job = []types.WriteRequest{}
		}
	}
	waitGroup.Wait()

	close(allErrors)
	// close(sem)

	for err := range allErrors {
		if err != nil {
			logger.Error(err.Error())
		}
	}

	logger.Info("finished writing batch items: ",
		zap.Any("batch-services", requests),
	)

	endTime := time.Now()
	logging.LogTimeTaken(logger, startTime, endTime)

	return nil
}
