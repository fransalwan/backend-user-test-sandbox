package main

import (
	"fmt"
	"log"
	"net/http"

	deliveryHttp "github.com/fransalwan/backend-user-test-sandbox/internal/delivery/http"
)

func main() {
	fmt.Println("🚀 Memulai Backend User Test Sandbox (Fintech Dojo)...")

	handler, err := deliveryHttp.NewHandler()
	if err != nil {
		log.Fatalf("Gagal menginisialisasi HTTP handler: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.Index)
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/api/events", handler.EventsStream)
	mux.HandleFunc("/api/wallets", handler.GetWallets)
	mux.HandleFunc("/api/wallets/reset", handler.ResetWallets)
	mux.HandleFunc("/api/scenarios/01/run", handler.RunScenario01)
	mux.HandleFunc("/api/scenarios/02/run", handler.RunScenario02)
	mux.HandleFunc("/api/scenarios/02/clear", handler.ClearIdempotencyRecords)

	port := ":8080"
	fmt.Printf("Dashboard aktif di http://localhost%s\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server gagal berjalan: %v", err)
	}
}
