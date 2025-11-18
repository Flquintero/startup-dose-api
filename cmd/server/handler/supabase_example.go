package handler

import (
	"encoding/json"
	"net/http"

	"startupdose.com/cmd/server/database"
)

// SupabaseExampleHandler demonstrates how to use Supabase in a handler
// This is a template - modify it based on your actual database schema
func SupabaseExampleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get Supabase client
	client := database.GetClient()
	if client == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "service_unavailable",
			Message: "Database connection not available",
		})
		return
	}

	// Example: Query data from a table
	// Replace 'your_table' with your actual table name
	// var results []YourStruct
	// err := client.DB.From("your_table").Select("*").Execute(&results)
	// if err != nil {
	// 	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	json.NewEncoder(w).Encode(ErrorResponse{
	// 		Error:   "database_error",
	// 		Message: "Failed to query database",
	// 	})
	// 	return
	// }

	// For now, just return a success message
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Supabase is connected and ready to use",
		"status":  "ok",
	})
}
