package domain

import "github.com/google/uuid"

type Dependency struct {
	SourceID          uuid.UUID `json:"source_id" dynamodbav:"source_id"`
	TargetDependentID uuid.UUID `json:"target_dependent_id" dynamodbav:"target_dependent_id"`
}
