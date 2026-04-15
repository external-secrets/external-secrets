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

package aws

import (
	"testing"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestConfigureV2ReferencedIRSAStoreRefUsesClusterProvider(t *testing.T) {
	t.Parallel()

	tc := &framework.TestCase{
		ExternalSecret: &esv1.ExternalSecret{
			Spec: esv1.ExternalSecretSpec{
				SecretStoreRef: esv1.SecretStoreRef{
					Name: "placeholder",
					Kind: esv1.ProviderKindStr,
				},
			},
		},
	}

	configureV2ReferencedIRSAStoreRef(tc, "aws-irsa-cluster-provider")

	if got := tc.ExternalSecret.Spec.SecretStoreRef.Kind; got != esv1.ClusterProviderKindStr {
		t.Fatalf("expected cluster provider kind %q, got %q", esv1.ClusterProviderKindStr, got)
	}
	if got := tc.ExternalSecret.Spec.SecretStoreRef.Name; got != "aws-irsa-cluster-provider" {
		t.Fatalf("expected cluster provider ref %q, got %q", "aws-irsa-cluster-provider", got)
	}
}
