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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

const (
	operationalPollInterval = 5 * time.Second
	operationalTimeout      = 3 * time.Minute
)

type OperationalRuntime struct {
	Provider               framework.SecretStoreProvider
	ProviderRef            esv1.SecretStoreRef
	DefaultRemoteNamespace string
	WaitForRemoteSecret    func(namespace, name, key, expectedValue string)
	ExpectNoRemoteSecret   func(namespace, name string)
	MakeUnavailable        func()
	RestoreAvailability    func()
	RestartBackend         func()
}

func (r *OperationalRuntime) SupportsDisruptionLifecycle() bool {
	return r != nil && r.MakeUnavailable != nil && r.RestoreAvailability != nil
}

func (r *OperationalRuntime) SupportsRestart() bool {
	return r != nil && r.RestartBackend != nil
}

type OperationalExternalSecretHarness struct {
	PrepareNamespaced func(tc *framework.TestCase) *OperationalRuntime
	PrepareCluster    func(tc *framework.TestCase, cfg ClusterProviderConfig) *OperationalRuntime
}

type OperationalPushSecretHarness struct {
	PrepareNamespaced func(tc *framework.TestCase) *OperationalRuntime
	PrepareCluster    func(tc *framework.TestCase, cfg ClusterProviderConfig) *OperationalRuntime
}

func NamespacedProviderUnavailable(f *framework.Framework, harness OperationalExternalSecretHarness, remoteKey, expectedValue string) (string, func(*framework.TestCase)) {
	return "[common] should surface namespaced Provider unavailability and recover after backend restoration", func(tc *framework.TestCase) {
		tc.ExternalSecret.ObjectMeta.Name = "operational-unavailable-es"
		tc.ExternalSecret.Spec.Target.Name = "operational-unavailable-target"
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      remoteKey,
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			remoteKey: {Value: jsonSecretValue(expectedValue)},
		}
		tc.ExpectedSecret = opaqueValueSecret(expectedValue)

		var runtime *OperationalRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime = harness.PrepareNamespaced(tc)
			applyOperationalExternalSecret(tc, runtime)
		}
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
			skipIfOperationalRuntimeMissingDisruptionLifecycle(runtime)
			DeferCleanup(func() {
				runtime.RestoreAvailability()
				waitForProviderRefCondition(tc.Framework, tc.ExternalSecret.Namespace, runtime.ProviderRef, metav1.ConditionTrue)
			})
			runtime.MakeUnavailable()
			waitForExternalSecretStatus(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, corev1.ConditionFalse)

			runtime.RestoreAvailability()
			waitForProviderRefCondition(tc.Framework, tc.ExternalSecret.Namespace, runtime.ProviderRef, metav1.ConditionTrue)
			waitForExternalSecretStatus(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, corev1.ConditionTrue)
			waitForSecretData(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Spec.Target.Name, map[string][]byte{
				"value": []byte(expectedValue),
			}, operationalTimeout)
		}
	}
}

func NamespacedProviderRestart(f *framework.Framework, harness OperationalExternalSecretHarness, remoteKey, expectedValue string) (string, func(*framework.TestCase)) {
	return "[common] should recover namespaced Provider reads after backend restart", func(tc *framework.TestCase) {
		tc.ExternalSecret.ObjectMeta.Name = "operational-restart-es"
		tc.ExternalSecret.Spec.Target.Name = "operational-restart-target"
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      remoteKey,
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			remoteKey: {Value: jsonSecretValue("before-restart")},
		}
		tc.ExpectedSecret = opaqueValueSecret("before-restart")

		var runtime *OperationalRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime = harness.PrepareNamespaced(tc)
			applyOperationalExternalSecret(tc, runtime)
		}
		tc.AfterSync = func(prov framework.SecretStoreProvider, _ *corev1.Secret) {
			skipIfOperationalRuntimeMissingRestart(runtime)
			runtime.RestartBackend()
			waitForProviderRefCondition(tc.Framework, tc.ExternalSecret.Namespace, runtime.ProviderRef, metav1.ConditionTrue)

			prov.DeleteSecret(remoteKey)
			prov.CreateSecret(remoteKey, framework.SecretEntry{Value: jsonSecretValue(expectedValue)})

			waitForExternalSecretStatus(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, corev1.ConditionTrue)
			waitForSecretData(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Spec.Target.Name, map[string][]byte{
				"value": []byte(expectedValue),
			}, operationalTimeout)
		}
	}
}

