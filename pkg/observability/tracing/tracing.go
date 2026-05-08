package tracing

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config holds minimal configuration for setting up tracing
type Config struct {
	Enabled       bool
	Endpoint      string
	ServiceName   string
	SamplingRatio float64
	Version       string
	Commit        string
}

// Init configures global OpenTelemetry tracing when enabled.
// It returns a shutdown function that should be called on service stop.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	// Exporter (OTLP over HTTP)
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}
	// Assume insecure unless explicitly using https in endpoint string
	if !strings.HasPrefix(cfg.Endpoint, "https://") {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Resource with service name and build metadata
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
			attribute.String("service.commit", cfg.Commit),
		),
	)
	if err != nil {
		return nil, err
	}

	// Tracer provider with parent-based ratio sampler
	sam := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplingRatio))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sam),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	return tp.Shutdown, nil
}
