package server

import (
	"log"
	"net/http"
)

type Server interface {
	Run()
}

type server struct {
}

func NewServer() Server {
	return &server{

	}
}

func (s *server) Run() {
	log.Println("starting server...")
	err := http.ListenAndServe(":5000", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("listening...")
	}))
	if err != nil {
		log.Fatal(err)
	}
}