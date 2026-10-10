package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/alirezarajaee/roveros/internal/api"
)

func main() {
	handler, err := api.NewServer()
	if err != nil {
		log.Fatalf("initialize ROVEROS API: %v", err)
	}

	server := &http.Server{
		Addr:              "127.0.0.1:8081",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("ROVEROS API listening at http://%s", server.Addr)
	log.Println("Press Ctrl+C to stop the server.")

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("ROVEROS API server failed: %v", err)
	}
}
