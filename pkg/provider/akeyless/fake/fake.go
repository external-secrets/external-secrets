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
)

type AkeylessMockClient struct {
	getSecret func(secretName, token string, version int32) (string, error)
}

func (mc *AkeylessMockClient) TokenFromSecretRef(_ context.Context) (string, error) {
	return "newToken", nil
}

func (mc *AkeylessMockClient) GetSecretByType(_ context.Context, secretName, token string, version int32) (string, error) {
	return mc.getSecret(secretName, token, version)
}

func (mc *AkeylessMockClient) ListSecrets(_ context.Context, _, _, _ string) ([]string, error) {
	return nil, nil
}

func (mc *AkeylessMockClient) WithValue(_ *Input, out *Output) {
	if mc != nil {
		mc.getSecret = func(secretName, token string, version int32) (string, error) {
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
