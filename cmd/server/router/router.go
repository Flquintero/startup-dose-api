package router

import (
	"net/http"
	"startupdose.com/cmd/server/config"
	"startupdose.com/cmd/server/handler"
	"startupdose.com/cmd/server/middleware"
)

// Setup configures and returns the HTTP router with all routes and middleware
func Setup(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// Create API key authentication middleware
	apiKeyAuth := middleware.APIKeyAuthMiddleware(cfg.APIKey)

	// Register public handlers
	mux.HandleFunc("GET /posts/1", handler.PostsHandler)
	mux.HandleFunc("GET /healthz", handler.HealthzHandler)
	mux.HandleFunc("GET /companies/latest", handler.CompanyLatestHandler)
	mux.HandleFunc("GET /debug/companies", handler.DebugCompaniesHandler)

	// Register protected handlers (require API key)
	mux.HandleFunc("POST /companies/generate", apiKeyAuth(handler.GenerateCompaniesHandler))

	// Wrap with middleware (order matters: outer wraps inner)
	var handlerWrapper http.Handler = mux
	handlerWrapper = middleware.RecoverMiddleware(handlerWrapper)
	handlerWrapper = middleware.CORSMiddleware(handlerWrapper)
	handlerWrapper = middleware.LoggingMiddleware(handlerWrapper)

	return handlerWrapper
}
