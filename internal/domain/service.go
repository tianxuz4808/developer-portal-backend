package domain 

import (
	"time"

	"github.com/google/uuid"
)

// Service struct
type Service struct {
	ID        uuid.UUID `json:"id" dynamodbav:"id"`
	Name      string    `json:"name" dynamodbav:"name"`
	Owner     string    `json:"owner" dynamodbav:"owner"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

// NewService func
func NewService(name string, owner string) Service {
	return Service{
		ID:        uuid.New(),
		Name:      name,
		Owner:     owner,
		CreatedAt: time.Now(),
	}
}

type DynamoStorageService struct {
	ID        string    `json:"id" dynamodbav:"id"`
	Name      string    `json:"name" dynamodbav:"name"`
	Owner     string    `json:"owner" dynamodbav:"owner"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

func ConvFromService(s Service) DynamoStorageService {
	return DynamoStorageService{
		ID: s.ID.String(),
		Name: s.Name,
		Owner: s.Owner,
		CreatedAt: s.CreatedAt,
	}
}

func (dss *DynamoStorageService) ConvToService() (Service, error) {
	id, err := uuid.Parse(dss.ID)
	if err != nil {
		return Service{}, err
	}

	return Service{
		ID:        id,
		Name:      dss.Name,
		Owner:     dss.Owner,
		CreatedAt: dss.CreatedAt,
	}, nil
}
