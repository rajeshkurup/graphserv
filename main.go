/**
 * @file main.go
 * @brief graphserv process entry: configuration, Neo4j graph store, REST API server, and graceful shutdown.
 * @auther rajeshkurup@live.com
 */
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"graphserv/internal/api"
	"graphserv/internal/config"
	"graphserv/internal/graphstore"

	_ "graphserv/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title graphserv API
// @version 1.0
// @description REST API for Neo4j topology graph — nodes, relationships, anomalies, root cause analysis, and blast radius impact.
// @contact.name Rajesh Kurup
// @contact.email rajeshkurup@live.com
// @host localhost:8080
// @BasePath /

/**
 * @brief Loads environment configuration, opens Neo4j, registers HTTP routes, serves until SIGINT/SIGTERM, then shuts down the server.
 * @return Does not return on success (blocks until signal); exits the process via log.Fatal on fatal errors.
 */
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	store, err := graphstore.New(ctx, cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword, cfg.Neo4jDatabase)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = store.Close(context.Background())
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	(&api.Server{Store: store}).Register(mux)
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("graphserv listening on %s (Neo4j %s)", cfg.HTTPAddr, cfg.Neo4jURI)
		log.Printf("Swagger UI at http://localhost%s/swagger/index.html", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

/**
 * @brief Wraps an HTTP handler to log method, path, and elapsed time after each request.
 * @param next the inner HTTP handler to invoke.
 * @return An http.Handler that performs request logging around next.
 */
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// healthHandler godoc
// @Summary Health check
// @Description Returns service health status
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

/**
 * @brief Serializes v as JSON with the given HTTP status code.
 * @param w HTTP response writer.
 * @param status HTTP status code to send before the body.
 * @param v value to encode as JSON (any type).
 * @return None; logs encode errors only.
 */
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode json: %v", err)
	}
}
