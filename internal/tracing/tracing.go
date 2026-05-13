package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Init sets up OpenTelemetry tracing and returns a shutdown function.
// Call shutdown on app exit to flush pending spans.
func Init(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
	// exporter sends spans to Jaeger over gRPC
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(), // no TLS for local dev
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// resource = metadata about this service (appears in Jaeger UI)
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)

	// tracer provider — manages span creation and export
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter), // batch spans before sending
		sdktrace.WithResource(res),
	)

	// register as global provider — otel.Tracer() anywhere in the app uses this
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
