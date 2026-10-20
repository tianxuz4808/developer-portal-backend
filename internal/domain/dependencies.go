package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type DependencyParams struct {
	SourceID          uuid.UUID `json:"source_id" dynamodbav:"source_id"`
	TargetDependentID uuid.UUID `json:"target_dependent_id" dynamodbav:"target_dependent_id"`
}

type Dependency struct {
	SourceService          Service
	TargetDependentService Service
}

// this function this represents is the call to Exists(). I'm not sure if this is anti pattern but it works for this case
type ExistsCheckerFunc func(context.Context, uuid.UUID) (bool, Service, error)

func NewDependency(ctx context.Context, params DependencyParams, checker ExistsCheckerFunc) (Dependency, error) {
	// this is where I should do the check to dynamo for both the source and target

	// TODO: MAKE THIS ASYNC
	serviceExists, service, err := checker(ctx, params.SourceID)
	if err != nil {
		return Dependency{}, err
	}

	if !serviceExists {
		return Dependency{}, errors.New("the service uuid does not exist!")
	}

	dependencyServiceExists, tdService, err := checker(ctx, params.TargetDependentID)
	if err != nil {
		return Dependency{}, err
	}

	if !dependencyServiceExists {
		return Dependency{}, errors.New("the dependency service uuid does not exist!")
	}

	return Dependency{
		SourceService:          service,
		TargetDependentService: tdService,
	}, nil
}
