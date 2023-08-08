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
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	ctrl "sigs.k8s.io/controller-runtime"
)
var (
	setupLog    = ctrl.Log.WithName("tracing")
	clientTrace otlptrace.Client
)
func TraceClient(provider string, collectorURL string)  (func(), error) {
	ctx := context.Background()

	clientTrace = otlpgrpc.NewClient(
		otlpgrpc.WithEndpoint(collectorURL),
		otlpgrpc.WithInsecure(),
	)
	newExporter, err := otlptrace.New(ctx, clientTrace)
	if err != nil {
		setupLog.Error(err, "OpenTelemetry Collector Exporter has not been created")
		return nil, err	
	}
	newResource, err := resource.New(
		ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(provider),
		), 
		resource.WithFromEnv(),
	)
	if err != nil {
		setupLog.Error(err, "Tracing resource has not been created")
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(newExporter),
		trace.WithResource(newResource),
	)

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil

}
