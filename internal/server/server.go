package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
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
		if err != nil {
			strErr := fmt.Sprintf("There was an error processing your request: ", err.Error())
			w.Write([]byte(strErr))
			return
		}

		logger.Info("the services returned by dynamo are: ", zap.Any("services_list", services))

		marshaledService, err := json.Marshal(services)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(marshaledService)
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

	mux.Handle("GET /services/exists/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceStringID := r.PathValue("id")
		serviceID, err := uuid.Parse(serviceStringID)

		if err != nil {
			logger.Error("there was an error parsing service string to uuid", zap.String("service_string_id", serviceStringID), zap.Any("service_uuid", serviceID))
			http.Error(w, "there was an error parsing your service uuid", 500)
			return
		}

		exists, service, err := s.ServiceRepo.Exists(ctx, serviceID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if !exists {
			w.Write([]byte("The id is not linked to a service"))
			return
		}

		item, err := json.Marshal(service)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Write(item)
	}))

	mux.Handle("POST /services/delete/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceID := r.PathValue("id")

		err := s.ServiceRepo.Delete(ctx, serviceID)
		if err != nil {
			http.Error(w, err.Error(), 500)
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

	mux.Handle("POST /dependencies/add/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// decode the request body -- DONE
		// check to make sure whether the source service exists FROM services table
		// check to make sure the target dependent exists FROM dependencies table
		// if either are false return an error to the user
		// on happy, insert the batch items into dynamo?
		defer r.Body.Close()
		var reqBody domain.Dependency
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			http.Error(w, "encountered an error with your request body", 500)
		}

		logger.Info("req body is: ", zap.Any("request_body", reqBody))
	}))

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
