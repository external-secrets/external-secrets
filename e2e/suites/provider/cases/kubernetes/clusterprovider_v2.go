/*
Copyright © 2026 ESO Maintainer Team

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

package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("[kubernetes] v2 cluster provider", Label("kubernetes", "v2", "cluster-provider"), func() {
	f := framework.New("eso-kubernetes-v2-clusterprovider")

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	It("uses the manifest namespace for auth when authenticationScope=ManifestNamespace", func() {
		s := newClusterProviderV2Scenario(f, "manifest")
		s.allowRemoteAccessFrom(s.workloadNamespace, "manifest")

		remoteSecretName := fmt.Sprintf("%s-source", s.namePrefix)
		targetSecretName := fmt.Sprintf("%s-target", s.namePrefix)
		s.createRemoteSecret(remoteSecretName, "manifest-value")

		clusterProviderName := s.createClusterProvider("manifest", esv1.AuthenticationScopeManifestNamespace, nil)
		frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

		externalSecretName := s.createExternalSecret(clusterProviderName, targetSecretName, remoteSecretName)
		s.waitForExternalSecretValue(externalSecretName, targetSecretName, "manifest-value")
	})

	It("uses the providerRef namespace for auth when authenticationScope=ProviderNamespace", func() {
		s := newClusterProviderV2Scenario(f, "provider")
		s.allowRemoteAccessFrom(s.providerNamespace, "provider")

		remoteSecretName := fmt.Sprintf("%s-source", s.namePrefix)
		targetSecretName := fmt.Sprintf("%s-target", s.namePrefix)
		s.createRemoteSecret(remoteSecretName, "provider-value")

		clusterProviderName := s.createClusterProvider("provider", esv1.AuthenticationScopeProviderNamespace, nil)
		frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

		externalSecretName := s.createExternalSecret(clusterProviderName, targetSecretName, remoteSecretName)
		s.waitForExternalSecretValue(externalSecretName, targetSecretName, "provider-value")
	})

	It("recovers after repairing cluster provider auth when authenticationScope=ManifestNamespace", func() {
		s := newClusterProviderV2Scenario(f, "manifest-recovery")
		s.allowRemoteAccessFrom(s.workloadNamespace, "manifest-recovery")

		remoteSecretName := fmt.Sprintf("%s-source", s.namePrefix)
		targetSecretName := fmt.Sprintf("%s-target", s.namePrefix)
		s.createRemoteSecret(remoteSecretName, "manifest-recovered")

		clusterProviderName := s.createClusterProvider("manifest-recovery", esv1.AuthenticationScopeManifestNamespace, nil)
		frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

		updateKubernetesProviderServiceAccount(f, s.providerNamespace, s.providerConfigName("manifest-recovery"), "missing-service-account")

		externalSecretName := s.createExternalSecret(clusterProviderName, targetSecretName, remoteSecretName)
		s.waitForExternalSecretFailure(externalSecretName)
		s.expectNoTargetSecret(targetSecretName)

		updateKubernetesProviderServiceAccount(f, s.providerNamespace, s.providerConfigName("manifest-recovery"), s.serviceAccount)

		s.waitForExternalSecretValue(externalSecretName, targetSecretName, "manifest-recovered")
	})

	It("recovers after repairing cluster provider auth when authenticationScope=ProviderNamespace", func() {
		s := newClusterProviderV2Scenario(f, "provider-recovery")
		s.allowRemoteAccessFrom(s.providerNamespace, "provider-recovery")

		remoteSecretName := fmt.Sprintf("%s-source", s.namePrefix)
		targetSecretName := fmt.Sprintf("%s-target", s.namePrefix)
		s.createRemoteSecret(remoteSecretName, "provider-recovered")

		clusterProviderName := s.createClusterProvider("provider-recovery", esv1.AuthenticationScopeProviderNamespace, nil)
		frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

		updateKubernetesProviderServiceAccount(f, s.providerNamespace, s.providerConfigName("provider-recovery"), "missing-service-account")

		externalSecretName := s.createExternalSecret(clusterProviderName, targetSecretName, remoteSecretName)
		s.waitForExternalSecretFailure(externalSecretName)
		s.expectNoTargetSecret(targetSecretName)

		updateKubernetesProviderServiceAccount(f, s.providerNamespace, s.providerConfigName("provider-recovery"), s.serviceAccount)

		s.waitForExternalSecretValue(externalSecretName, targetSecretName, "provider-recovered")
	})

	It("denies workload namespaces that do not match ClusterProvider conditions", func() {
		s := newClusterProviderV2Scenario(f, "deny")
		s.allowRemoteAccessFrom(s.workloadNamespace, "deny")

		remoteSecretName := fmt.Sprintf("%s-source", s.namePrefix)
		targetSecretName := fmt.Sprintf("%s-target", s.namePrefix)
		s.createRemoteSecret(remoteSecretName, "should-not-sync")

		clusterProviderName := s.createClusterProvider("deny", esv1.AuthenticationScopeManifestNamespace, []esv1.ClusterSecretStoreCondition{{
			Namespaces: []string{"not-" + s.workloadNamespace},
		}})
		frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

		externalSecretName := s.createExternalSecret(clusterProviderName, targetSecretName, remoteSecretName)
		s.waitForExternalSecretFailure(externalSecretName)
		s.expectNoTargetSecret(targetSecretName)
		s.expectExternalSecretEvent(externalSecretName, fmt.Sprintf("using ClusterProvider %q is not allowed from namespace %q: denied by spec.conditions", clusterProviderName, s.workloadNamespace))
	})
})

type clusterProviderV2Scenario struct {
	f                 *framework.Framework
	namePrefix        string
	workloadNamespace string
	providerNamespace string
	remoteNamespace   string
	serviceAccount    string
	caBundle          []byte
}

func newClusterProviderV2Scenario(f *framework.Framework, prefix string) *clusterProviderV2Scenario {
	s := &clusterProviderV2Scenario{
		f:                 f,
		namePrefix:        fmt.Sprintf("%s-%s", f.Namespace.Name, prefix),
		workloadNamespace: f.Namespace.Name,
		serviceAccount:    "eso-auth",
		caBundle:          frameworkv2.GetClusterCABundle(f, f.Namespace.Name),
	}

	s.providerNamespace = createE2ENamespace(f, prefix+"-provider")
	s.remoteNamespace = createE2ENamespace(f, prefix+"-remote")

	s.createServiceAccount(s.workloadNamespace)
	s.createServiceAccount(s.providerNamespace)

	return s
}

func (s *clusterProviderV2Scenario) createServiceAccount(namespace string) {
	Expect(s.f.CRClient.Create(context.Background(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.serviceAccount,
			Namespace: namespace,
		},
	})).To(Succeed())
}

func (s *clusterProviderV2Scenario) allowRemoteAccessFrom(serviceAccountNamespace, suffix string) {
	frameworkv2.CreateKubernetesAccessRole(
		s.f,
		fmt.Sprintf("%s-access-%s", s.namePrefix, suffix),
		s.serviceAccount,
		serviceAccountNamespace,
		s.remoteNamespace,
	)
}

func (s *clusterProviderV2Scenario) createClusterProvider(suffix string, authScope esv1.AuthenticationScope, conditions []esv1.ClusterSecretStoreCondition) string {
	providerConfigName := s.providerConfigName(suffix)
	frameworkv2.CreateKubernetesProvider(
		s.f,
		s.providerNamespace,
		providerConfigName,
		s.remoteNamespace,
		s.serviceAccount,
		nil,
		s.caBundle,
	)

	clusterProviderName := fmt.Sprintf("%s-cluster-provider-%s", s.namePrefix, suffix)
	frameworkv2.CreateClusterProviderConnection(
		s.f,
		clusterProviderName,
		frameworkv2.ProviderAddress("kubernetes"),
		kubernetesProviderAPIVersion,
		"Kubernetes",
		providerConfigName,
		s.providerNamespace,
		authScope,
		conditions,
	)
	return clusterProviderName
}

func (s *clusterProviderV2Scenario) providerConfigName(suffix string) string {
	return fmt.Sprintf("%s-config-%s", s.namePrefix, suffix)
}

func (s *clusterProviderV2Scenario) createRemoteSecret(name, value string) {
	Expect(s.f.CRClient.Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.remoteNamespace,
		},
		Data: map[string][]byte{
			"value": []byte(value),
		},
	})).To(Succeed())
}

func (s *clusterProviderV2Scenario) createExternalSecret(clusterProviderName, targetSecretName, remoteSecretName string) string {
	externalSecretName := fmt.Sprintf("%s-external-secret", s.namePrefix)
	Expect(s.f.CRClient.Create(context.Background(), &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalSecretName,
			Namespace: s.workloadNamespace,
		},
		Spec: esv1.ExternalSecretSpec{
			RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
			SecretStoreRef: esv1.SecretStoreRef{
				Name: clusterProviderName,
				Kind: esv1.ClusterProviderKindStr,
			},
			Target: esv1.ExternalSecretTarget{
				Name: targetSecretName,
			},
			Data: []esv1.ExternalSecretData{{
				SecretKey: "value",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{
					Key:      remoteSecretName,
					Property: "value",
				},
			}},
		},
	})).To(Succeed())

	DeferCleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		err := deleteExternalSecretAndWait(ctx, s.f.CRClient, types.NamespacedName{
			Name:      externalSecretName,
			Namespace: s.workloadNamespace,
		})
		Expect(err).NotTo(HaveOccurred())
	})

	return externalSecretName
}

func (s *clusterProviderV2Scenario) waitForExternalSecretValue(externalSecretName, targetSecretName, expectedValue string) {
	Eventually(func(g Gomega) {
		var externalSecret esv1.ExternalSecret
		g.Expect(s.f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      externalSecretName,
			Namespace: s.workloadNamespace,
		}, &externalSecret)).To(Succeed())
		condition := esv1.GetExternalSecretCondition(externalSecret.Status, esv1.ExternalSecretReady)
		g.Expect(condition).NotTo(BeNil())
		g.Expect(externalSecretConditionHasStatus(condition, corev1.ConditionTrue)).To(BeTrue())

		var syncedSecret corev1.Secret
		g.Expect(s.f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      targetSecretName,
			Namespace: s.workloadNamespace,
		}, &syncedSecret)).To(Succeed())
		g.Expect(syncedSecret.Data["value"]).To(Equal([]byte(expectedValue)))
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
}

func (s *clusterProviderV2Scenario) waitForExternalSecretFailure(externalSecretName string) {
	Eventually(func(g Gomega) {
		var externalSecret esv1.ExternalSecret
		g.Expect(s.f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      externalSecretName,
			Namespace: s.workloadNamespace,
		}, &externalSecret)).To(Succeed())
		condition := esv1.GetExternalSecretCondition(externalSecret.Status, esv1.ExternalSecretReady)
		g.Expect(condition).NotTo(BeNil())
		g.Expect(externalSecretConditionHasStatus(condition, corev1.ConditionFalse)).To(BeTrue())
		g.Expect(condition.Reason).To(Equal(esv1.ConditionReasonSecretSyncedError))
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
}

func externalSecretConditionHasStatus(condition *esv1.ExternalSecretStatusCondition, want corev1.ConditionStatus) bool {
	return condition != nil && condition.Status == want
}

func deleteExternalSecretAndWait(ctx context.Context, kubeClient client.Client, key types.NamespacedName) error {
	externalSecret := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}

	err := kubeClient.Delete(ctx, externalSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return wait.PollUntilContextTimeout(ctx, defaultV2PollInterval, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		var existing esv1.ExternalSecret
		err := kubeClient.Get(ctx, key, &existing)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})
}

func (s *clusterProviderV2Scenario) expectNoTargetSecret(targetSecretName string) {
	Consistently(func() bool {
		var syncedSecret corev1.Secret
		err := s.f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      targetSecretName,
			Namespace: s.workloadNamespace,
		}, &syncedSecret)
		return apierrors.IsNotFound(err)
	}, 10*time.Second, defaultV2PollInterval).Should(BeTrue())
}

func (s *clusterProviderV2Scenario) expectExternalSecretEvent(externalSecretName, expectedMessage string) {
	Eventually(func() string {
		events, err := s.f.KubeClientSet.CoreV1().Events(s.workloadNamespace).List(context.Background(), metav1.ListOptions{
			FieldSelector: "involvedObject.name=" + externalSecretName + ",involvedObject.kind=ExternalSecret",
		})
		Expect(err).NotTo(HaveOccurred())
		messages := make([]string, 0, len(events.Items))
		for _, event := range events.Items {
			if event.Message != "" {
				messages = append(messages, event.Message)
			}
		}
		return strings.Join(messages, "\n")
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(ContainSubstring(expectedMessage))
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

		err = wait.PollUntilContextTimeout(ctx, defaultV2PollInterval, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
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
