package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bcc-code/pdf-service/internal/app"
)

func main() {
	port := getEnv("PORT", "8080")
	authority := mustGetEnv("AUTH_AUTHORITY")
	audience := mustGetEnv("AUTH_AUDIENCE")
	otelServiceName := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME"))

	obs := app.Observability(app.NewMockObservabilityProvider())

	if otelServiceName != "" {
		obs = app.NewObservabilityProvider(otelServiceName)
	}
	defer obs.Shutdown()
	logger := obs.Logger()

	validator, err := app.NewOIDCValidator(context.Background(), authority, audience, defaultRequiredScope, obs)
	if err != nil {
		log.Fatalf("failed to initialize authentication: %s", err)
	}

	svc := app.NewService(
		validator,
		app.WeasyprintRunner{BwrapPath: defaultBwrapPath, WeasyprintPath: defaultWeasyprintPath, DefaultStylesheetPath: defaultStylesheetPath},
		app.Config{
			MaxRequestBytes: defaultMaxRequestBytes,
			RequestTimeout:  defaultRequestTimeout,
		},
		obs,
	)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           obs.HttpHandler(otelServiceName, svc.Routes()),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      defaultRequestTimeout + 15*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("service starting", "listen_address", ":"+port, "authority", authority, "audience", audience)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %s", err)
	}
}

const (
	defaultRequiredScope   = "pdf#create"
	defaultMaxRequestBytes = int64(104857600)
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
		_, _ = os.Stderr.WriteString("missing required environment variable: " + name + "\n")
		os.Exit(1)
	}
	return value
}
