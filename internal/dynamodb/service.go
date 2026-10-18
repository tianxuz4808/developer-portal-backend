package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/google/uuid"
	"github.com/tianxuz4808/developer-portal-backend/internal/domain"
	"github.com/tianxuz4808/developer-portal-backend/internal/logging"
	"github.com/tianxuz4808/developer-portal-backend/internal/repo"
	"go.uber.org/zap"
)

const TABLE_NAME = "services"

type DynamoStorageService struct {
	ID    string `json:"id" dynamodbav:"id"`
	Name  string `json:"name" dynamodbav:"name"`
	Owner string `json:"owner" dynamodbav:"owner"`

	Team         string   `json:"team" dynamodbav:"team"`
	Dependencies []string `json:"dependencies" dynamodbav:"dependencies"`

	Type           string `json:"type" dynamodbav:"type"`
	Environment    string `json:"environment" dynamodbav:"environment"`
	HealthCheckURL string `json:"health_check_url" dynamodbav:"health_check_url"`

	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

func toDynamoStorageService(s domain.Service) DynamoStorageService {
	dependenciesStr := []string{}
	for _, dep := range s.Dependencies {
		dependenciesStr = append(dependenciesStr, dep.String())
	}
	return DynamoStorageService{
		ID:    s.ID.String(),
		Name:  s.Name,
		Owner: s.Owner,
		Team:  s.Team,

		Dependencies: dependenciesStr,

		Type:           s.Type,
		Environment:    s.Environment,
		HealthCheckURL: s.HealthCheckURL,

		CreatedAt: s.CreatedAt,
	}
}

func (dss *DynamoStorageService) toDomainService() (domain.Service, error) {
	id, err := uuid.Parse(dss.ID)
	if err != nil {
		return domain.Service{}, err
	}

	dependencies := []uuid.UUID{}
	for _, depStrID := range dss.Dependencies {
		depID, err := uuid.Parse(depStrID)
		if err != nil {
			return domain.Service{}, err
		}
		dependencies = append(dependencies, depID)
	}

	return domain.Service{
		ID:    id,
		Name:  dss.Name,
		Owner: dss.Owner,
		Team:  dss.Team,

		Dependencies: dependencies,

		Type:           dss.Type,
		Environment:    dss.Environment,
		HealthCheckURL: dss.HealthCheckURL,

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
		var storageService DynamoStorageService
		err := attributevalue.UnmarshalMap(unmarshaledServices, &storageService)
		if err != nil {
			return nil, err
		}
		service, err := storageService.toDomainService()
		if err != nil {
			return nil, err
		}
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

func (dc DynamoClient) Write(ctx context.Context, service domain.Service) error {
	// Steps:
	//	1. create the service using NewService
	//	1. convert service to dynamo item
	// 	1. call putItem
	// 	2. profit?
	dc.Logger.Info("the service to write is: ", zap.Any("service", service))
	dynamoItem := toDynamoStorageService(service)
	attributeItem, err := attributevalue.MarshalMap(dynamoItem)
	if err != nil {
		return err
	}
	putItemParams := dynamodb.PutItemInput{
		Item:      attributeItem,
		TableName: aws.String(TABLE_NAME),
	}
	output, err := dc.Client.PutItem(ctx, &putItemParams)

	if err != nil {
		return err
	}

	dc.Logger.Info("the service written was: ", zap.Any("service", output.Attributes))

	metadata := output.ResultMetadata
	dc.Logger.Info("The result metadata from the putItem operation is: ",
		zap.Any("metadata", metadata),
	)

	return nil
}

func (dc DynamoClient) Delete(ctx context.Context, id string) error {
	// dc.Client.DeleteItem()
	pk, err := attributevalue.MarshalMap(map[string]string{"id": id})
	if err != nil {
		return err
	}

	_, err = dc.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		Key:       pk,
		TableName: aws.String(TABLE_NAME),
	})
	if err != nil {
		return err
	}

	return nil
}

func (dc DynamoClient) Exists(ctx context.Context, id uuid.UUID) (bool, domain.Service, error) {
	dc.Logger.Info("The id to be retrieved is: ", zap.Any("id", id))
	keyCondition := expression.Key("id").Equal(expression.Value(id.String()))
	builtExpression, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return false, domain.Service{}, err
	}

	dc.Logger.Info("the built expression is", zap.Any("expr", builtExpression))
	params := dynamodb.QueryInput{
		TableName:                 aws.String(TABLE_NAME),
		KeyConditionExpression:    builtExpression.KeyCondition(),
		ExpressionAttributeNames:  builtExpression.Names(),
		ExpressionAttributeValues: builtExpression.Values(),
	}
	output, err := dc.Client.Query(ctx, &params)
	if err != nil {
		return false, domain.Service{}, err
	}

	if len(output.Items) > 1 {
		return true, domain.Service{}, errors.New("returned multiple service for a given id")
	}

	if len(output.Items) <= 0 {
		return false, domain.Service{}, nil
	}

	var service domain.Service
	for _, item := range output.Items {
		var dynamoService DynamoStorageService
		err := attributevalue.UnmarshalMap(item, &dynamoService)
		if err != nil {
			return false, domain.Service{}, err
		}

		service, err = dynamoService.toDomainService()
		if err != nil {
			return true, domain.Service{}, err
		}
	}

	// TODO: implement a check to see if service is an empty domain.Service{}

	return true, service, nil
}

const AWS_DYNAMODB_BATCH_LIMIT = 20

func (dc DynamoClient) WriteBatch(ctx context.Context, batchServices []domain.Service) []error {
	requests := []types.WriteRequest{}

	// this is the real logical block when i want i remove the if above
	startTime := time.Now()

	for _, service := range batchServices {
		// requestService, err := attributevalue.MarshalMap(dynamoService.ConvFromService(newService))
		putItem, err := serviceToDynamoWriteRequest(service)
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