func ClusterProviderUnavailable(f *framework.Framework, harness OperationalExternalSecretHarness, remoteKey, expectedValue string, authScope esv1.AuthenticationScope) (string, func(*framework.TestCase)) {
	return fmt.Sprintf("[common] should surface ClusterProvider unavailability and recover with %s auth", authScope), func(tc *framework.TestCase) {
		scopeSuffix := operationalScopeSuffix(authScope)
		tc.ExternalSecret.ObjectMeta.Name = fmt.Sprintf("operational-cluster-unavailable-%s", scopeSuffix)
		tc.ExternalSecret.Spec.Target.Name = fmt.Sprintf("operational-cluster-unavailable-%s-target", scopeSuffix)
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      remoteKey,
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			remoteKey: {Value: jsonSecretValue(expectedValue)},
		}
		tc.ExpectedSecret = opaqueValueSecret(expectedValue)

		var runtime *OperationalRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime = harness.PrepareCluster(tc, ClusterProviderConfig{
				Name:      "operational-unavailable",
				AuthScope: authScope,
			})
			applyOperationalExternalSecret(tc, runtime)
		}
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
			skipIfOperationalRuntimeMissingDisruptionLifecycle(runtime)
			DeferCleanup(func() {
				runtime.RestoreAvailability()
				waitForProviderRefCondition(tc.Framework, tc.ExternalSecret.Namespace, runtime.ProviderRef, metav1.ConditionTrue)
			})
			runtime.MakeUnavailable()
			waitForExternalSecretStatus(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, corev1.ConditionFalse)

			runtime.RestoreAvailability()
			waitForProviderRefCondition(tc.Framework, tc.ExternalSecret.Namespace, runtime.ProviderRef, metav1.ConditionTrue)
			waitForExternalSecretStatus(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, corev1.ConditionTrue)
			waitForSecretData(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Spec.Target.Name, map[string][]byte{
				"value": []byte(expectedValue),
			}, operationalTimeout)
		}
	}
}

func ClusterProviderRestart(f *framework.Framework, harness OperationalExternalSecretHarness, remoteKey, expectedValue string, authScope esv1.AuthenticationScope) (string, func(*framework.TestCase)) {
	return fmt.Sprintf("[common] should recover ClusterProvider reads after backend restart with %s auth", authScope), func(tc *framework.TestCase) {
		scopeSuffix := operationalScopeSuffix(authScope)
		tc.ExternalSecret.ObjectMeta.Name = fmt.Sprintf("operational-cluster-restart-%s", scopeSuffix)
		tc.ExternalSecret.Spec.Target.Name = fmt.Sprintf("operational-cluster-restart-%s-target", scopeSuffix)
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      remoteKey,
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			remoteKey: {Value: jsonSecretValue("before-restart")},
		}
		tc.ExpectedSecret = opaqueValueSecret("before-restart")

		var runtime *OperationalRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime = harness.PrepareCluster(tc, ClusterProviderConfig{
				Name:      "operational-restart",
				AuthScope: authScope,
			})
			applyOperationalExternalSecret(tc, runtime)
		}
		tc.AfterSync = func(prov framework.SecretStoreProvider, _ *corev1.Secret) {
			skipIfOperationalRuntimeMissingRestart(runtime)
			runtime.RestartBackend()
			waitForProviderRefCondition(tc.Framework, tc.ExternalSecret.Namespace, runtime.ProviderRef, metav1.ConditionTrue)

			prov.DeleteSecret(remoteKey)
			prov.CreateSecret(remoteKey, framework.SecretEntry{Value: jsonSecretValue(expectedValue)})

			waitForExternalSecretStatus(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, corev1.ConditionTrue)
			waitForSecretData(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Spec.Target.Name, map[string][]byte{
				"value": []byte(expectedValue),
			}, operationalTimeout)
		}
	}
}

func NamespacedPushSecretUnavailable(f *framework.Framework, harness OperationalPushSecretHarness) (string, func(*framework.TestCase)) {
	return "[common] should surface namespaced Provider push unavailability and recover after backend restoration", func(tc *framework.TestCase) {
		tc.PushSecretSource = operationalPushSourceSecret(f, "operational-push-namespaced-source", "before-outage")

		var runtime *OperationalRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime = harness.PrepareNamespaced(tc)
			remoteSecretName := f.MakeRemoteRefKey("operational-push-namespaced-remote")
			applyOperationalPushSecret(tc, runtime, remoteSecretName)
			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
				runtime.WaitForRemoteSecret(runtime.DefaultRemoteNamespace, remoteSecretName, "value", "before-outage")

				skipIfOperationalRuntimeMissingDisruptionLifecycle(runtime)
				DeferCleanup(func() {
					runtime.RestoreAvailability()
					waitForProviderRefCondition(tc.Framework, ps.Namespace, runtime.ProviderRef, metav1.ConditionTrue)
				})
				runtime.MakeUnavailable()
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionFalse)

				runtime.RestoreAvailability()
				waitForProviderRefCondition(tc.Framework, ps.Namespace, runtime.ProviderRef, metav1.ConditionTrue)
				updatePushSecretSource(tc.Framework, tc.PushSecretSource.Namespace, tc.PushSecretSource.Name, "after-outage")
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
				runtime.WaitForRemoteSecret(runtime.DefaultRemoteNamespace, remoteSecretName, "value", "after-outage")
			}
		}
	}
}

