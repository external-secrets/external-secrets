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

	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client"
)

// Implementation of LockboxClientCreator.
type LockboxClientCreator struct {
}

func (lb *LockboxClientCreator) Create(ctx context.Context, endpoint string, authorizedKey *iamkey.Key) (client.LockboxClient, error) {
	credentials, err := ycsdk.ServiceAccountKey(authorizedKey)
	if err != nil {
		return nil, err
	}

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: credentials,
		Endpoint:    endpoint,
	})
	if err != nil {
		return nil, err
	}

	return &LockboxClient{sdk}, nil
}

// Implementation of LockboxClient.
type LockboxClient struct {
	sdk *ycsdk.SDK
}

func (lb *LockboxClient) GetPayloadEntries(ctx context.Context, secretID, versionID string) ([]*lockbox.Payload_Entry, error) {
	payload, err := lb.sdk.LockboxPayload().Payload().Get(ctx, &lockbox.GetPayloadRequest{
		SecretId:  secretID,
		VersionId: versionID,
	})
	if err != nil {
		return nil, err
	}
	return payload.Entries, nil
}

func (lb *LockboxClient) Close(ctx context.Context) error {
	err := lb.sdk.Shutdown(ctx)
	if err != nil {
		return err
	}
	return nil
}
