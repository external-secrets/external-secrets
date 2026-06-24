/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conjur

import (
	"errors"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type ValidateStoreTestCase struct {
	store *esv1.SecretStore
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
			err:   errors.New("conjur URL cannot be empty"),
		},
		{
			store: makeAPIKeySecretStore(svcURL, "", svcApikey, svcAccount),
			err:   errors.New("missing Auth.Apikey.UserRef"),
		},
		{
			store: makeAPIKeySecretStore(svcURL, svcUser, "", svcAccount),
			err:   errors.New("missing Auth.Apikey.ApiKeyRef"),
		},
		{
			store: makeAPIKeySecretStore(svcURL, svcUser, svcApikey, ""),
			err:   errors.New("missing Auth.ApiKey.Account"),
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
			err:   errors.New("missing Auth.Jwt.Account"),
		},
		{
			store: makeJWTSecretStore(svcURL, "conjur", "", "", "", "myconjuraccount"),
			err:   errors.New("missing Auth.Jwt.ServiceID"),
		},
		{
			store: makeJWTSecretStore("", "conjur", "", jwtAuthnService, "", "myconjuraccount"),
			err:   errors.New("conjur URL cannot be empty"),
		},
		{
			store: makeJWTSecretStore(svcURL, "", "", jwtAuthnService, "", "myconjuraccount"),
			err:   errors.New("must specify Auth.Jwt.SecretRef or Auth.Jwt.ServiceAccountRef"),
		},

		{
			store: makeCertSecretStore(svcURL, certServiceID, "", svcAccount),
			err:   nil,
		},
		{
			store: makeCertSecretStore(svcURL, certServiceID, "myhostid", svcAccount),
			err:   nil,
		},
		{
			store: makeCertSecretStore("", certServiceID, "", svcAccount),
			err:   errors.New("conjur URL cannot be empty"),
		},
		{
			store: makeCertSecretStore(svcURL, "", "", svcAccount),
			err:   errors.New("missing Auth.Cert.ServiceID"),
		},
		{
			store: makeCertSecretStore(svcURL, certServiceID, "", ""),
			err:   errors.New("missing Auth.Cert.Account"),
		},
		{
			store: makeCertSecretStoreWithMissingRefs(svcURL, certServiceID, svcAccount, true, false),
			err:   errors.New("missing Auth.Cert.ClientKeyRef"),
		},
		{
			store: makeCertSecretStoreWithMissingRefs(svcURL, certServiceID, svcAccount, false, true),
			err:   errors.New("missing Auth.Cert.ClientCertRef"),
		},
		{
			store: makeCertSecretStoreWithEmptyRefNames(svcURL, certServiceID, svcAccount, true, false),
			err:   errors.New("missing Auth.Cert.ClientCertRef.Name"),
		},
		{
			store: makeCertSecretStoreWithEmptyRefNames(svcURL, certServiceID, svcAccount, false, true),
			err:   errors.New("missing Auth.Cert.ClientKeyRef.Name"),
		},
		{
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Conjur: &esv1.ConjurProvider{
							URL:  svcURL,
							Auth: esv1.ConjurAuth{},
						},
					},
				},
			},
			err: errors.New("must specify exactly one Auth.* method"),
		},
		{
			store: makeMultiAuthSecretStore(svcURL),
			err:   errors.New("must specify exactly one Auth.* method"),
		},

		{
			store: makeIAMSecretStore(svcURL, "myorg", "prod", "data/myapp/123456789/MyRole", false),
			err:   nil,
		},
		{
			store: makeIAMSecretStore(svcURL, "myorg", "prod", "data/myapp/123456789/MyRole", true),
			err:   nil,
		},
		{
			store: makeIAMSecretStore(svcURL, "", "prod", "data/myapp/123456789/MyRole", false),
			err:   errors.New("missing Auth.Iam.Account"),
		},
		{
			store: makeIAMSecretStore(svcURL, "myorg", "", "data/myapp/123456789/MyRole", false),
			err:   errors.New("missing Auth.Iam.ServiceID"),
		},
		{
			store: makeIAMSecretStore(svcURL, "myorg", "prod", "", false),
			err:   errors.New("missing Auth.Iam.HostID"),
		},
		{
			store: makeIAMSecretStore("", "myorg", "prod", "data/myapp/123456789/MyRole", false),
			err:   errors.New("conjur URL cannot be empty"),
		},

		{
			store: makeAzureSecretStore(svcURL, "myorg", "prod", "data/myapp/myhost", false),
			err:   nil,
		},
		{
			store: makeAzureSecretStore(svcURL, "myorg", "prod", "data/myapp/myhost", true),
			err:   nil,
		},
		{
			store: makeAzureSecretStore(svcURL, "", "prod", "data/myapp/myhost", false),
			err:   errors.New("missing Auth.Azure.Account"),
		},
		{
			store: makeAzureSecretStore(svcURL, "myorg", "", "data/myapp/myhost", false),
			err:   errors.New("missing Auth.Azure.ServiceID"),
		},
		{
			store: makeAzureSecretStore(svcURL, "myorg", "prod", "", false),
			err:   errors.New("missing Auth.Azure.HostID"),
		},
		{
			store: makeAzureSecretStore("", "myorg", "prod", "data/myapp/myhost", false),
			err:   errors.New("conjur URL cannot be empty"),
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

func makeAzureSecretStore(svcURL, account, serviceID, hostID string, withServiceAccountRef bool) *esv1.SecretStore {
	azure := &esv1.ConjurAzure{
		Account:   account,
		ServiceID: serviceID,
		HostID:    hostID,
	}
	if withServiceAccountRef {
		azure.ServiceAccountRef = &esmeta.ServiceAccountSelector{
			Name:      "my-service-account",
			Audiences: []string{"conjur"},
		}
	}
	return &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Conjur: &esv1.ConjurProvider{
					URL: svcURL,
					Auth: esv1.ConjurAuth{
						Azure: azure,
					},
				},
			},
		},
	}
}

func makeIAMSecretStore(svcURL, account, serviceID, hostID string, withSecretRef bool) *esv1.SecretStore {
	iam := &esv1.ConjurIAM{
		Account:   account,
		ServiceID: serviceID,
		HostID:    hostID,
	}
	if withSecretRef {
		iam.SecretRef = &esv1.ConjurIAMSecretRef{
			AccessKeyIDSecretRef: esmeta.SecretKeySelector{
				Name: "aws-creds",
				Key:  "access-key-id",
			},
			SecretAccessKeySecretRef: esmeta.SecretKeySelector{
				Name: "aws-creds",
				Key:  "secret-access-key",
			},
		}
	}
	return &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Conjur: &esv1.ConjurProvider{
					URL: svcURL,
					Auth: esv1.ConjurAuth{
						Iam: iam,
					},
				},
			},
		},
	}
}
