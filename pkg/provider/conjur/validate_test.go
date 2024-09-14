//Copyright External Secrets Inc. All Rights Reserved

package conjur

import (
	"errors"
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
			store: makeNoAuthSecretStore(svcURL),
			err:   errors.New("missing Auth.* configuration"),
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
