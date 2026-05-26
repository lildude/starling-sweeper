package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
	"github.com/lildude/starling-sweep/internal/feeditem"
	"github.com/lildude/starling-sweep/internal/ping"
)

func main() {
	port := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		port = ":" + val
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /_ping", ping.Handler)
	mux.HandleFunc("POST /feed-item", feeditem.Handler)

	srv := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("starting server", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	if err := srv.Shutdown(context.Background()); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}
}
