package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bcc-code/pdf-service/internal/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := getEnv("PORT", "8080")
	authority := mustGetEnv("AUTH_AUTHORITY")
	audience := mustGetEnv("AUTH_AUDIENCE")

	validator, err := app.NewOIDCValidator(context.Background(), authority, audience, defaultRequiredScope)
	if err != nil {
		slog.Error("failed to initialize authentication", "error", err)
		os.Exit(1)
	}

	svc := app.NewService(
		validator,
		app.WeasyprintRunner{BwrapPath: defaultBwrapPath, WeasyprintPath: defaultWeasyprintPath, DefaultStylesheetPath: defaultStylesheetPath},
		app.Config{
			MaxRequestBytes: defaultMaxRequestBytes,
			MaxPartBytes:    defaultMaxPartBytes,
			RequestTimeout:  defaultRequestTimeout,
		},
	)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           svc.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      defaultRequestTimeout + 15*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("service starting", "listen_address", ":"+port, "authority", authority, "audience", audience)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

const (
	defaultRequiredScope   = "pdf#create"
	defaultMaxRequestBytes = int64(104857600)
	defaultMaxPartBytes    = int64(52428800)
	defaultRequestTimeout  = 120 * time.Second
	defaultBwrapPath       = "bwrap"
	defaultWeasyprintPath  = "weasyprint"
	defaultStylesheetPath  = "assets/default.css"
)

func getEnv(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func mustGetEnv(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		slog.Error("missing required environment variable", "name", name)
		os.Exit(1)
	}
	return value
}
