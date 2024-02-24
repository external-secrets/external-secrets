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

package kubernetes

import (
	"fmt"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const referentAuth = "with referent auth"

var _ = Describe("[kubernetes] ", Label("kubernetes"), func() {
	f := framework.New("eso-kubernetes")
	prov := NewProvider(f)

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f,
			prov),
		Entry(common.JSONDataWithProperty(f)),
		Entry(common.JSONDataWithoutTargetName(f)),
		Entry(common.JSONDataWithTemplate(f)),
		Entry(common.DataPropertyDockerconfigJSON(f)),
		Entry(common.SSHKeySyncDataProperty(f)),
		Entry(common.JSONDataFromSync(f)),
		Entry(common.JSONDataFromRewrite(f)),
		Entry(FindByTag(f)),
		Entry(FindByName(f)),

		framework.Compose(referentAuth, f, common.JSONDataWithProperty, withReferentStore),
		framework.Compose(referentAuth, f, common.JSONDataWithoutTargetName, withReferentStore),
	)
})

func withReferentStore(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = referentStoreName(tc.Framework)
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esapi.ClusterSecretStoreKind
}

const (
	secretValue1 = "{\"foo1\":\"foo1-val\"}"
)

// This case creates multiple secrets with simple key/value pairs and syncs them using multiple .Spec.Data blocks.
func FindByName(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should find secrets by name using .DataFrom[]", func(tc *framework.TestCase) {
		const namePrefix = "e2e-find-name-%s-%s"
		secretKeyOne := fmt.Sprintf(namePrefix, f.Namespace.Name, "one")
		secretKeyTwo := fmt.Sprintf(namePrefix, f.Namespace.Name, "two")
		secretKeyThree := fmt.Sprintf(namePrefix, f.Namespace.Name, "three")
		secretValue := "{\"foo1\":\"foo1-val\"}"
		tc.Secrets = map[string]framework.SecretEntry{
			secretKeyOne:   {Value: secretValue},
			secretKeyTwo:   {Value: secretValue},
			secretKeyThree: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKeyOne:   []byte(secretValue),
				secretKeyTwo:   []byte(secretValue),
				secretKeyThree: []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Find: &esapi.ExternalSecretFind{
					Name: &esapi.FindName{
						RegExp: fmt.Sprintf("e2e-find-name-%s.+", f.Namespace.Name),
					},
				},
			},
		}
	}
}

// This case creates multiple secrets with simple key/value pairs and syncs them using multiple .Spec.Data blocks.
func FindByTag(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should find secrets by tags using .DataFrom[]", func(tc *framework.TestCase) {
		const namePrefix = "e2e-find-name-%s-%s"
		secretKeyOne := fmt.Sprintf(namePrefix, f.Namespace.Name, "one")
		secretKeyTwo := fmt.Sprintf(namePrefix, f.Namespace.Name, "two")
		secretKeyThree := fmt.Sprintf(namePrefix, f.Namespace.Name, "three")
		tc.Secrets = map[string]framework.SecretEntry{
			secretKeyOne: {
				Value: secretValue1,
				Tags: map[string]string{
					"test": f.Namespace.Name,
				}},
			secretKeyTwo: {
				Value: secretValue1,
				Tags: map[string]string{
					"test": f.Namespace.Name,
				}},
			secretKeyThree: {
				Value: secretValue1,
				Tags: map[string]string{
					"noop": f.Namespace.Name,
				}},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				fmt.Sprintf("e2e-find-name-%s-one", f.Namespace.Name): []byte(secretValue1),
				fmt.Sprintf("e2e-find-name-%s-two", f.Namespace.Name): []byte(secretValue1),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Find: &esapi.ExternalSecretFind{
					Tags: map[string]string{
						"test": f.Namespace.Name,
					},
				},
			},
		}
	}
}
