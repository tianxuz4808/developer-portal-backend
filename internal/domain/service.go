package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Service name
// 	- deployed-environment
// 	- dependencies
// 	- health_check_url
// 	- owner
// 	- team
// 	- type?
// 		- This could be database, api, etc.

// Service struct
type Service struct {
	ID           uuid.UUID   `json:"id" dynamodbav:"id"`
	Name         string      `json:"name" dynamodbav:"name"`
	Owner        string      `json:"owner" dynamodbav:"owner"`
	Team         string      `json:"team" dynamodbav:"team"`
	Dependencies []uuid.UUID `json:"dependencies" dynamodbav:"dependencies"`

	Type           string `json:"type" dynamodbav:"type"`
	Environment    string `json:"environment" dynamodbav:"environment"`
	HealthCheckURL string `json:"health_check_url" dynamodbav:"health_check_url"`

	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

// NewService func
func NewService(s Service) (Service, error) {

	if len(s.Name) <= 0 {
		return Service{}, errors.New("cannot have an empty name")
	}

	if len(s.Type) <= 0 {
		return Service{}, errors.New("cannot have an empty type")
	}

	if len(s.Owner) <= 0 {
		return Service{}, errors.New("cannot have an empty owner")
	}

	if len(s.Team) <= 0 {
		return Service{}, errors.New("cannot have an empty team")
	}

	return Service{
		ID:             uuid.New(),
		Name:           s.Name,
		Owner:          s.Owner,
		Team:           s.Team,
		Dependencies:   s.Dependencies,
		Type:           s.Type,
		Environment:    s.Environment,
		HealthCheckURL: s.HealthCheckURL,
		CreatedAt:      time.Now(),
	}, nil
}
