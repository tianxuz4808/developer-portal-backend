package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tianxuz4808/developer-portal-backend/internal/domain"
	"github.com/tianxuz4808/developer-portal-backend/internal/repo"
	"go.uber.org/zap"
)

type DynamoDependency struct {
	SourceID          string
	TargetDependentID string
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
	return nil
}
