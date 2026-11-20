package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"smoi/internal/api"
	"smoi/internal/auth"
	"smoi/internal/database"
	"smoi/internal/storage"
	"smoi/internal/upstream"
)

func environmentValue(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if exists {
		return value
	}
	return fallback
}

func positiveEnvironmentNumber(key string, fallback int) int {
	value, err := strconv.Atoi(environmentValue(key, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func nonNegativeEnvironmentNumber(key string, fallback int) int {
	value, err := strconv.Atoi(environmentValue(key, ""))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func main() {
	configureLogging()

	server, err := initializeServer()
	if err != nil {
		slog.Error("SMOI initialization failed", "error", err)
		os.Exit(1)
	}
	defer server.DB.Close()

	server.Housekeeping(context.Background())
	server.Jobs.Start()
	go runHousekeeping(server)

	router := http.NewServeMux()
	server.RegisterRoutes(router)
	if err := registerFrontendRoutes(router); err != nil {
		slog.Error("embedded assets unavailable", "error", err)
		os.Exit(1)
	}

	address := environmentValue("SMOI_HOST", "0.0.0.0") + ":" + environmentValue("SMOI_PORT", "8080")
	httpServer := &http.Server{
		Addr:              address,
		Handler:           requestLogger(router),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	slog.Info("SMOI listening", "address", address)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}

func configureLogging() {
	level := slog.LevelInfo
	switch strings.ToLower(environmentValue("SMOI_LOG_LEVEL", "info")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
}

func initializeServer() (*api.Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dataDirectory := environmentValue("SMOI_DATA_DIR", "/data")
	store, err := storage.New(dataDirectory)
	if err != nil {
		return nil, err
	}

	databasePath := environmentValue("SMOI_DB_PATH", filepath.Join(dataDirectory, "database", "smoi.db"))
	if err := prepareDatabaseDirectory(dataDirectory, databasePath); err != nil {
		return nil, err
	}

	db, err := database.Open(databasePath)
	if err != nil {
		return nil, err
	}

	identity, err := auth.New(
		ctx, db,
		environmentValue("OIDC_ISSUER_URL", ""),
		environmentValue("OIDC_CLIENT_ID", ""),
		environmentValue("OIDC_CLIENT_SECRET", ""),
		environmentValue("OIDC_REDIRECT_URL", ""),
		positiveEnvironmentNumber("SMOI_SESSION_DURATION_HOURS", 168),
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	upstreamClient, err := upstream.New(
		environmentValue("OPENAI_BASE_URL", ""),
		environmentValue("OPENAI_API_KEY", ""),
		environmentValue("OPENAI_DEFAULT_MODEL", ""),
		time.Duration(nonNegativeEnvironmentNumber("OPENAI_TIMEOUT_SECONDS", 0))*time.Second,
		store,
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	server := &api.Server{
		DB:        db,
		Auth:      identity,
		Storage:   store,
		Upstream:  upstreamClient,
		MaxUpload: int64(positiveEnvironmentNumber("SMOI_MAX_UPLOAD_MB", 25)) * 1024 * 1024,
	}
	server.Jobs = api.NewJobManager(server)
	return server, nil
}

func prepareDatabaseDirectory(dataDirectory, databasePath string) error {
	relative, err := filepath.Rel(dataDirectory, databasePath)
	if err != nil || !filepath.IsLocal(relative) {
		return errors.New("SMOI_DB_PATH must be within SMOI_DATA_DIR")
	}
	return os.MkdirAll(filepath.Dir(databasePath), 0700)
}

func runHousekeeping(server *api.Server) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		server.Housekeeping(ctx)
		cancel()
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		slog.Debug("HTTP request received", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
		slog.Debug("HTTP request finished", "method", r.Method, "path", r.URL.Path,
			"duration", time.Since(started).String())
	})
}
