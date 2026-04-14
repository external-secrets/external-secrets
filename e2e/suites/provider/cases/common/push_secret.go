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

package common

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

type ClusterProviderPushHarness struct {
	Prepare func(tc *framework.TestCase, cfg ClusterProviderConfig) *ClusterProviderPushRuntime
}

type ClusterProviderPushRuntime struct {
	ClusterProviderName       string
	DefaultRemoteNamespace    string
	BreakAuth                 func()
	RepairAuth                func()
	WaitForRemoteSecretValue  func(namespace, name, key, expectedValue string)
	ExpectNoRemoteSecret      func(namespace, name string)
	CreateWritableRemoteScope func(prefix string) string
}

func (r *ClusterProviderPushRuntime) SupportsAuthLifecycle() bool {
	return r != nil && r.BreakAuth != nil && r.RepairAuth != nil
}

func (r *ClusterProviderPushRuntime) SupportsRemoteAbsenceAssertions() bool {
	return r != nil && r.ExpectNoRemoteSecret != nil
}

func (r *ClusterProviderPushRuntime) SupportsRemoteNamespaceOverrides() bool {
	return r != nil && r.CreateWritableRemoteScope != nil
}

func PushSecretPreservesSourceMetadata(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should preserve source secret type, labels, and annotations when pushing to the namespaced Provider", func(tc *framework.TestCase) {
		tc.PushSecretSource = &corev1.Secret{
			Type: corev1.SecretTypeDockerConfigJson,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-secret-metadata",
				Namespace: f.Namespace.Name,
				Labels: map[string]string{
					"team": "platform",
				},
				Annotations: map[string]string{
					"owner": "eso",
				},
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"registry.example.com":{"auth":"ZXNvOnNlY3JldA=="}}}`),
			},
		}
		tc.PushSecret.ObjectMeta.Name = "test-pushsecret-metadata"
		tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
			Secret: &esv1alpha1.PushSecretSecret{
				Name: tc.PushSecretSource.Name,
			},
		}
		tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{{
			Match: esv1alpha1.PushSecretMatch{
				SecretKey: corev1.DockerConfigJsonKey,
				RemoteRef: esv1alpha1.PushSecretRemoteRef{
					RemoteKey: "pushed-docker-secret",
					Property:  corev1.DockerConfigJsonKey,
				},
			},
		}}
		tc.VerifyPushSecretOutcome = func(_ *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
			waitForPushSecretStatus(tc.Framework, tc.PushSecret.Namespace, tc.PushSecret.Name, corev1.ConditionTrue)

			var pushedSecret corev1.Secret
			Eventually(func(g Gomega) {
				g.Expect(tc.Framework.CRClient.Get(context.Background(), types.NamespacedName{
					Name:      "pushed-docker-secret",
					Namespace: f.Namespace.Name,
				}, &pushedSecret)).To(Succeed())
			}, time.Minute, 5*time.Second).Should(Succeed())

			Expect(pushedSecret.Type).To(Equal(tc.PushSecretSource.Type))
			Expect(pushedSecret.Labels).To(Equal(tc.PushSecretSource.Labels))
			Expect(pushedSecret.Annotations).To(Equal(tc.PushSecretSource.Annotations))
			Expect(pushedSecret.Data).To(Equal(tc.PushSecretSource.Data))
		}
	}
}

func PushSecretImplicitProviderKind(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should support namespaced Provider refs when kind is omitted", func(tc *framework.TestCase) {
		tc.PushSecretSource = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-secret-implicit-kind",
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"key1": []byte("value1"),
			},
		}
		tc.PushSecret.ObjectMeta.Name = "test-pushsecret-implicit-kind"
		tc.PushSecret.Spec.DeletionPolicy = esv1alpha1.PushSecretDeletionPolicyDelete
		tc.PushSecret.Spec.SecretStoreRefs[0].Kind = ""
		tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
			Secret: &esv1alpha1.PushSecretSecret{
				Name: tc.PushSecretSource.Name,
			},
		}
		tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{{
			Match: esv1alpha1.PushSecretMatch{
				SecretKey: "key1",
				RemoteRef: esv1alpha1.PushSecretRemoteRef{
					RemoteKey: "pushed-secret-implicit-kind",
					Property:  "key1",
				},
			},
		}}
		tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
			waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
			Eventually(func(g Gomega) {
				var pushedSecret corev1.Secret
				g.Expect(tc.Framework.CRClient.Get(context.Background(), types.NamespacedName{
					Name:      "pushed-secret-implicit-kind",
					Namespace: f.Namespace.Name,
				}, &pushedSecret)).To(Succeed())
				g.Expect(string(pushedSecret.Data["key1"])).To(Equal("value1"))
			}, time.Minute, 5*time.Second).Should(Succeed())

			Expect(tc.Framework.CRClient.Delete(context.Background(), ps)).To(Succeed())
			Eventually(func() bool {
				var pushedSecret corev1.Secret
				err := tc.Framework.CRClient.Get(context.Background(), types.NamespacedName{
					Name:      "pushed-secret-implicit-kind",
					Namespace: f.Namespace.Name,
				}, &pushedSecret)
				return apierrors.IsNotFound(err)
			}, time.Minute, 5*time.Second).Should(BeTrue())
		}
	}
}

func PushSecretRejectsNamespacedRemoteNamespaceOverride(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should reject remote namespace overrides when pushing through a namespaced Provider", func(tc *framework.TestCase) {
		overrideNamespace := createE2ENamespace(tc.Framework, "push-provider-override")
		tc.PushSecretSource = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-secret-provider-override",
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"value": []byte("should-not-push"),
			},
		}
		tc.PushSecret.ObjectMeta.Name = "test-pushsecret-provider-override"
		tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
			Secret: &esv1alpha1.PushSecretSecret{
				Name: tc.PushSecretSource.Name,
			},
		}
		tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{{
			Match: esv1alpha1.PushSecretMatch{
				SecretKey: "value",
				RemoteRef: esv1alpha1.PushSecretRemoteRef{
					RemoteKey: "pushed-secret-provider-override",
					Property:  "value",
				},
			},
			Metadata: pushSecretMetadataWithRemoteNamespace(overrideNamespace),
		}}
		tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
			waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionFalse)
			expectNoSecretInNamespace(tc.Framework, f.Namespace.Name, "pushed-secret-provider-override")
			expectNoSecretInNamespace(tc.Framework, overrideNamespace, "pushed-secret-provider-override")
			expectEventMessage(tc.Framework, ps.Namespace, ps.Name, "PushSecret", "remoteNamespace override is only supported with ClusterSecretStore")
		}
	}
}

func ClusterProviderPushManifestNamespace(f *framework.Framework, harness ClusterProviderPushHarness) (string, func(*framework.TestCase)) {
	return clusterProviderPushSyncCase(f, harness, "push-manifest", "manifest-push-value", esv1.AuthenticationScopeManifestNamespace)
}

func ClusterProviderPushProviderNamespace(f *framework.Framework, harness ClusterProviderPushHarness) (string, func(*framework.TestCase)) {
	return clusterProviderPushSyncCase(f, harness, "push-provider", "provider-push-value", esv1.AuthenticationScopeProviderNamespace)
}

func ClusterProviderPushManifestNamespaceRecovery(f *framework.Framework, harness ClusterProviderPushHarness) (string, func(*framework.TestCase)) {
	return clusterProviderPushRecoveryCase(f, harness, "push-manifest-recovery", "manifest-push-recovered", esv1.AuthenticationScopeManifestNamespace)
}

func ClusterProviderPushProviderNamespaceRecovery(f *framework.Framework, harness ClusterProviderPushHarness) (string, func(*framework.TestCase)) {
	return clusterProviderPushRecoveryCase(f, harness, "push-provider-recovery", "provider-push-recovered", esv1.AuthenticationScopeProviderNamespace)
}

func ClusterProviderPushAllowsRemoteNamespaceOverride(f *framework.Framework, harness ClusterProviderPushHarness) (string, func(*framework.TestCase)) {
	return "[common] should allow ClusterProvider pushes to override the target remote namespace via metadata", func(tc *framework.TestCase) {
		tc.PushSecretSource = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "push-remote-override-source",
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"value": []byte("override-push-value"),
			},
		}

		var runtime *ClusterProviderPushRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			remoteSecretName := f.MakeRemoteRefKey("push-remote-override-remote")
			runtime = harness.Prepare(tc, ClusterProviderConfig{
				Name:      "push-remote-override",
				AuthScope: esv1.AuthenticationScopeManifestNamespace,
			})
			applyClusterProviderPushSecret(tc, runtime, remoteSecretName)
			if !runtime.SupportsRemoteNamespaceOverrides() {
				Skip(fmt.Sprintf("provider %q does not support remote namespace override hooks", runtime.ClusterProviderName))
			}
			overrideNamespace := runtime.CreateWritableRemoteScope("push-remote-override-target")
			tc.PushSecret.Spec.Data[0].Metadata = pushSecretMetadataWithRemoteNamespace(overrideNamespace)
			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
				runtime.WaitForRemoteSecretValue(overrideNamespace, remoteSecretName, "value", "override-push-value")
				if runtime.SupportsRemoteAbsenceAssertions() {
					runtime.ExpectNoRemoteSecret(runtime.DefaultRemoteNamespace, remoteSecretName)
				}
			}
		}
	}
}

func ClusterProviderPushDeniedByConditions(f *framework.Framework, harness ClusterProviderPushHarness) (string, func(*framework.TestCase)) {
	return "[common] should deny PushSecrets from namespaces that do not match ClusterProvider conditions", func(tc *framework.TestCase) {
		tc.PushSecretSource = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "push-deny-source",
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"value": []byte("should-not-push"),
			},
		}

		var runtime *ClusterProviderPushRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			remoteSecretName := f.MakeRemoteRefKey("push-deny-remote")
			runtime = harness.Prepare(tc, ClusterProviderConfig{
				Name:      "push-deny",
				AuthScope: esv1.AuthenticationScopeManifestNamespace,
				Conditions: []esv1.ClusterSecretStoreCondition{{
					Namespaces: []string{"not-" + f.Namespace.Name},
				}},
			})
			applyClusterProviderPushSecret(tc, runtime, remoteSecretName)
			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionFalse)
				if runtime.SupportsRemoteAbsenceAssertions() {
					runtime.ExpectNoRemoteSecret(runtime.DefaultRemoteNamespace, remoteSecretName)
				}
				expectEventMessage(tc.Framework, ps.Namespace, ps.Name, "PushSecret", fmt.Sprintf("using ClusterProvider %q is not allowed from namespace %q: denied by spec.conditions", runtime.ClusterProviderName, f.Namespace.Name))
			}
		}
	}
}

func clusterProviderPushSyncCase(f *framework.Framework, harness ClusterProviderPushHarness, name, expectedValue string, authScope esv1.AuthenticationScope) (string, func(*framework.TestCase)) {
	return fmt.Sprintf("[common] should push through a ClusterProvider with %s auth", authScope), func(tc *framework.TestCase) {
		tc.PushSecretSource = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-source", name),
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"value": []byte(expectedValue),
			},
		}

		var runtime *ClusterProviderPushRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			remoteSecretName := f.MakeRemoteRefKey(fmt.Sprintf("%s-remote", name))
			runtime = harness.Prepare(tc, ClusterProviderConfig{
				Name:      name,
				AuthScope: authScope,
			})
			applyClusterProviderPushSecret(tc, runtime, remoteSecretName)
			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
				runtime.WaitForRemoteSecretValue(runtime.DefaultRemoteNamespace, remoteSecretName, "value", expectedValue)
			}
		}
	}
}

func clusterProviderPushRecoveryCase(f *framework.Framework, harness ClusterProviderPushHarness, name, expectedValue string, authScope esv1.AuthenticationScope) (string, func(*framework.TestCase)) {
	return fmt.Sprintf("[common] should recover after repairing ClusterProvider push auth with %s scope", authScope), func(tc *framework.TestCase) {
		tc.PushSecretSource = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-source", name),
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"value": []byte(expectedValue),
			},
		}

		var runtime *ClusterProviderPushRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			remoteSecretName := f.MakeRemoteRefKey(fmt.Sprintf("%s-remote", name))
			runtime = harness.Prepare(tc, ClusterProviderConfig{
				Name:      name,
				AuthScope: authScope,
			})
			applyClusterProviderPushSecret(tc, runtime, remoteSecretName)
			if !runtime.SupportsAuthLifecycle() {
				Skip(fmt.Sprintf("provider %q does not support auth lifecycle recovery hooks", runtime.ClusterProviderName))
			}
			tc.PushSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Hour}
			runtime.BreakAuth()
			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionFalse)
				if runtime.SupportsRemoteAbsenceAssertions() {
					runtime.ExpectNoRemoteSecret(runtime.DefaultRemoteNamespace, remoteSecretName)
				}
				runtime.RepairAuth()
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
				runtime.WaitForRemoteSecretValue(runtime.DefaultRemoteNamespace, remoteSecretName, "value", expectedValue)
			}
		}
	}
}

func applyClusterProviderPushSecret(tc *framework.TestCase, runtime *ClusterProviderPushRuntime, remoteSecretName string) {
	if runtime == nil {
		panic("cluster provider push harness returned nil runtime")
	}

	tc.PushSecret.ObjectMeta.Name = fmt.Sprintf("%s-push-secret", tc.PushSecretSource.Name)
	tc.PushSecret.Spec.SecretStoreRefs = []esv1alpha1.PushSecretStoreRef{{
		Name:       runtime.ClusterProviderName,
		Kind:       esv1.ClusterProviderKindStr,
		APIVersion: esv1.SchemeGroupVersion.String(),
	}}
	tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
		Secret: &esv1alpha1.PushSecretSecret{
			Name: tc.PushSecretSource.Name,
		},
	}
	tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{{
		Match: esv1alpha1.PushSecretMatch{
			SecretKey: "value",
			RemoteRef: esv1alpha1.PushSecretRemoteRef{
				RemoteKey: remoteSecretName,
				Property:  "value",
			},
		},
	}}
}

func waitForPushSecretStatus(f *framework.Framework, namespace, name string, status corev1.ConditionStatus) {
	Eventually(func(g Gomega) {
		var ps esv1alpha1.PushSecret
		g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, &ps)).To(Succeed())
		g.Expect(ps.Status.Conditions).NotTo(BeEmpty())
		ready := false
		for _, condition := range ps.Status.Conditions {
			if condition.Type == esv1alpha1.PushSecretReady && condition.Status == status {
				ready = true
			}
		}
		g.Expect(ready).To(BeTrue())
	}, time.Minute, 5*time.Second).Should(Succeed())
}

func pushSecretMetadataWithRemoteNamespace(namespace string) *apiextensionsv1.JSON {
	return &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"remoteNamespace":"%s"}}`, namespace))}
}

func createE2ENamespace(f *framework.Framework, prefix string) string {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("e2e-tests-%s-", prefix),
		},
	}
	Expect(f.CRClient.Create(context.Background(), namespace)).To(Succeed())

	DeferCleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		err := f.CRClient.Delete(ctx, namespace)
		if err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
			_, err := f.KubeClientSet.CoreV1().Namespaces().Get(ctx, namespace.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			if err != nil {
				return false, err
			}
			return false, nil
		})
		Expect(err).To(Succeed())
	})

	return namespace.Name
}
