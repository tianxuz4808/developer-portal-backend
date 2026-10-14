package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/tianxuz4808/developer-portal-backend/internal/domain"
	"github.com/tianxuz4808/developer-portal-backend/internal/repo"
	"go.uber.org/zap"
)

type Server interface {
	Run(context.Context, zap.Logger)
}

type server struct {
	ServiceRepo repo.ServiceRepo
}

func NewServer(serviceRepo repo.ServiceRepo) Server {
	return &server{
		ServiceRepo: serviceRepo,
	}
}

func (s *server) Run(ctx context.Context, logger zap.Logger) {
	log.Println("starting server...")
	log.Println("haha")

	mux := http.NewServeMux()
	mux.Handle("GET /services/list/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		// services, err := s.DynamodbClient.List(ctx)
		services, err := s.ServiceRepo.List(ctx)
		// services, err := clients.ListServices(ctx)
		if err != nil {
			strErr := fmt.Sprintf("There was an error processing your request: ", err.Error())
			w.Write([]byte(strErr))
			return
		}

		retStr := fmt.Sprintf("The amount of services collected are: ", len(services))
		w.Write([]byte(retStr))
	}))

	mux.Handle("POST /services/batch/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var batchServices []domain.Service

		err := json.NewDecoder(r.Body).Decode(&batchServices)

		if err != nil {
			errStr := fmt.Sprintf("There was an error processing your batch services request body")
			http.Error(w, errStr, 500)
			return
		}

		logger.Info("Retrieved batch services from request",
			zap.Any("batch-services", batchServices),
		)

		//errs := s.DynamodbClient.Write(ctx, batchServices)
		// errs := clients.CreateBatchServices(ctx, batchServices)
		errs := s.ServiceRepo.Write(ctx, batchServices)
		if len(errs) > 0 {
			errStr := fmt.Sprintf("There was an issue processing one or more of your batch write items request")
			http.Error(w, errStr, 500)
		}
	}))

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
