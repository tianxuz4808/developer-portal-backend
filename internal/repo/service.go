package repo

import (
	"context"

	"github.com/tianxuz4808/developer-portal-backend/internal/service"
)

// this package creates the general interface structure I want to use to communticate to any repo (right now dynamo, but I should be able to plug and play this whenever I want)

type Repo interface {
	List(context.Context) ([]service.Service, error)
	Write(context.Context, []service.Service) []error // write happens in batches so i expect an error of errors?
}


