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

	secrets "github.com/oracle/oci-go-sdk/v45/secrets"
)

type OracleMockClient struct {
	getSecret func(ctx context.Context, request secrets.GetSecretBundleRequest) (response secrets.GetSecretBundleResponse, err error)
}

func (mc *OracleMockClient) GetSecretBundle(ctx context.Context, request secrets.GetSecretBundleRequest) (response secrets.GetSecretBundleResponse, err error) {
	return mc.getSecret(ctx, request)
}

func (mc *OracleMockClient) WithValue(input secrets.GetSecretBundleRequest, output secrets.GetSecretBundleResponse, err error) {
	if mc != nil {
		mc.getSecret = func(ctx context.Context, paramReq secrets.GetSecretBundleRequest) (secrets.GetSecretBundleResponse, error) {
			return output, err
		}
	}
}
