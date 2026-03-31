package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jlfowle/asterism/services/cluster/internal/api"
)

func main() {
	mux := http.NewServeMux()
	handler := api.NewHandler("cluster")
	handler.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("starting cluster service on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
