package http

import "net/http"

// Handler holds dependencies for delivery layer HTTP routes.
type Handler struct{}

// NewHandler initializes a new HTTP delivery handler.
func NewHandler() *Handler {
	return &Handler{}
}

// HealthCheck provides a standard liveness probe.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
