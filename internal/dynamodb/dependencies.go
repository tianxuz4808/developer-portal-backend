package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tianxuz4808/developer-portal-backend/internal/domain"
	"github.com/tianxuz4808/developer-portal-backend/internal/repo"
	"go.uber.org/zap"
)

type DynamoDependency struct {
	SourceID  string
	DependsOn []string
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

func (ddc DynamoDependencyClient) List(ctx context.Context) ([]domain.Dependency, error) {
	return nil, nil
}

func (ddc DynamoDependencyClient) AddDependencies(ctx context.Context, dep []domain.Dependency) error {
	return nil
}
