package conjur

import (
	"context"
	"fmt"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	fakeconjur "github.com/external-secrets/external-secrets/pkg/provider/conjur/fake"
)

var (
	svcUrl     string = "https://example.com"
	svcUser    string = "user"
	svcApikey  string = "apikey"
	svcAccount string = "account1"
)

type secretManagerTestCase struct {
	err    error
	refKey string
}

func TestConjurGetSecret(t *testing.T) {
	p := Provider{}
	p.ConjurClient = &fakeconjur.ConjurMockClient{}

	testCases := []*secretManagerTestCase{
		{
			err:    nil,
			refKey: "secret",
		},
		{
			err:    fmt.Errorf("error"),
			refKey: "error",
		},
	}

	for _, tc := range testCases {
		ref := makeValidRef(tc.refKey)
		_, err := p.GetSecret(context.Background(), *ref)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}

}

func makeValidRef(k string) *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     k,
		Version: "default",
	}
}

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

func TestValidateStore(t *testing.T) {
	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore(svcUrl, svcUser, svcApikey, svcAccount),
			err:   nil,
		},
		{
			store: makeSecretStore("", svcUser, svcApikey, svcAccount),
			err:   fmt.Errorf("ServiceURL cannot be empty"),
		},
		{
			store: makeSecretStore(svcUrl, "", svcApikey, svcAccount),
			err:   fmt.Errorf("ServiceUser cannot be empty"),
		},
		{
			store: makeSecretStore(svcUrl, svcUser, "", svcAccount),
			err:   fmt.Errorf("ServiceApiKey cannot be empty"),
		},
		{
			store: makeSecretStore(svcUrl, svcUser, svcApikey, ""),
			err:   fmt.Errorf("SeviceAccount cannot be empty"),
		},
	}
	p := Provider{}
	for _, tc := range testCases {
		err := p.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}
}

func makeSecretStore(svcUrl string, svcUser string, svcApikey string, svcAccount string) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Conjur: &esv1beta1.ConjurProvider{
					ServiceURL:     &svcUrl,
					ServiceUser:    &svcUser,
					ServiceApiKey:  &svcApikey,
					ServiceAccount: &svcAccount,
				},
			},
		},
	}
	return store
}
