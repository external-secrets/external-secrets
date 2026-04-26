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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[aws] v2 push secret", Label("aws", "secretsmanager", "v2", "push-secret"), func() {
	f := framework.New("eso-aws-sm-v2-push")
	prov := NewProviderV2(f)
	harness := newAWSClusterProviderPushHarness(f, prov)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("push secret",
		framework.TableFuncWithPushSecret(f, prov, nil),
		Entry(awsPushSecretImplicitProviderKind(f, prov)),
		Entry(awsPushSecretRejectsNamespacedRemoteNamespaceOverride(f, prov)),
		Entry(common.ClusterProviderPushManifestNamespace(f, harness)),
		Entry(common.ClusterProviderPushProviderNamespace(f, harness)),
		Entry(common.ClusterProviderPushDeniedByConditions(f, harness)),
	)
})

func newAWSClusterProviderPushHarness(f *framework.Framework, prov *ProviderV2) common.ClusterProviderPushHarness {
	return common.ClusterProviderPushHarness{
		Prepare: func(_ *framework.TestCase, cfg common.ClusterProviderConfig) *common.ClusterProviderPushRuntime {
			s := newAWSClusterProviderScenario(f, cfg.Name, cfg.AuthScope, prov.access, prov.backend)
			clusterProviderName := s.createClusterProvider(cfg.Conditions)
			frameworkv2.WaitForClusterSecretStoreReady(f, clusterProviderName, defaultV2WaitTimeout)

			return &common.ClusterProviderPushRuntime{
				ClusterProviderName: clusterProviderName,
				StoreRef: esv1.SecretStoreRef{
					Name: clusterProviderName,
					Kind: esv1.ClusterSecretStoreKind,
				},
				DefaultRemoteNamespace: "",
				WaitForRemoteSecretValue: func(_, name, key, expectedValue string) {
					s.waitForRemoteSecretValue(name, key, expectedValue)
				},
				ExpectNoRemoteSecret: func(_, name string) {
					s.backend.ExpectSecretAbsent(name)
				},
			}
		},
	}
}

func (s *awsClusterProviderScenario) waitForRemoteSecretValue(name, key, expectedValue string) {
	if key == "" {
		s.backend.WaitForSecretValue(name, expectedValue)
		return
	}
	s.backend.WaitForSecretValue(name, fmt.Sprintf(`{"%s":"%s"}`, key, expectedValue))
}

func awsPushSecretImplicitProviderKind(f *framework.Framework, prov *ProviderV2) (string, func(*framework.TestCase)) {
	return "[aws] should support namespaced Provider refs when push kind is omitted", func(tc *framework.TestCase) {
		remoteKey := f.MakeRemoteRefKey("aws-v2-push-implicit")
		tc.Prepare = prov.prepareNamespacedProvider(awsAuthProfileStatic)
		tc.PushSecretSource = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aws-v2-push-implicit-source",
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"value": []byte("value1"),
			},
		}
		tc.PushSecret.ObjectMeta.Name = "aws-v2-push-implicit"
		tc.PushSecret.Spec.DeletionPolicy = esv1alpha1.PushSecretDeletionPolicyDelete
		tc.PushSecret.Spec.SecretStoreRefs[0].Kind = ""
		tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
			Secret: &esv1alpha1.PushSecretSecret{
				Name: tc.PushSecretSource.Name,
			},
		}
		tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{{
			Match: esv1alpha1.PushSecretMatch{
				SecretKey: "value",
				RemoteRef: esv1alpha1.PushSecretRemoteRef{
					RemoteKey: remoteKey,
					Property:  "value",
				},
			},
		}}
		tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
			awscommon.WaitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
			prov.backend.WaitForSecretValue(remoteKey, `{"value":"value1"}`)

			Expect(tc.Framework.CRClient.Delete(context.Background(), ps)).To(Succeed())
			prov.backend.ExpectSecretAbsent(remoteKey)
		}
	}
}

func awsPushSecretRejectsNamespacedRemoteNamespaceOverride(f *framework.Framework, prov *ProviderV2) (string, func(*framework.TestCase)) {
	return "[aws] should reject remote namespace overrides when pushing through a namespaced Provider", func(tc *framework.TestCase) {
		remoteKey := f.MakeRemoteRefKey("aws-v2-push-override")
		tc.Prepare = prov.prepareNamespacedProvider(awsAuthProfileStatic)
		tc.PushSecretSource = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aws-v2-push-override-source",
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"value": []byte("should-not-push"),
			},
		}
		tc.PushSecret.ObjectMeta.Name = "aws-v2-push-override"
		tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
			Secret: &esv1alpha1.PushSecretSecret{
				Name: tc.PushSecretSource.Name,
			},
		}
		tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{{
			Match: esv1alpha1.PushSecretMatch{
				SecretKey: "value",
				RemoteRef: esv1alpha1.PushSecretRemoteRef{
					RemoteKey: remoteKey,
					Property:  "value",
				},
			},
			Metadata: awscommon.PushSecretMetadataWithRemoteNamespace("ignored-aws-namespace"),
		}}
		tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
			awscommon.WaitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionFalse)
			prov.backend.ExpectSecretAbsent(remoteKey)
			awscommon.ExpectPushSecretEventMessage(tc.Framework, ps.Namespace, ps.Name, `unknown field "remoteNamespace"`)
		}
	}
}
