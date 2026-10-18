package repo

import (
	"context"

	"github.com/tianxuz4808/developer-portal-backend/internal/domain"
)

// this package creates the general interface structure I want to use to communticate to any repo (right now dynamo, but I should be able to plug and play this whenever I want)

type ServiceRepo interface {
	List(context.Context) ([]domain.Service, error)
	WriteBatch(context.Context, []domain.Service) []error // write happens in batches so i expect an error of errors?
	Write(context.Context, domain.Service) error
	Delete(context.Context, string) error
}
