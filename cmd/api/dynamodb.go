package api

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamoService "github.com/tianxuz4808/developer-portal-backend/internal/dynamodb"
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
	item, err := attributevalue.MarshalMap(dynamoService.ConvFromService(service))
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

	if err != nil {
		return nil, err
	}

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

	endTime := time.Now()
	logging.LogTimeTaken(logger, startTime, endTime)
	return allServices, nil
}

// testing function, should move this later
func generateRandomServices(limit int) []services.Service {

	batchServices := []services.Service{}
	for i := 0; i < limit; i++ {
		randPart := strconv.Itoa(i) + local.GenerateRandomString(5)
		serviceName := fmt.Sprintf("%s-service", randPart)

		newRandService := services.NewService(serviceName, local.GenerateRandomString(5)+"-owner")

		batchServices = append(batchServices, newRandService)
	}

	return batchServices
}

func generateRandomDynamoServiceWrites(limit int) ([]types.WriteRequest, error) {
	randServices := generateRandomServices(limit)
	allPuts := []types.WriteRequest{}
	for _, service := range randServices {
		putItem, err := serviceToDynamoWriteRequest(service)
		if err != nil {
			return nil, err
		}

		allPuts = append(allPuts, putItem)
	}

	return allPuts, nil
}

func serviceToDynamoWriteRequest(service services.Service) (types.WriteRequest, error) {
	dynamoItem, err := attributevalue.MarshalMap(dynamoService.ConvFromService(service))
	if err != nil {
		return types.WriteRequest{}, err
	}

	writeRequest := types.WriteRequest{
		PutRequest: &types.PutRequest{
			Item: dynamoItem,
		},
	}

	return writeRequest, nil
}

// the true batch limit for the BatchWriteItem() function is 25, but i'm using 20 just to be safe
const AWS_DYNAMODB_BATCH_LIMIT = 20

func (clients *Clients) CreateBatchServices(ctx context.Context, logger zap.Logger, batchServices []services.Service) []error {

	logger.Info("the batch services from the request is: ",
		zap.Any("request-batch-services", batchServices),
	)

	// for testing. if the size is zero, i'll generate a shit ton of services to try stress the process out
	if len(batchServices) == 0 {
		batchServices = generateRandomServices(10)
	}

	requests := []types.WriteRequest{}

	// this is the real logical block when i want i remove the if above
	startTime := time.Now()

	for _, service := range batchServices {
		newService := services.NewService(service.Name, service.Owner)
		// requestService, err := attributevalue.MarshalMap(dynamoService.ConvFromService(newService))
		putItem, err := serviceToDynamoWriteRequest(newService)
		if err != nil {
			return []error{err}
		}

		requests = append(requests, putItem)
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

		// SOLEY FOR TESTING WHETHER MY UNPROCCESSED ITEMS WORKS
		randomUnproccessedItems, err := generateRandomDynamoServiceWrites(3)
		if err != nil {
			return nil, err
		}

		opt.UnprocessedItems = map[string][]types.WriteRequest{TABLE_NAME: randomUnproccessedItems}

		// END SOLELY FOR TESTING

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
				if len(unprocessedItems) == 0 {
					return
				}
				retryItem := dynamoRetryItem{
					items:     unprocessedItems,
					numTries:  1,
					lastTried: time.Now(),
				}
				clients.RetryDynamoUnprocessedItems(ctx, logger, &retryItem)
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

	errs := []error{}
	for cErr := range allErrors {
		if cErr != nil {
			errs = append(errs, cErr)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	logger.Info("finished writing batch items: ",
		zap.Any("batch-services", requests),
	)

	endTime := time.Now()
	logging.LogTimeTaken(logger, startTime, endTime)

	return nil
}

type dynamoRetryItem struct {
	items     []types.WriteRequest
	numTries  int
	lastTried time.Time
}

const MAX_RETRY_LIMIT = 3
const RETRY_TIME_LIMIT = time.Second * 15

func (clients *Clients) RetryDynamoUnprocessedItems(ctx context.Context, logger zap.Logger, unprocessedItems *dynamoRetryItem) (*dynamoRetryItem, error) {
	if unprocessedItems == nil {
		return nil, nil
	}

	if unprocessedItems.numTries >= 3 {
		return unprocessedItems, fmt.Errorf("Unable to process your batch write. Maximum number of retries attempted to dynamo")
	}

	unprocessedTime := unprocessedItems.lastTried
	timeToProcessAgain := unprocessedTime.Add(RETRY_TIME_LIMIT)

	if time.Now().Compare(timeToProcessAgain) == -1 {
		logger.Info("we are sleeping until we are allowed to write to dynamo again based off the RETRY_TIME_LIMIT")
	}

	for time.Now().Compare(timeToProcessAgain) == -1 {

	}

	opt, err := clients.DynamoClient.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			TABLE_NAME: unprocessedItems.items,
		},
	})

	if err != nil {
		return nil, err
	}

	unprocessedItems.items = opt.UnprocessedItems[TABLE_NAME]
	unprocessedItems.numTries += 1
	unprocessedItems.lastTried = time.Now()

	return clients.RetryDynamoUnprocessedItems(ctx, logger, unprocessedItems)
}
