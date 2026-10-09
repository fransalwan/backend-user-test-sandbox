package http_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	deliveryhttp "github.com/fransalwan/backend-user-test-sandbox/internal/delivery/http"
)

// TestTableDriven_LiveCodingAlgorithm menguji algoritma deduplikasi transaksi finansial
// dengan standar Table-Driven Unit Tests di Go.
func TestTableDriven_LiveCodingAlgorithm(t *testing.T) {
	type Transaction struct {
		ID        string
		UserID    string
		Amount    int64
		Timestamp int64
	}

	findDuplicates := func(txs []Transaction, windowSeconds int64) []string {
		lastSeen := make(map[string]int64)
		var duplicates []string
		for _, tx := range txs {
			key := tx.UserID + "_" + string(rune(tx.Amount))
			if prevTime, exists := lastSeen[key]; exists {
				if tx.Timestamp-prevTime <= windowSeconds {
					duplicates = append(duplicates, tx.ID)
				}
			}
			lastSeen[key] = tx.Timestamp
		}
		return duplicates
	}

	tests := []struct {
		name          string
		txs           []Transaction
		windowSeconds int64
		wantCount     int
		expectedIDs   []string
	}{
		{
			name:          "Empty transactions list",
			txs:           []Transaction{},
			windowSeconds: 60,
			wantCount:     0,
			expectedIDs:   nil,
		},
		{
			name: "No duplicates - different users",
			txs: []Transaction{
				{ID: "tx-1", UserID: "usr-A", Amount: 50000, Timestamp: 100},
				{ID: "tx-2", UserID: "usr-B", Amount: 50000, Timestamp: 110},
			},
			windowSeconds: 60,
			wantCount:     0,
			expectedIDs:   nil,
		},
		{
			name: "No duplicates - timestamp exceeds window",
			txs: []Transaction{
				{ID: "tx-1", UserID: "usr-A", Amount: 50000, Timestamp: 100},
				{ID: "tx-2", UserID: "usr-A", Amount: 50000, Timestamp: 250}, // selisih 150s > 60s
			},
			windowSeconds: 60,
			wantCount:     0,
			expectedIDs:   nil,
		},
		{
			name: "Duplicate detected within sliding window",
			txs: []Transaction{
				{ID: "tx-1", UserID: "usr-A", Amount: 50000, Timestamp: 100},
				{ID: "tx-2", UserID: "usr-A", Amount: 50000, Timestamp: 120}, // selisih 20s <= 60s
			},
			windowSeconds: 60,
			wantCount:     1,
			expectedIDs:   []string{"tx-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findDuplicates(tt.txs, tt.windowSeconds)
			if len(got) != tt.wantCount {
				t.Fatalf("expected %d duplicates, got %d", tt.wantCount, len(got))
			}
			for i, id := range tt.expectedIDs {
				if got[i] != id {
					t.Errorf("expected duplicate ID %s at index %d, got %s", id, i, got[i])
				}
			}
		})
	}
}

// TestHttpEndpoints_TableDriven menguji endpoint HTTP utama sandbox menggunakan mock & table-driven testing.
func TestHttpEndpoints_TableDriven(t *testing.T) {
	handler, err := deliveryhttp.NewHandler()
	if err != nil {
		t.Fatalf("failed to initialize handler: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		url            string
		formData       url.Values
		expectedStatus int
		containsBody   string
	}{
		{
			name:           "Health Check probe",
			method:         "GET",
			url:            "/health",
			formData:       nil,
			expectedStatus: http.StatusOK,
			containsBody:   `{"status":"ok"}`,
		},
		{
			name:           "Assessment Status pipeline",
			method:         "GET",
			url:            "/api/gamification/status",
			formData:       nil,
			expectedStatus: http.StatusOK,
			containsBody:   "Tahap",
		},
		{
			name:   "System Design Defense - Valid architectural combination",
			method: "POST",
			url:    "/api/scenarios/02/defend",
			formData: url.Values{
				"idempotency_storage": {"redis_lock"},
				"ledger_model":        {"double_entry"},
				"failure_handling":    {"circuit_breaker_dlq"},
			},
			expectedStatus: http.StatusOK,
			containsBody:   "APPROVED",
		},
		{
			name:   "Take-Home Repository Submission - Valid GitHub Repo",
			method: "POST",
			url:    "/api/scenarios/03/submit-repo",
			formData: url.Values{
				"repo_url": {"https://github.com/fransalwan/backend-user-test-sandbox"},
				"branch":   {"main"},
			},
			expectedStatus: http.StatusOK,
			containsBody:   "APPROVED",
		},
		{
			name:           "Download Postman Collection JSON",
			method:         "GET",
			url:            "/api/docs/postman",
			expectedStatus: http.StatusOK,
			containsBody:   "Fintech Core Banking & Transfer Sandbox API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.formData != nil {
				req = httptest.NewRequest(tt.method, tt.url, strings.NewReader(tt.formData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}

			w := httptest.NewRecorder()

			switch tt.url {
			case "/health":
				handler.HealthCheck(w, req)
			case "/api/gamification/status":
				handler.GetGamificationStatus(w, req)
			case "/api/scenarios/02/defend":
				handler.DefendScenario02(w, req)
			case "/api/scenarios/03/submit-repo":
				handler.SubmitScenario03Repo(w, req)
			case "/api/docs/postman":
				handler.DownloadPostmanCollection(w, req)
			}

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.containsBody) {
				t.Errorf("expected response to contain %q, but body was: %s", tt.containsBody, body)
			}
		})
	}
}
