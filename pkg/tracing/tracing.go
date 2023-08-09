/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tracing

import (
	"context"
	"crypto/x509"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	//DefaultProviderName is default name used for tracing.
	DefaultProviderName = "external-secrets"
)

var (
	tracer       trace.Tracer
	otelProvider trace.TracerProvider
	setupLog     = ctrl.Log.WithName("tracing")
	clientTrace  otlptrace.Client
	TracerName   = "external-secrets"
)

func init() {
	// otelProvider is the definition for the OpenTelemetry Provider created for tracing.
	// Provider is the definition for the providers used onto ExternalSecrets.
	otelProvider = trace.NewNoopTracerProvider()
	tracer = otelProvider.Tracer(TracerName)
}

func Tracer() trace.Tracer {
	return tracer
}

// NewTracerProvider creates a new trace provider with the given options.
func NewTraceProvider(provider, namespace, caCert, collectorURL string, sampleRate float64, fallbackToNoOpTracer bool) (err error) {
	ctx := context.Background()

	defer func(p trace.TracerProvider) {
		if err != nil && !fallbackToNoOpTracer {
			return
		}
		if err != nil && fallbackToNoOpTracer {
			setupLog.Error(err, "Error enabling trace provider for OpenTelemetry")
			err = nil
			otelProvider = trace.NewNoopTracerProvider()
		}
		otel.SetTextMapPropagator(propagation.TraceContext{})
		otel.SetTracerProvider(otelProvider)
		otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
			setupLog.Error(err, "Error handling connection with OpenTelemetry Collector")
		}))
		tracer = otel.GetTracerProvider().Tracer(TracerName)
	}(otelProvider)

	var data []byte
	if caCert != "" {
		data, err = os.ReadFile(caCert)
		if err != nil {
			setupLog.Error(err, "Ran into an error during CA Certificate extraction")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(data) {
			setupLog.Error(err, "failed to create cert pool using the ca certificate provided.")
			return err
		}
		setupLog.Info("Enabling gRPC trace channel in secure TLS mode.")
		clientTrace = otlpgrpc.NewClient(
			otlpgrpc.WithEndpoint(collectorURL),
			otlpgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(pool, "")),
		)
	} else {
		clientTrace = otlpgrpc.NewClient(
			otlpgrpc.WithEndpoint(collectorURL),
			otlpgrpc.WithInsecure(),
		)
	}

	newExporter, err := otlptrace.New(ctx, clientTrace)
	if err != nil {
		setupLog.Error(err, "OpenTelemetry Collector Exporter has not been created")
		return err
	}
	if provider == "" {
		provider = DefaultProviderName
	}

	newResource, err := resource.New(
		ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(provider),
			semconv.ServiceNamespaceKey.String(namespace),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
	)
	if err != nil {
		setupLog.Error(err, "Tracing resource has not been created")
	}

	spanProcessor := sdktrace.NewBatchSpanProcessor(newExporter)
	otelProvider = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRate))),
		sdktrace.WithSpanProcessor(spanProcessor),
		sdktrace.WithResource(newResource),
	)
	setupLog.Info("Trace Provider has been created")
	return
}

func Shutdown(ctx context.Context) error {
	tp, ok := otelProvider.(*sdktrace.TracerProvider)
	if !ok {
		return nil
	}
	if err := tp.Shutdown(ctx); err != nil {
		otel.Handle(err)
		setupLog.Error(err, "failed to shutdown the trace exporter")
		return err
	}
	return nil
}
