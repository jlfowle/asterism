package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jlfowle/asterism/services/unifi/internal/api"
)

func main() {
	mux := http.NewServeMux()
	handler := api.NewHandler("unifi")
	handler.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("starting unifi service on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
