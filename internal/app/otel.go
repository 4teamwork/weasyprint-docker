package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/detectors/gcp"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type ObservabilityProvider struct {
	serviceName string
	logger      *slog.Logger
	lp          *sdklog.LoggerProvider
	tp          *sdktrace.TracerProvider
	propagator  propagation.TextMapPropagator
}

func (t *ObservabilityProvider) Logger() *slog.Logger {
	return t.logger
}

func (t *ObservabilityProvider) HttpClient(base http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(base,
			otelhttp.WithTracerProvider(t.tp),
			otelhttp.WithPropagators(t.propagator)),
	}
}

func (t *ObservabilityProvider) HttpHandler(operation string, h http.Handler) http.Handler {
	return otelhttp.NewHandler(h, operation,
		otelhttp.WithTracerProvider(t.tp),
		otelhttp.WithPropagators(t.propagator))
}

func (t *ObservabilityProvider) Shutdown() {
	_ = t.lp.Shutdown(context.Background())
	_ = t.tp.Shutdown(context.Background())
}

func NewObservabilityProvider(serviceName string) *ObservabilityProvider {
	ctx := context.Background()

	logExporter, err := otlploggrpc.New(ctx)
	if err != nil {
		panic(fmt.Errorf("otlploggrpc.New: %w", err))
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		panic(fmt.Errorf("otlptracegrpc.New: %w", err))
	}

	res, err := resource.New(ctx,
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
	)

	if err != nil {
		panic(fmt.Errorf("cannot create resource: %w", err))
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	propagator := BCCPropagator{}
	logger := slog.New(otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(lp)))

	return &ObservabilityProvider{
		serviceName: serviceName,
		logger:      logger,
		lp:          lp,
		tp:          tp,
		propagator:  propagator,
	}
}

type BCCPropagator struct{}

var _ propagation.TextMapPropagator = BCCPropagator{}

const bccTracestateKey = "bcc"

var w3cPropagation = propagation.TraceContext{}

func (BCCPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		w3cPropagation.Inject(ctx, carrier)
		return
	}
	traceState := spanContext.TraceState()
	newTraceState, _ := traceState.Insert(bccTracestateKey, spanContext.SpanID().String())
	newSpanContext := spanContext.WithTraceState(newTraceState)
	newCtx := trace.ContextWithSpanContext(ctx, newSpanContext)

	w3cPropagation.Inject(newCtx, carrier)
}

func (BCCPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	extractedCtx := w3cPropagation.Extract(ctx, carrier)
	spanContext := trace.SpanContextFromContext(extractedCtx)
	if !spanContext.IsValid() {
		return extractedCtx
	}
	traceState := spanContext.TraceState()
	newSpanIdKey := traceState.Get(bccTracestateKey)

	newSpanId, err := trace.SpanIDFromHex(newSpanIdKey)
	if err != nil {
		return extractedCtx
	}
	newSpanContext := spanContext.WithSpanID(newSpanId)
	return trace.ContextWithSpanContext(extractedCtx, newSpanContext)
}

func (BCCPropagator) Fields() []string {
	return w3cPropagation.Fields()
}
