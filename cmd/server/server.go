package server

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tianxuz4808/developer-portal-backend/cmd/api"
	"go.uber.org/zap"
)

type Server interface {
	Run(context.Context, zap.Logger)
}

type server struct {
	DynamodbClient dynamodb.Client
}

func NewServer(dynamodbClient dynamodb.Client) Server {
	return &server{
		DynamodbClient: dynamodbClient,
	}
}

func (s *server) Run(ctx context.Context, logger zap.Logger) {
	log.Println("starting server...")
	mux := http.NewServeMux()

	clients := api.Clients{
		DynamoClient: &s.DynamodbClient,
	}

	mux.Handle("GET /list/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clients.ListServices(ctx, logger)
	}))

	err := http.ListenAndServe(":5000", mux)
	if err != nil {
		log.Fatal(err)
	}
}
