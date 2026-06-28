package main

import (
	"errors"
	"log"
	"net/http"

	"bank-system/internal/httpapi"
)

func main() {
	server := &http.Server{
		Addr:    ":8080",
		Handler: httpapi.NewRouter(),
	}

	log.Printf("starting bank-system server on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped unexpectedly: %v", err)
	}
}
