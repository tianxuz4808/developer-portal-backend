package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/tianxuz4808/developer-portal-backend/internal/domain"
)

type DependencyRepo interface {
	GetByServiceID(context.Context, uuid.UUID) ([]domain.Dependency, error)
	AddDependencies(context.Context, []domain.DependencyParams) error
	Add(context.Context, domain.Dependency) error
}
