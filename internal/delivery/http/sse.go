package http

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEHub manages active browser client SSE subscriptions.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan string]bool
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan string]bool),
	}
}

func (h *SSEHub) Broadcast(msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
			// Client buffer full; skip to prevent blocking
		}
	}
}

func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Mencegah buffering di Nginx / Cloudflare Tunnel
	w.Header().Set("Access-Control-Allow-Origin", "*")

	msgChan := make(chan string, 100)
	h.mu.Lock()
	h.clients[msgChan] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, msgChan)
		close(msgChan)
		h.mu.Unlock()
	}()

	// Send initial connection event
	_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", "Connected to Fintech Dojo SSE Stream")
	flusher.Flush()

	// Ticker keep-alive untuk mencegah timeout 524 pada Cloudflare Tunnel / Reverse Proxy
	keepAliveTicker := time.NewTicker(15 * time.Second)
	defer keepAliveTicker.Stop()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case <-keepAliveTicker.C:
			// Komentar SSE (diawali titik dua) menjaga koneksi TCP tetap hidup tanpa memicu event listener di browser
			_, _ = fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		case msg, open := <-msgChan:
			if !open {
				return
			}
			_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		}
	}
}
