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

package common

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"time"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/endpoint"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// Creates a connection to the given Yandex.Cloud API endpoint.
func NewGrpcConnection(
	ctx context.Context,
	apiEndpoint string,
	apiEndpointID string, // an ID from https://api.cloud.yandex.net/endpoints
	authorizedKey *iamkey.Key,
	caCertificate []byte,
) (*grpc.ClientConn, error) {
	tlsConfig, err := tlsConfig(caCertificate)
	if err != nil {
		return nil, err
	}

	sdk, err := buildSDK(ctx, apiEndpoint, authorizedKey, tlsConfig)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = closeSDK(ctx, sdk)
	}()

	serviceAPIEndpoint, err := sdk.ApiEndpoint().ApiEndpoint().Get(ctx, &endpoint.GetApiEndpointRequest{
		ApiEndpointId: apiEndpointID,
	})
	if err != nil {
		return nil, err
	}

	// Until gRPC proposal A61 is implemented in grpc-go, default gRPC name resolver (dns)
	// is incompatible with dualstack backends, and YC API backends are dualstack.
	// However, if passthrough resolver is used instead, grpc-go won't do any name resolution
	// and will pass the endpoint to net.Dial as-is, which would utilize happy-eyeballs
	// support in Go's net package.
	// So we explicitly set gRPC resolver to `passthrough` to match `ycsdk`s behavior,
	// which uses `passthrough` resolver implicitly by using deprecated grpc.DialContext
	// instead of grpc.NewClient used here
	target := "passthrough:///" + serviceAPIEndpoint.Address
	return grpc.NewClient(target,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Second * 30,
			Timeout:             time.Second * 10,
			PermitWithoutStream: false,
		}),
		grpc.WithUserAgent("external-secrets"),
	)
}

// Exchanges the given authorized key to an IAM token.
func NewIamToken(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (*IamToken, error) {
	tlsConfig, err := tlsConfig(caCertificate)
	if err != nil {
		return nil, err
	}

	sdk, err := buildSDK(ctx, apiEndpoint, authorizedKey, tlsConfig)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = closeSDK(ctx, sdk)
	}()

	iamToken, err := sdk.CreateIAMToken(ctx)
	if err != nil {
		return nil, err
	}

	return &IamToken{Token: iamToken.IamToken, ExpiresAt: iamToken.ExpiresAt.AsTime()}, nil
}

func tlsConfig(caCertificate []byte) (*tls.Config, error) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if caCertificate != nil {
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCertificate)
		if !ok {
			return nil, errors.New("unable to read trusted CA certificates")
		}
		tlsConfig.RootCAs = caCertPool
	}
	return tlsConfig, nil
}

func buildSDK(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, tlsConfig *tls.Config) (*ycsdk.SDK, error) {
	creds, err := ycsdk.ServiceAccountKey(authorizedKey)
	if err != nil {
		return nil, err
	}

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: creds,
		Endpoint:    apiEndpoint,
		TLSConfig:   tlsConfig,
	})
	if err != nil {
		return nil, err
	}

	return sdk, nil
}

func closeSDK(ctx context.Context, sdk *ycsdk.SDK) error {
	return sdk.Shutdown(ctx)
}

type PerRPCCredentials struct {
	IamToken string
}

func (t PerRPCCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"Authorization": "Bearer " + t.IamToken}, nil
}

func (PerRPCCredentials) RequireTransportSecurity() bool {
	return true
}