func ClusterProviderPushUnavailable(f *framework.Framework, harness OperationalPushSecretHarness, authScope esv1.AuthenticationScope) (string, func(*framework.TestCase)) {
	return fmt.Sprintf("[common] should surface ClusterProvider push unavailability and recover with %s auth", authScope), func(tc *framework.TestCase) {
		scopeSuffix := operationalScopeSuffix(authScope)
		tc.PushSecretSource = operationalPushSourceSecret(f, fmt.Sprintf("operational-push-cluster-source-%s", scopeSuffix), "before-outage")

		var runtime *OperationalRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime = harness.PrepareCluster(tc, ClusterProviderConfig{
				Name:      "operational-push-unavailable",
				AuthScope: authScope,
			})
			remoteSecretName := f.MakeRemoteRefKey(fmt.Sprintf("operational-push-cluster-remote-%s", scopeSuffix))
			applyOperationalPushSecret(tc, runtime, remoteSecretName)
			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
				runtime.WaitForRemoteSecret(runtime.DefaultRemoteNamespace, remoteSecretName, "value", "before-outage")

				skipIfOperationalRuntimeMissingDisruptionLifecycle(runtime)
				DeferCleanup(func() {
					runtime.RestoreAvailability()
					waitForProviderRefCondition(tc.Framework, ps.Namespace, runtime.ProviderRef, metav1.ConditionTrue)
				})
				runtime.MakeUnavailable()
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionFalse)

				runtime.RestoreAvailability()
				waitForProviderRefCondition(tc.Framework, ps.Namespace, runtime.ProviderRef, metav1.ConditionTrue)
				updatePushSecretSource(tc.Framework, tc.PushSecretSource.Namespace, tc.PushSecretSource.Name, "after-outage")
				waitForPushSecretStatus(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
				runtime.WaitForRemoteSecret(runtime.DefaultRemoteNamespace, remoteSecretName, "value", "after-outage")
			}
		}
	}
}

func applyOperationalExternalSecret(tc *framework.TestCase, runtime *OperationalRuntime) {
	Expect(runtime).NotTo(BeNil(), "operational harness returned nil runtime")
	tc.ExternalSecret.Spec.SecretStoreRef = runtime.ProviderRef
	if runtime.Provider != nil {
		tc.ProviderOverride = runtime.Provider
	}
}

func applyOperationalPushSecret(tc *framework.TestCase, runtime *OperationalRuntime, remoteSecretName string) {
	Expect(runtime).NotTo(BeNil(), "operational harness returned nil runtime")
	Expect(runtime.ProviderRef.Name).NotTo(BeEmpty(), "operational runtime provider ref name must be set")
	Expect(runtime.WaitForRemoteSecret).NotTo(BeNil(), "operational runtime WaitForRemoteSecret hook must be set")

	tc.PushSecret.ObjectMeta.Name = fmt.Sprintf("%s-push-secret", tc.PushSecretSource.Name)
	tc.PushSecret.Spec.SecretStoreRefs = []esv1alpha1.PushSecretStoreRef{{
		Name:       runtime.ProviderRef.Name,
		Kind:       runtime.ProviderRef.Kind,
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

func waitForProviderRefCondition(f *framework.Framework, namespace string, ref esv1.SecretStoreRef, status metav1.ConditionStatus) {
	switch ref.Kind {
	case esv1.ClusterProviderKindStr:
		frameworkv2.WaitForClusterProviderCondition(f, ref.Name, status, operationalTimeout)
	default:
		frameworkv2.WaitForProviderConnectionCondition(f, namespace, ref.Name, status, operationalTimeout)
	}
}

func skipIfOperationalRuntimeMissingDisruptionLifecycle(runtime *OperationalRuntime) {
	Expect(runtime).NotTo(BeNil(), "operational harness returned nil runtime")
	if !runtime.SupportsDisruptionLifecycle() {
		Skip(fmt.Sprintf("provider ref %q does not support disruption lifecycle hooks", runtime.ProviderRef.Name))
	}
}

func skipIfOperationalRuntimeMissingRestart(runtime *OperationalRuntime) {
	Expect(runtime).NotTo(BeNil(), "operational harness returned nil runtime")
	if !runtime.SupportsRestart() {
		Skip(fmt.Sprintf("provider ref %q does not support restart hooks", runtime.ProviderRef.Name))
	}
}

func operationalPushSourceSecret(f *framework.Framework, name, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace.Name,
		},
		Data: map[string][]byte{
			"value": []byte(value),
		},
	}
}

func updatePushSecretSource(f *framework.Framework, namespace, name, value string) {
	Eventually(func(g Gomega) {
		var secret corev1.Secret
		g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, &secret)).To(Succeed())
		secret.Data["value"] = []byte(value)
		g.Expect(f.CRClient.Update(context.Background(), &secret)).To(Succeed())
	}, operationalTimeout, operationalPollInterval).Should(Succeed())
}

func opaqueValueSecret(value string) *corev1.Secret {
	return &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"value": []byte(value),
		},
	}
}

func operationalScopeSuffix(authScope esv1.AuthenticationScope) string {
	replacer := strings.NewReplacer(
		"ManifestNamespace", "manifest-namespace",
		"ProviderNamespace", "provider-namespace",
	)
	return replacer.Replace(string(authScope))
}
