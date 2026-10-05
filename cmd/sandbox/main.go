package main

import (
	"fmt"
	"log"
	"net/http"

	deliveryHttp "github.com/fransalwan/backend-user-test-sandbox/internal/delivery/http"
)

func main() {
	fmt.Println("🚀 Starting Backend User Test Sandbox (Fintech Dojo)...")

	handler := deliveryHttp.NewHandler()
	http.HandleFunc("/health", handler.HealthCheck)

	port := ":8080"
	fmt.Printf("Listening on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
