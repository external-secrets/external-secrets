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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type ClusterProviderConfig struct {
	Name       string
	AuthScope  esv1.AuthenticationScope
	Conditions []esv1.ClusterSecretStoreCondition
}

type ClusterProviderExternalSecretHarness struct {
	Prepare func(tc *framework.TestCase, cfg ClusterProviderConfig) *ClusterProviderExternalSecretRuntime
}

type ClusterProviderExternalSecretRuntime struct {
	ClusterProviderName string
	Provider            framework.SecretStoreProvider
	BreakAuth           func()
	RepairAuth          func()
}

func (r *ClusterProviderExternalSecretRuntime) SupportsAuthLifecycle() bool {
	return r != nil && r.BreakAuth != nil && r.RepairAuth != nil
}

func ClusterProviderManifestNamespace(f *framework.Framework, harness ClusterProviderExternalSecretHarness) (string, func(*framework.TestCase)) {
	return clusterProviderSyncCase(f, harness, "manifest", "manifest-value", esv1.AuthenticationScopeManifestNamespace)
}

func ClusterProviderProviderNamespace(f *framework.Framework, harness ClusterProviderExternalSecretHarness) (string, func(*framework.TestCase)) {
	return clusterProviderSyncCase(f, harness, "provider", "provider-value", esv1.AuthenticationScopeProviderNamespace)
}

func ClusterProviderManifestNamespaceRecovery(f *framework.Framework, harness ClusterProviderExternalSecretHarness) (string, func(*framework.TestCase)) {
	return clusterProviderRecoveryCase(f, harness, "manifest-recovery", "manifest-recovered", esv1.AuthenticationScopeManifestNamespace)
}

func ClusterProviderProviderNamespaceRecovery(f *framework.Framework, harness ClusterProviderExternalSecretHarness) (string, func(*framework.TestCase)) {
	return clusterProviderRecoveryCase(f, harness, "provider-recovery", "provider-recovered", esv1.AuthenticationScopeProviderNamespace)
}

func ClusterProviderDeniedByConditions(f *framework.Framework, harness ClusterProviderExternalSecretHarness) (string, func(*framework.TestCase)) {
	return "[common] should deny workload namespaces that do not match ClusterProvider conditions", func(tc *framework.TestCase) {
		targetSecretName := "denied-target"
		remoteSecretName := "denied-source"
		expectedMessage := "should-not-sync"

		tc.ExpectedSecret = nil
		tc.ExternalSecret.ObjectMeta.Name = "denied-external-secret"
		tc.ExternalSecret.Spec.Target.Name = targetSecretName
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      remoteSecretName,
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			remoteSecretName: {Value: jsonSecretValue(expectedMessage)},
		}

		var runtime *ClusterProviderExternalSecretRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime = harness.Prepare(tc, ClusterProviderConfig{
				Name:      "deny",
				AuthScope: esv1.AuthenticationScopeManifestNamespace,
				Conditions: []esv1.ClusterSecretStoreCondition{{
					Namespaces: []string{"not-" + f.Namespace.Name},
				}},
			})
			applyClusterProviderExternalSecret(tc, runtime)
		}
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
			waitForExternalSecretStatus(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, corev1.ConditionFalse)
			expectNoSecretInNamespace(tc.Framework, tc.ExternalSecret.Namespace, targetSecretName)
			expectEventMessage(
				tc.Framework,
				tc.ExternalSecret.Namespace,
				tc.ExternalSecret.Name,
				"ExternalSecret",
				fmt.Sprintf("using ClusterProvider %q is not allowed from namespace %q: denied by spec.conditions", runtime.ClusterProviderName, f.Namespace.Name),
			)
		}
	}
}

func clusterProviderSyncCase(f *framework.Framework, harness ClusterProviderExternalSecretHarness, name, expectedValue string, authScope esv1.AuthenticationScope) (string, func(*framework.TestCase)) {
	return fmt.Sprintf("[common] should use %s auth with ClusterProvider", authScope), func(tc *framework.TestCase) {
		targetSecretName := fmt.Sprintf("%s-target", name)
		remoteSecretName := fmt.Sprintf("%s-source", name)

		tc.ExternalSecret.ObjectMeta.Name = fmt.Sprintf("%s-external-secret", name)
		tc.ExternalSecret.Spec.Target.Name = targetSecretName
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      remoteSecretName,
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			remoteSecretName: {Value: jsonSecretValue(expectedValue)},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"value": []byte(expectedValue),
			},
		}
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime := harness.Prepare(tc, ClusterProviderConfig{
				Name:      name,
				AuthScope: authScope,
			})
			applyClusterProviderExternalSecret(tc, runtime)
		}
	}
}

