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

package fake

import (
	"context"

	akeyless "github.com/akeylesslabs/akeyless-go/v3"
)

type AkeylessMockClient struct {
	getSecret    func(secretName string, version int32) (string, error)
	createSecret func(ctx context.Context, remoteKey, data string) error
	updateSecret func(ctx context.Context, remoteKey, data string) error
	deleteSecret func(ctx context.Context, remoteKey string) error
	describeItem func(ctx context.Context, itemName string) (*akeyless.Item, error)
}

func New() *AkeylessMockClient {
	return &AkeylessMockClient{}
}

func (mc *AkeylessMockClient) SetGetSecretFn(f func(secretName string, version int32) (string, error)) *AkeylessMockClient {
	mc.getSecret = f
	return mc
}

func (mc *AkeylessMockClient) SetCreateSecretFn(f func(ctx context.Context, remoteKey, data string) error) *AkeylessMockClient {
	mc.createSecret = f
	return mc
}

func (mc *AkeylessMockClient) SetUpdateSecretFn(f func(ctx context.Context, remoteKey, data string) error) *AkeylessMockClient {
	mc.updateSecret = f
	return mc
}

func (mc *AkeylessMockClient) SetDeleteSecretFn(f func(ctx context.Context, remoteKey string) error) *AkeylessMockClient {
	mc.deleteSecret = f
	return mc
}

func (mc *AkeylessMockClient) SetDescribeItemFn(f func(ctx context.Context, itemName string) (*akeyless.Item, error)) *AkeylessMockClient {
	mc.describeItem = f
	return mc
}

func (mc *AkeylessMockClient) CreateSecret(ctx context.Context, remoteKey, data string) error {
	return mc.createSecret(ctx, remoteKey, data)
}

func (mc *AkeylessMockClient) DeleteSecret(ctx context.Context, remoteKey string) error {
	return mc.deleteSecret(ctx, remoteKey)
}

func (mc *AkeylessMockClient) DescribeItem(ctx context.Context, itemName string) (*akeyless.Item, error) {
	return mc.describeItem(ctx, itemName)
}

func (mc *AkeylessMockClient) UpdateSecret(ctx context.Context, remoteKey, data string) error {
	return mc.updateSecret(ctx, remoteKey, data)
}

func (mc *AkeylessMockClient) TokenFromSecretRef(_ context.Context) (string, error) {
	return "newToken", nil
}

func (mc *AkeylessMockClient) GetSecretByType(_ context.Context, secretName string, version int32) (string, error) {
	return mc.getSecret(secretName, version)
}

func (mc *AkeylessMockClient) ListSecrets(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

func (mc *AkeylessMockClient) WithValue(_ *Input, out *Output) {
	if mc != nil {
		mc.getSecret = func(secretName string, version int32) (string, error) {
			return out.Value, out.Err
		}
	}
}

type Input struct {
	SecretName string
	Token      string
	Version    int32
}

type Output struct {
	Value string
	Err   error
}
