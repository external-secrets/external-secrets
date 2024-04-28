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

package conjur

import (
	"fmt"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

func TestValidateStore(t *testing.T) {
	testCases := []ValidateStoreTestCase{
		{
			store: makeAPIKeySecretStore(svcURL, svcUser, svcApikey, svcAccount),
			err:   nil,
		},
		{
			store: makeAPIKeySecretStore("", svcUser, svcApikey, svcAccount),
			err:   fmt.Errorf("conjur URL cannot be empty"),
		},
		{
			store: makeAPIKeySecretStore(svcURL, "", svcApikey, svcAccount),
			err:   fmt.Errorf("missing Auth.Apikey.UserRef"),
		},
		{
			store: makeAPIKeySecretStore(svcURL, svcUser, "", svcAccount),
			err:   fmt.Errorf("missing Auth.Apikey.ApiKeyRef"),
		},
		{
			store: makeAPIKeySecretStore(svcURL, svcUser, svcApikey, ""),
			err:   fmt.Errorf("missing Auth.ApiKey.Account"),
		},

		{
			store: makeJWTSecretStore(svcURL, "conjur", "", jwtAuthnService, "", "myconjuraccount"),
			err:   nil,
		},
		{
			store: makeJWTSecretStore(svcURL, "", jwtSecretName, jwtAuthnService, "", "myconjuraccount"),
			err:   nil,
		},
		{
			store: makeJWTSecretStore(svcURL, "conjur", "", jwtAuthnService, "", ""),
			err:   fmt.Errorf("missing Auth.Jwt.Account"),
		},
		{
			store: makeJWTSecretStore(svcURL, "conjur", "", "", "", "myconjuraccount"),
			err:   fmt.Errorf("missing Auth.Jwt.ServiceID"),
		},
		{
			store: makeJWTSecretStore("", "conjur", "", jwtAuthnService, "", "myconjuraccount"),
			err:   fmt.Errorf("conjur URL cannot be empty"),
		},
		{
			store: makeJWTSecretStore(svcURL, "", "", jwtAuthnService, "", "myconjuraccount"),
			err:   fmt.Errorf("must specify Auth.Jwt.SecretRef or Auth.Jwt.ServiceAccountRef"),
		},

		{
			store: makeNoAuthSecretStore(svcURL),
			err:   fmt.Errorf("missing Auth.* configuration"),
		},
	}
	p := Provider{}
	for _, tc := range testCases {
		_, err := p.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}
}
