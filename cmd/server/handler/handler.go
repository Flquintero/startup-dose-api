package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"startupdose.com/cmd/server/client"
)

// ErrorResponse represents a JSON http.Error response
type ErrorResponse struct {
	Error   string `json:"Error"`
	Message string `json:"message"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	OK bool `json:"ok"`
}

// PostsHandler handles GET /posts/1
func PostsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Create http.Request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://jsonplaceholder.typicode.com/posts/1", nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_gateway",
			Message: "Failed to create http.Request",
		})
		return
	}

	// Call upstream
	httpClient := client.GetClient()
	resp, err := httpClient.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_gateway",
			Message: "Failed to reach upstream service",
		})
		return
	}
	defer resp.Body.Close()

	// Check upstream status
	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_gateway",
			Message: "Upstream returned non-200 status",
		})
		return
	}

	// Stream upstream response
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Read and forward the full response
	var data interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

// HealthzHandler handles GET /healthz
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{OK: true})
}
