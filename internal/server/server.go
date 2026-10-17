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

	mux.Handle("POST /services/write/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		logger.Info("the body from the request is ", zap.Any("reqbody", r.Body))
		var service domain.Service
		err := json.NewDecoder(r.Body).Decode(&service)
		if err != nil {
			errStr := fmt.Sprint("There was an error processing your write service request body")
			http.Error(w, errStr, 500)
			return
		}

		newService, err := domain.NewService(service)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		err = s.ServiceRepo.Write(ctx, newService)
		if err != nil {
			errStr := fmt.Sprintf("There was an error writing your service to storage: %+v", err)
			http.Error(w, errStr, 500)
			return
		}
		_, err = w.Write([]byte("success"))
		if err != nil {
			logger.Error("error writing success for some reason?", zap.Error(err))
			return
		}
	}))

	mux.Handle("POST /services/batch/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var batchServices []domain.Service

		err := json.NewDecoder(r.Body).Decode(&batchServices)
		if err != nil {
			errStr := fmt.Sprint("There was an error processing your batch services request body")
			http.Error(w, errStr, 500)
			return
		}

		logger.Info("Retrieved batch services from request",
			zap.Any("batch-services", batchServices),
		)

		fmtdServices := []domain.Service{}
		for _, service := range batchServices {
			newService, err := domain.NewService(service)
			if err != nil {
				strErr := fmt.Sprintf("one or more of your services has an issue: %s", err.Error())
				http.Error(w, strErr, 500)
				return
			}
			fmtdServices = append(fmtdServices, newService)
		}

		errs := s.ServiceRepo.WriteBatch(ctx, fmtdServices)
		if len(errs) > 0 {
			errStr := fmt.Sprint("There was an issue processing one or more of your batch write items request")
			http.Error(w, errStr, 500)
			return
		}
	}))

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
