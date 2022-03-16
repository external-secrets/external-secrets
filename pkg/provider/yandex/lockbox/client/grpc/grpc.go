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
package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"time"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/endpoint"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client"
)

// Implementation of YandexCloudCreator.
type YandexCloudCreator struct {
}

func (lb *YandexCloudCreator) CreateLockboxClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (client.LockboxClient, error) {
	sdk, err := buildSDK(ctx, apiEndpoint, authorizedKey)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = closeSDK(ctx, sdk)
	}()

	payloadAPIEndpoint, err := sdk.ApiEndpoint().ApiEndpoint().Get(ctx, &endpoint.GetApiEndpointRequest{
		ApiEndpointId: "lockbox-payload", // the ID from https://api.cloud.yandex.net/endpoints
	})
	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{MinVersion: tls.VersionTLS12}

	if caCertificate != nil {
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCertificate)
		if !ok {
			return nil, errors.New("unable to read certificate from PEM file")
		}
		tlsConfig.RootCAs = caCertPool
	}

	conn, err := grpc.Dial(payloadAPIEndpoint.Address,
		grpc.WithTransportCredentials(credentials.NewTLS(&tlsConfig)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Second * 30,
			Timeout:             time.Second * 10,
			PermitWithoutStream: false,
		}),
		grpc.WithUserAgent("external-secrets"),
	)
	if err != nil {
		return nil, err
	}

	return &LockboxClient{lockbox.NewPayloadServiceClient(conn)}, nil
}

func (lb *YandexCloudCreator) CreateIamToken(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key) (*client.IamToken, error) {
	sdk, err := buildSDK(ctx, apiEndpoint, authorizedKey)
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

	return &client.IamToken{Token: iamToken.IamToken, ExpiresAt: iamToken.ExpiresAt.AsTime()}, nil
}

func (lb *YandexCloudCreator) Now() time.Time {
	return time.Now()
}

func buildSDK(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key) (*ycsdk.SDK, error) {
	creds, err := ycsdk.ServiceAccountKey(authorizedKey)
	if err != nil {
		return nil, err
	}

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: creds,
		Endpoint:    apiEndpoint,
	})
	if err != nil {
		return nil, err
	}

	return sdk, nil
}

func closeSDK(ctx context.Context, sdk *ycsdk.SDK) error {
	return sdk.Shutdown(ctx)
}

// Implementation of LockboxClient.
type LockboxClient struct {
	lockboxPayloadClient lockbox.PayloadServiceClient
}

func (lc *LockboxClient) GetPayloadEntries(ctx context.Context, iamToken, secretID, versionID string) ([]*lockbox.Payload_Entry, error) {
	payload, err := lc.lockboxPayloadClient.Get(
		ctx,
		&lockbox.GetPayloadRequest{
			SecretId:  secretID,
			VersionId: versionID,
		},
		grpc.PerRPCCredentials(perRPCCredentials{iamToken: iamToken}),
	)
	if err != nil {
		return nil, err
	}
	return payload.Entries, nil
}

type perRPCCredentials struct {
	iamToken string
}

func (t perRPCCredentials) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{"Authorization": "Bearer " + t.iamToken}, nil
}

func (perRPCCredentials) RequireTransportSecurity() bool {
	return true
}
