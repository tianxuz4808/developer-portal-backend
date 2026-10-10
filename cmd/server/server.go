package server

import (
	"context"
	"encoding/json"
	"fmt"
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
		defer r.Body.Close()
		services, err := clients.ListServices(ctx, logger)
		if err != nil {
			strErr := fmt.Sprintf("There was an error processing your request: ", err.Error())
			w.Write([]byte(strErr))
			return
		}

		retStr := fmt.Sprintf("The amount of services collected are: ", len(services))
		w.Write([]byte(retStr))
		return
	}))

	mux.Handle("POST /services/batch/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var batchServices []services.Service

		err := json.NewDecoder(r.Body).Decode(&batchServices)

		if err != nil {
			errStr := fmt.Sprintf("There was an error processing your batch services request body")
			http.Error(w, errStr, 500)
			return
		}

		logger.Info("Retrieved batch services from request",
			zap.Any("batch-services", batchServices),
		)

		errs := clients.CreateBatchServices(ctx, logger, batchServices)
		if len(errs) > 0 {
			errStr := fmt.Sprintf("There was an issue processing one or more of your batch write items request")
			http.Error(w, errStr, 500)
		}

		return
	}))

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
