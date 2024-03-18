package main

import (
	"context"
	"errors"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

func initJaegerTracer() func() {
	tp, err := JaegerTraceProvider()
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("failed to shutdown TracerProvider: %v", err)
		}
	}
}
func JaegerTraceProvider() (*sdktrace.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("manual-service"),
			semconv.DeploymentEnvironmentKey.String("manual-env"),
		)),
	)
	return tp, nil
}

func main() {

	shutdown := initJaegerTracer()
	defer shutdown()

	basicTracer := otel.GetTracerProvider().Tracer("basic-tracer")
	exceptionTracer := otel.GetTracerProvider().Tracer("exception-tracer")
	ctx := context.Background()

	// work begins
	parentFunction(ctx, basicTracer)

	// bonus work
	exceptionFunction(ctx, exceptionTracer)
}

func parentFunction(ctx context.Context, tracer trace.Tracer) {
	ctx, parentSpan := tracer.Start(
		ctx,
		"parentSpanName",
		trace.WithAttributes(attribute.String("parentAttributeKey1", "parentAttributeValue1")))

	parentSpan.AddEvent("ParentSpan-Event")
	log.Printf("In parent span, before calling a child function.")

	defer parentSpan.End()

	childFunction(ctx, tracer)

	log.Printf("In parent span, after calling a child function. When this function ends, parentSpan will complete.")
}

func childFunction(ctx context.Context, tracer trace.Tracer) {
	ctx, childSpan := tracer.Start(
		ctx,
		"childSpanName",
		trace.WithAttributes(attribute.String("childAttributeKey1", "childAttributeValue1")))

	childSpan.AddEvent("ChildSpan-Event")
	defer childSpan.End()

	log.Printf("In child span, when this function returns, childSpan will complete.")
}

func exceptionFunction(ctx context.Context, tracer trace.Tracer) {
	ctx, exceptionSpan := tracer.Start(
		ctx,
		"exceptionSpanName",
		trace.WithAttributes(attribute.String("exceptionAttributeKey1", "exceptionAttributeValue1")))
	defer exceptionSpan.End()
	log.Printf("Call division function.")
	_, err := divide(10, 0)
	if err != nil {
		exceptionSpan.RecordError(err)
		exceptionSpan.SetStatus(codes.Error, err.Error())
	}
}

func divide(x int, y int) (int, error) {
	if y == 0 {
		return -1, errors.New("division by zero")
	}
	return x / y, nil
}
