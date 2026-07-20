package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// httpRequestsCounter é a métrica customizada exposta ao Prometheus
// (via OTel Collector -> prometheusremotewrite) com o nome
// "togglemaster_http_requests_total", usada no dashboard do Grafana.
var httpRequestsCounter metric.Int64Counter

// initOTel configura os providers de Trace e Métricas do OpenTelemetry,
// exportando via OTLP/gRPC para o OTel Collector (endpoint definido pelas
// variáveis padrão OTEL_EXPORTER_OTLP_ENDPOINT / OTEL_SERVICE_NAME, já
// setadas no Deployment do Kubernetes). Retorna uma função de shutdown.
func initOTel(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, err
	}

	// --- Traces ---
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// --- Métricas ---
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	meter := otel.Meter(serviceName)
	httpRequestsCounter, err = meter.Int64Counter(
		"togglemaster_http_requests_total",
		metric.WithDescription("Total de requisições HTTP recebidas, por método/rota/status"),
	)
	if err != nil {
		return nil, err
	}

	log.Printf("OpenTelemetry inicializado para o serviço '%s'", serviceName)

	return func(shutdownCtx context.Context) error {
		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return meterProvider.Shutdown(shutdownCtx)
	}, nil
}

// statusRecorder captura o status code da resposta para alimentar a métrica
// customizada de requisições HTTP.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// withMetrics incrementa o contador togglemaster_http_requests_total a cada
// requisição, com os atributos service_name, http_method, http_route e
// http_status_code (usados no dashboard "ToggleMaster - Overview").
func withMetrics(serviceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		if httpRequestsCounter != nil {
			httpRequestsCounter.Add(r.Context(), 1, metric.WithAttributes(
				attribute.String("service_name", serviceName),
				attribute.String("http_method", r.Method),
				attribute.String("http_route", r.URL.Path),
				attribute.String("http_status_code", strconv.Itoa(rec.status)),
			))
		}
	})
}

// instrumentHandler encapsula o handler HTTP raiz com o middleware de
// tracing automático do OTel (otelhttp) + a métrica customizada acima.
func instrumentHandler(serviceName string, next http.Handler) http.Handler {
	return otelhttp.NewHandler(withMetrics(serviceName, next), serviceName)
}

// mapCarrier adapta um map[string]string para o propagation.TextMapCarrier,
// usado para injetar/extrair o contexto de trace em mensagens de fila
// (Service Bus / SQS), já que elas não têm "headers HTTP".
type mapCarrier map[string]string

func (c mapCarrier) Get(key string) string { return c[key] }
func (c mapCarrier) Set(key, value string) { c[key] = value }
func (c mapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// injectTraceContext serializa o SpanContext ativo em ctx como um conjunto
// de propriedades de mensagem (padrão W3C traceparent/tracestate), para que
// o consumidor (analytics-service) reconecte o span ao mesmo trace
// distribuído no APM.
func injectTraceContext(ctx context.Context) map[string]string {
	carrier := mapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier
}

// instrumentedHTTPClient devolve um *http.Client cujo Transport propaga o
// contexto de trace (traceparent) nas chamadas de saída para outros
// microsserviços (flag-service, targeting-service), permitindo que o APM
// monte o Distributed Trace / Service Map completo.
func instrumentedHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
}
