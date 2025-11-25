package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type OrderResponse struct {
	OrderID string  `json:"order_id"`
	Item    string  `json:"item"`
	Price   float64 `json:"price"`
}

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

	// Важно: установка глобального propagator для передачи trace контекста
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}

func getOrderPrice(ctx context.Context, orderID string) (float64, error) {
	tracer := otel.Tracer("service-a")
	var price float64
	err := func(ctx context.Context) error {
		ctx, span := tracer.Start(ctx, "GetPriceFromServiceB")
		defer span.End()

		serviceBURL := os.Getenv("SERVICE_B_URL")
		if serviceBURL == "" {
			serviceBURL = "http://service-b:8080"
		}
		url := fmt.Sprintf("%s/price?order_id=%s", serviceBURL, orderID)

		client := http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			span.RecordError(err)
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			span.RecordError(err)
			return err
		}
		defer resp.Body.Close()

		var priceResp struct {
			OrderID string  `json:"order_id"`
			Price   float64 `json:"price"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
			span.RecordError(err)
			return err
		}
		price = priceResp.Price
		return nil
	}(ctx)
	return price, err
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		http.Error(w, "order_id is required", http.StatusBadRequest)
		return
	}

	price, err := getOrderPrice(ctx, orderID)
	if err != nil {
		http.Error(w, "failed to get price: "+err.Error(), http.StatusInternalServerError)
		return
	}

	orderResp := OrderResponse{
		OrderID: orderID,
		Item:    "Example Item",
		Price:   price,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orderResp)
}

func main() {
	tp, err := initTracer("service-a")
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer tp.Shutdown(context.Background())

	mux := http.NewServeMux()
	mux.Handle("/order", otelhttp.NewHandler(http.HandlerFunc(orderHandler), "OrderHandler"))

	http.ListenAndServe(":8080", mux)
}
