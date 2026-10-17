package domain

import "github.com/google/uuid"

type Dependency struct {
	SourceID          uuid.UUID
	TargetDependentID uuid.UUID
}
