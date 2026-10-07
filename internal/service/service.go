package service

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

func (s Service) DynamoItemService() map[string]any {
	return map[string]any{
		"id": s.ID.String(),
		"name": s.Name,
		"owner": s.Owner,
		"created_at": s.CreatedAt,
	}
}
