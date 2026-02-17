package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bcc-code/pdf-service/internal/app"
)

func main() {
	cfg := loadConfig()

	validator, err := app.NewOIDCValidator(context.Background(), cfg.authAuthority, cfg.authIssuer, cfg.authAudiences)
	if err != nil {
		log.Fatalf("failed to initialize authentication: %v", err)
	}

	svc := app.NewService(
		validator,
		app.WeasyprintRunner{BwrapPath: cfg.bwrapPath, WeasyprintPath: cfg.weasyprintPath},
		app.Config{
			RequiredScope:   cfg.requiredScope,
			MaxRequestBytes: cfg.maxRequestBytes,
			MaxPartBytes:    cfg.maxPartBytes,
			RequestTimeout:  cfg.requestTimeout,
			BwrapPath:       cfg.bwrapPath,
			WeasyprintPath:  cfg.weasyprintPath,
		},
	)

	server := &http.Server{
		Addr:              cfg.listenAddress,
		Handler:           svc.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      cfg.requestTimeout + 15*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on %s", cfg.listenAddress)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

type runtimeConfig struct {
	listenAddress   string
	authAuthority   string
	authIssuer      string
	authAudiences   []string
	requiredScope   string
	maxRequestBytes int64
	maxPartBytes    int64
	requestTimeout  time.Duration
	bwrapPath       string
	weasyprintPath  string
}

func loadConfig() runtimeConfig {
	port := getEnv("PORT", "8080")
	authority := mustGetEnv("AUTH_AUTHORITY")
	audiences := splitCommaSeparated(mustGetEnv("AUTH_AUDIENCES"))
	return runtimeConfig{
		listenAddress:   ":" + port,
		authAuthority:   authority,
		authIssuer:      getEnv("AUTH_ISSUER", authority),
		authAudiences:   audiences,
		requiredScope:   getEnv("AUTH_REQUIRED_SCOPE", "pdf#create"),
		maxRequestBytes: getEnvInt64("MAX_REQUEST_BYTES", 104857600),
		maxPartBytes:    getEnvInt64("MAX_PART_BYTES", 52428800),
		requestTimeout:  time.Duration(getEnvInt64("REQUEST_TIMEOUT_SECONDS", 120)) * time.Second,
		bwrapPath:       getEnv("BWRAP_PATH", "/usr/bin/bwrap"),
		weasyprintPath:  getEnv("WEASYPRINT_PATH", "/usr/bin/weasyprint"),
	}
}

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
		log.Fatalf("missing required environment variable %s", name)
	}
	return value
}

func getEnvInt64(name string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		log.Fatalf("invalid value for %s: %q", name, value)
	}
	return parsed
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized := strings.TrimSpace(part)
		if normalized != "" {
			result = append(result, normalized)
		}
	}
	return result
}
