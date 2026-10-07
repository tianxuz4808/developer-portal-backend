package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tianxuz4808/developer-portal-backend/cmd/api"
	services "github.com/tianxuz4808/developer-portal-backend/internal/service"
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

	mux.Handle("GET /services/list/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clients.ListServices(ctx, logger)
	}))

	mux.Handle("POST /services/batch/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		// body, _ := io.ReadAll(r.Body)
		// logger.Info("", zap.String("body", string(body)))
		var batchServices []services.Service

		err := json.NewDecoder(r.Body).Decode(&batchServices)

		if err != nil {
			// logger.Fatal("could not decode the batch services to write to dynamo",
			// 	zap.Error(err),
			// )
			w.Write([]byte("There was an error processing your batch services request body"))
		}

		logger.Info("Retrieved batch services from request",
			zap.Any("batch-services", batchServices),
		)
		clients.CreateBatchServices(ctx, logger, batchServices)
	}))

	err := http.ListenAndServe(":5000", mux)
	if err != nil {
		log.Fatal(err)
	}
}
