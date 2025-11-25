package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func initTracer(serviceName string) (*sdktrace.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://simplest-collector:14268/api/traces")))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", "dev"),
		)),
	)
	otel.SetTracerProvider(tp)

	// Глобальный propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}

func priceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer("service-b")

	ctx, span := tracer.Start(ctx, "CalculatePrice")
	defer span.End()

	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		http.Error(w, "order_id is required", http.StatusBadRequest)
		return
	}

	price := 42.0
	span.SetAttributes(attribute.String("order_id", orderID))
	span.AddEvent("Price calculated")

	resp := struct {
		OrderID string  `json:"order_id"`
		Price   float64 `json:"price"`
	}{
		OrderID: orderID,
		Price:   price,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	tp, err := initTracer("service-b")
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer tp.Shutdown(context.Background())

	mux := http.NewServeMux()
	mux.Handle("/price", otelhttp.NewHandler(http.HandlerFunc(priceHandler), "PriceHandler"))

	http.ListenAndServe(":8080", mux)
}
