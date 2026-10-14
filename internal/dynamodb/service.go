package dynamodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/tianxuz4808/developer-portal-backend/internal/domain"
	"github.com/tianxuz4808/developer-portal-backend/internal/logging"
	"github.com/tianxuz4808/developer-portal-backend/internal/repo"
	"go.uber.org/zap"
)

const TABLE_NAME = "services"

type DynamoStorageService struct {
	ID        string    `json:"id" dynamodbav:"id"`
	Name      string    `json:"name" dynamodbav:"name"`
	Owner     string    `json:"owner" dynamodbav:"owner"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

func toDynamoStorageService(s domain.Service) DynamoStorageService {
	return DynamoStorageService{
		ID:        s.ID.String(),
		Name:      s.Name,
		Owner:     s.Owner,
		CreatedAt: s.CreatedAt,
	}
}

func (dss *DynamoStorageService) toDomainService() (domain.Service, error) {
	id, err := uuid.Parse(dss.ID)
	if err != nil {
		return domain.Service{}, err
	}

	return domain.Service{
		ID:        id,
		Name:      dss.Name,
		Owner:     dss.Owner,
		CreatedAt: dss.CreatedAt,
	}, nil
}

type DynamoClient struct {
	Client *dynamodb.Client
	Logger *zap.Logger
}

func NewDynamoRepo(dc DynamoClient) repo.ServiceRepo {
	return dc
}

func (dc DynamoClient) List(ctx context.Context) ([]domain.Service, error) {
	startTime := time.Now()
	// TODO: add parallel processing for large amounts of services
	op, err := dc.Client.Scan(ctx, &dynamodb.ScanInput{
		TableName:              aws.String(TABLE_NAME),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityTotal,
	})

	if err != nil {
		return nil, err
	}

	allServices := []domain.Service{}
	for i, unmarshaledServices := range op.Items {
		var service domain.Service
		attributevalue.UnmarshalMap(unmarshaledServices, &service)
		allServices = append(allServices, service)
		dc.Logger.Info("",
			zap.Int("record", i),
			zap.Any("service", unmarshaledServices),
		)
	}

	endTime := time.Now()
	logging.LogTimeTaken(*dc.Logger, startTime, endTime)
	return allServices, nil
}

func serviceToDynamoWriteRequest(service domain.Service) (types.WriteRequest, error) {
	dynamoItem, err := attributevalue.MarshalMap(toDynamoStorageService(service))
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

const AWS_DYNAMODB_BATCH_LIMIT = 20

func (dc DynamoClient) Write(ctx context.Context, batchServices []domain.Service) []error {
	requests := []types.WriteRequest{}

	// this is the real logical block when i want i remove the if above
	startTime := time.Now()

	for _, service := range batchServices {
		newService := domain.NewService(service.Name, service.Owner, service.Dependencies)
		// requestService, err := attributevalue.MarshalMap(dynamoService.ConvFromService(newService))
		putItem, err := serviceToDynamoWriteRequest(newService)
		if err != nil {
			return []error{err}
		}

		requests = append(requests, putItem)
	}
	dc.Logger.Info("the requests are",
		zap.Any("requests", requests),
	)

	// work batch writes to dynamodb. The request takes in jobs or requests to write to dynamo and
	// returns a slice of unprocessed items along with an error if there is any
	work := func(ctx context.Context, jobs []types.WriteRequest) ([]types.WriteRequest, error) {
		if len(jobs) == 0 {
			return nil, nil
		}

		// need to do this bc jobs can contain WriteRequests's with null values which is not valid
		// TODO: BatchWriteItem returns a list of items that were not written. I need to be able to get those items and retry.
		// There can also be the case of throttling in which i need some kind of backoff / exponential back off.
		opt, err := dc.Client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				TABLE_NAME: jobs,
			},
		})

		if err != nil {
			dc.Logger.Error("error writing batch items to dynamo",
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
				unprocessedItems, err := work(ctx, batch)
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
					retries:   0,
					lastTried: time.Now(),
				}
				dc.RetryDynamoUnprocessedItems(ctx, &retryItem)
				dc.Logger.Info("", zap.Any("unprocessedItems", unprocessedItems))

			}()

			// done using the more modern waitgroup.Go()
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

	dc.Logger.Info("finished writing batch items: ",
		zap.Any("batch-services", requests),
	)

	endTime := time.Now()
	logging.LogTimeTaken(*dc.Logger, startTime, endTime)

	return nil
}

type dynamoRetryItem struct {
	items     []types.WriteRequest
	retries   int
	lastTried time.Time
}

const MAX_RETRY_LIMIT = 3
const RETRY_TIME_LIMIT = time.Second * 15

func (dc DynamoClient) RetryDynamoUnprocessedItems(ctx context.Context, unprocessedItems *dynamoRetryItem) (*dynamoRetryItem, error) {
	if unprocessedItems == nil || len(unprocessedItems.items) == 0 {
		return unprocessedItems, nil
	}

	for range MAX_RETRY_LIMIT {

		timer := time.NewTimer(RETRY_TIME_LIMIT)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return unprocessedItems, fmt.Errorf("retry canceled: %w", ctx.Err())
		}

		opt, err := dc.Client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				TABLE_NAME: unprocessedItems.items,
			},
		})

		if err != nil {
			return unprocessedItems, err
		}

		unprocessedItems.items = opt.UnprocessedItems[TABLE_NAME]
		unprocessedItems.retries++
		unprocessedItems.lastTried = time.Now()

		if len(unprocessedItems.items) == 0 {
			return nil, nil
		}
	}

	if len(unprocessedItems.items) > 0 {
		return unprocessedItems, fmt.Errorf("Unable to process %d of your items", len(unprocessedItems.items))
	}

	return nil, nil
}