func clusterProviderRecoveryCase(f *framework.Framework, harness ClusterProviderExternalSecretHarness, name, expectedValue string, authScope esv1.AuthenticationScope) (string, func(*framework.TestCase)) {
	return fmt.Sprintf("[common] should recover after repairing ClusterProvider auth with %s scope", authScope), func(tc *framework.TestCase) {
		targetSecretName := fmt.Sprintf("%s-target", name)
		remoteSecretName := fmt.Sprintf("%s-source", name)

		tc.ExpectedSecret = nil
		tc.ExternalSecret.ObjectMeta.Name = fmt.Sprintf("%s-external-secret", name)
		tc.ExternalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Hour}
		tc.ExternalSecret.Spec.Target.Name = targetSecretName
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      remoteSecretName,
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			remoteSecretName: {Value: jsonSecretValue(expectedValue)},
		}

		var runtime *ClusterProviderExternalSecretRuntime
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			runtime = harness.Prepare(tc, ClusterProviderConfig{
				Name:      name,
				AuthScope: authScope,
			})
			if !runtime.SupportsAuthLifecycle() {
				providerName := ""
				if runtime != nil {
					providerName = runtime.ClusterProviderName
				}
				Skip(fmt.Sprintf("provider %q does not support auth lifecycle recovery hooks", providerName))
			}
			applyClusterProviderExternalSecret(tc, runtime)
			runtime.BreakAuth()
		}
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
			waitForExternalSecretStatus(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Name, corev1.ConditionFalse)
			expectNoSecretInNamespace(tc.Framework, tc.ExternalSecret.Namespace, targetSecretName)
			runtime.RepairAuth()
			waitForSecretData(tc.Framework, tc.ExternalSecret.Namespace, targetSecretName, map[string][]byte{
				"value": []byte(expectedValue),
			}, 30*time.Second)
		}
	}
}

func applyClusterProviderExternalSecret(tc *framework.TestCase, runtime *ClusterProviderExternalSecretRuntime) {
	tc.ProviderOverride = runtime.Provider
	tc.ExternalSecret.Spec.SecretStoreRef.Name = runtime.ClusterProviderName
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterProviderKindStr
}

func waitForExternalSecretStatus(f *framework.Framework, namespace, name string, status corev1.ConditionStatus) {
	Eventually(func(g Gomega) {
		var externalSecret esv1.ExternalSecret
		g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, &externalSecret)).To(Succeed())
		condition := esv1.GetExternalSecretCondition(externalSecret.Status, esv1.ExternalSecretReady)
		g.Expect(condition).NotTo(BeNil())
		g.Expect(condition.Status).To(Equal(status))
	}, time.Minute, 5*time.Second).Should(Succeed())
}

func waitForSecretData(f *framework.Framework, namespace, name string, expected map[string][]byte, timeout time.Duration) {
	Eventually(func(g Gomega) {
		var syncedSecret corev1.Secret
		g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, &syncedSecret)).To(Succeed())
		g.Expect(syncedSecret.Type).To(Equal(corev1.SecretTypeOpaque))
		g.Expect(syncedSecret.Data).To(Equal(expected))
	}, timeout, 5*time.Second).Should(Succeed())
}

func expectNoSecretInNamespace(f *framework.Framework, namespace, name string) {
	Consistently(func() bool {
		var secret corev1.Secret
		err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, &secret)
		return apierrors.IsNotFound(err)
	}, 10*time.Second, 5*time.Second).Should(BeTrue())
}

func expectEventMessage(f *framework.Framework, namespace, objectName, objectKind, expectedMessage string) {
	Eventually(func() string {
		events, err := f.KubeClientSet.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{
			FieldSelector: "involvedObject.name=" + objectName + ",involvedObject.kind=" + objectKind,
		})
		Expect(err).NotTo(HaveOccurred())
		messages := make([]string, 0, len(events.Items))
		for _, event := range events.Items {
			if event.Message != "" {
				messages = append(messages, event.Message)
			}
		}
		return fmt.Sprintf("%v", messages)
	}, time.Minute, 5*time.Second).Should(ContainSubstring(expectedMessage))
}

func jsonSecretValue(value string) string {
	return fmt.Sprintf(`{"value":%q}`, value)
}
