package dynamodb

import (
	"time"

	"github.com/google/uuid"
	"github.com/tianxuz4808/developer-portal-backend/internal/service"
)

type DynamoStorageService struct {
	ID        string    `json:"id" dynamodbav:"id"`
	Name      string    `json:"name" dynamodbav:"name"`
	Owner     string    `json:"owner" dynamodbav:"owner"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

func ConvFromService(s service.Service) DynamoStorageService {
	return DynamoStorageService{
		ID: s.ID.String(),
		Name: s.Name,
		Owner: s.Owner,
		CreatedAt: s.CreatedAt,
	}
}

func (dss *DynamoStorageService) ConvToService() (service.Service, error) {
	id, err := uuid.Parse(dss.ID)
	if err != nil {
		return service.Service{}, err
	}

	return service.Service{
		ID:        id,
		Name:      dss.Name,
		Owner:     dss.Owner,
		CreatedAt: dss.CreatedAt,
	}, nil
}
