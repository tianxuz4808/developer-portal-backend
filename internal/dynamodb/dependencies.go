package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tianxuz4808/developer-portal-backend/internal/domain"
	"github.com/tianxuz4808/developer-portal-backend/internal/repo"
	"go.uber.org/zap"
)

type DynamoDependency struct {
	SourceID          string `json:"source_id" dynamodbav:"source_id"`
	TargetDependentID string `json:"target_dependent_id" dynamodbav:"dependency_id"`
}

type DynamoDependencyClient struct {
	Client *dynamodb.Client
	Logger *zap.Logger
}

func NewDynamoDependencyClient(client *dynamodb.Client, logger *zap.Logger) repo.DependencyRepo {
	return DynamoDependencyClient{
		Client: client,
		Logger: logger,
	}
}

const DEPENDENCIES_TABLE_NAME = "dependencies"

func (ddc DynamoDependencyClient) List(ctx context.Context) ([]domain.DependencyParams, error) {

	return nil, nil
}

func (ddc DynamoDependencyClient) RemoveDependecy(ctx context.Context, dep domain.DependencyParams) error {
	//
	return nil
}

func (ddc DynamoDependencyClient) AddDependencies(ctx context.Context, dep []domain.DependencyParams) error {
	// steps:
	// 2. Add or update the `dependencies` table by insterting each item
	// 3. If I fail to add a single dependency i should fail adding any dependency to the table
	// ddc.Client.BatchWriteItem()

	return nil
}

func (ddc DynamoDependencyClient) Add(ctx context.Context, dep domain.Dependency) error {
	// call dynamo putItem to add the single dependency. All of the ansillary stuff like checking whether the ids exist
	// should already be taken care of. The reason I pass the domain.Dependency obj is so I know dep exists

	dynamoItem := DynamoDependency{
		SourceID:          dep.SourceService.ID.String(),
		TargetDependentID: dep.TargetDependentService.ID.String(),
	}

	item, err := attributevalue.MarshalMap(dynamoItem)
	if err != nil {
		return err
	}

	ddc.Logger.Info("the item to be layed down is", zap.Any("item", item))

	output, err := ddc.Client.PutItem(ctx, &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(DEPENDENCIES_TABLE_NAME),
	})

	if err != nil {
		return err
	}

	ddc.Logger.Info("the output metadata is", zap.Any("dependencies_putitem_output", output))

	return nil
}
