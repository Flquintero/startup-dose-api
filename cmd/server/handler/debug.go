package handler

import (
	"encoding/json"
	"net/http"

	"startupdose.com/cmd/server/database"
)

// DebugCompaniesHandler helps debug the companies table query
func DebugCompaniesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	client := database.GetClient()
	if client == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Database client not initialized",
		})
		return
	}

	// Try to get all companies without ordering
	var companies []map[string]interface{}
	_, err := client.
		From("companies").
		Select("*", "", false).
		ExecuteTo(&companies)

	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Query failed",
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":     len(companies),
		"companies": companies,
	})
}
