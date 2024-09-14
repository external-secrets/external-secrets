// Copyright External Secrets Inc. All Rights Reserved
package aws

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// This case creates secret with specific tags which are checked by the assumed IAM policy
func SimpleSyncWithNamespaceTags(prov *Provider) func(f *framework.Framework) (string, func(*framework.TestCase)) {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[common] should sync tagged simple secrets from .Data[]", func(tc *framework.TestCase) {
			secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
			secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "other")
			remoteRefKey1 := f.MakeRemoteRefKey(secretKey1)
			remoteRefKey2 := f.MakeRemoteRefKey(secretKey2)
			secretValue := "bar"
			tc.Secrets = map[string]framework.SecretEntry{
				// add specific tags to the secret resource. The assumed role only allows access to those
				remoteRefKey1: {Value: secretValue, Tags: map[string]string{"namespace": "e2e-test"}},
				remoteRefKey2: {Value: secretValue, Tags: map[string]string{"namespace": "e2e-test"}},
			}
			tc.ExpectedSecret = &v1.Secret{
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					secretKey1: []byte(secretValue),
					secretKey2: []byte(secretValue),
				},
			}
			tc.ExternalSecret.Spec.Data = []esapi.ExternalSecretData{
				{
					SecretKey: secretKey1,
					RemoteRef: esapi.ExternalSecretDataRemoteRef{
						Key: remoteRefKey1,
					},
				},
				{
					SecretKey: secretKey2,
					RemoteRef: esapi.ExternalSecretDataRemoteRef{
						Key: remoteRefKey2,
					},
				},
			}
		}
	}
}
