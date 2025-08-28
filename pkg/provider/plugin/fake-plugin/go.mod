module fake-plugin

go 1.24.6

require (
	github.com/external-secrets/external-secrets v0.0.0
	google.golang.org/grpc v1.74.2
)

require (
	go.opentelemetry.io/otel/sdk/metric v1.37.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250826171959-ef028d996bc1 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

replace github.com/external-secrets/external-secrets => ../../../..
