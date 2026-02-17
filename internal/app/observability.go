package app

import (
	"log/slog"
	"net/http"
	"os"
)

type Observability interface {
	Logger() *slog.Logger
	HttpClient(base http.RoundTripper) *http.Client
	HttpHandler(operation string, handler http.Handler) http.Handler
	Shutdown()
}

type MockObservabilityProvider struct {
	logger *slog.Logger
}

func NewMockObservabilityProvider() *MockObservabilityProvider {
	return &MockObservabilityProvider{
		logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
}

func (m *MockObservabilityProvider) Logger() *slog.Logger {
	return m.logger
}

func (m *MockObservabilityProvider) HttpClient(base http.RoundTripper) *http.Client {
	return &http.Client{Transport: base}
}

func (m *MockObservabilityProvider) HttpHandler(_ string, handler http.Handler) http.Handler {
	return handler
}

func (m *MockObservabilityProvider) Shutdown() {}
