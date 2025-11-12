package router

import (
	"net/http"
	"startupdose.com/cmd/server/handler"
	"startupdose.com/cmd/server/middleware"
)

// Setup configures and returns the HTTP router with all routes and middleware
func Setup() http.Handler {
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("GET /posts/1", handler.PostsHandler)
	mux.HandleFunc("GET /healthz", handler.HealthzHandler)

	// Wrap with middleware (order matters: outer wraps inner)
	var handlerWrapper http.Handler = mux
	handlerWrapper = middleware.RecoverMiddleware(handlerWrapper)
	handlerWrapper = middleware.CORSMiddleware(handlerWrapper)
	handlerWrapper = middleware.LoggingMiddleware(handlerWrapper)

	return handlerWrapper
}
