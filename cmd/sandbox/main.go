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
	mux.HandleFunc("/hiring", handler.Index)
	mux.HandleFunc("/bootcamp", handler.Bootcamp)
	mux.HandleFunc("/exercise", handler.Exercise)
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/api/events", handler.EventsStream)
	mux.HandleFunc("/api/wallets", handler.GetWallets)
	mux.HandleFunc("/api/wallets/reset", handler.ResetWallets)
	mux.HandleFunc("/api/scenarios/01/run", handler.RunScenario01)
	mux.HandleFunc("/api/scenarios/01/eval", handler.EvalScenario01Code)
	mux.HandleFunc("/api/scenarios/02/run", handler.RunScenario02)
	mux.HandleFunc("/api/scenarios/02/defend", handler.DefendScenario02)
	mux.HandleFunc("/api/scenarios/02/clear", handler.ClearIdempotencyRecords)
	mux.HandleFunc("/api/scenarios/03/run", handler.RunScenario03)
	mux.HandleFunc("/api/scenarios/03/submit-repo", handler.SubmitScenario03Repo)
	mux.HandleFunc("/api/scenarios/03/clear", handler.ClearScenario03)
	mux.HandleFunc("/api/scenarios/04/run", handler.RunScenario04)
	mux.HandleFunc("/api/bootcamp/eval-livecode", handler.EvalBootcampLiveCode)
	mux.HandleFunc("/api/bootcamp/eval-systemdesign", handler.EvalBootcampSystemDesignCode)
	mux.HandleFunc("/api/gamification/status", handler.GetGamificationStatus)
	mux.HandleFunc("/api/gamification/reset", handler.ResetGamification)
	mux.HandleFunc("/api/exercise/eval-drill", handler.EvalExerciseDrill)
	mux.HandleFunc("/api/exercise/quiz-check", handler.CheckExerciseQuiz)
	mux.HandleFunc("/api/docs/postman", handler.DownloadPostmanCollection)

	port := ":8080"
	fmt.Printf("Dashboard aktif di http://localhost%s\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server gagal berjalan: %v", err)
	}
}
