package repo

import (
	"context"

	"github.com/tianxuz4808/developer-portal-backend/internal/domain"
)

type DependencyRepo interface {
	List(context.Context) ([]domain.Dependency, error)
	AddDependencies(context.Context, []domain.Dependency) error
}

