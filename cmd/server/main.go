package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/korjavin/fastfooddb/internal/api"
	"github.com/korjavin/fastfooddb/internal/auth"
	"github.com/korjavin/fastfooddb/internal/metrics"
	"github.com/korjavin/fastfooddb/internal/middleware"
	"github.com/korjavin/fastfooddb/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		slog.Error("DATA_DIR environment variable is required")
		os.Exit(1)
	}

	apiKeys := auth.ParseAPIKeys(os.Getenv("API_KEYS"))
	if len(apiKeys) == 0 {
		slog.Warn("API_KEYS not set — all requests will be accepted without authentication")
	}

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "*"
	}

	slog.Info("opening store", "data_dir", dataDir)
	s, err := store.OpenReadOnly(dataDir)
	if err != nil {
		slog.Error("failed to open store", "error", err)
		os.Exit(1)
	}
	defer s.Close()

	manifest, err := store.ReadManifest(dataDir)
	if err != nil {
		slog.Warn("manifest not found or unreadable", "error", err)
		manifest = nil
	} else {
		slog.Info("manifest loaded",
			"schema_version", manifest.SchemaVersion,
			"product_count", manifest.ProductCount,
			"build_time", manifest.BuildTime,
		)
	}

	reg := metrics.NewRegistry()
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, apiKeys, s, manifest, reg)

	// Middleware chain (outer to inner): Logging → CORS → RateLimit → mux
	handler := middleware.Chain(
		mux,
		middleware.Logging(logger),
		middleware.CORS(corsOrigins),
		middleware.RateLimit(100, 20),
	)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited")
}
