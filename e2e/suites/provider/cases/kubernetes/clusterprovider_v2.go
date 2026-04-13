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

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("[kubernetes] v2 cluster provider", Label("kubernetes", "v2", "cluster-provider"), func() {
	f := framework.New("eso-kubernetes-v2-clusterprovider")
	prov := NewProvider(f)
	harness := newKubernetesClusterProviderExternalSecretHarness(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("cluster provider external secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.ClusterProviderManifestNamespace(f, harness)),
		Entry(common.ClusterProviderProviderNamespace(f, harness)),
		Entry(common.ClusterProviderManifestNamespaceRecovery(f, harness)),
		Entry(common.ClusterProviderProviderNamespaceRecovery(f, harness)),
		Entry(common.ClusterProviderDeniedByConditions(f, harness)),
	)
})

func newKubernetesClusterProviderExternalSecretHarness(f *framework.Framework) common.ClusterProviderExternalSecretHarness {
	return common.ClusterProviderExternalSecretHarness{
		Prepare: func(tc *framework.TestCase, cfg common.ClusterProviderConfig) *common.ClusterProviderExternalSecretRuntime {
			s := newClusterProviderV2Scenario(f, cfg.Name)
			s.allowRemoteAccessForScope(cfg.AuthScope, cfg.Name)

			clusterProviderName := s.createClusterProvider(cfg.Name, cfg.AuthScope, cfg.Conditions)
			frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

			return &common.ClusterProviderExternalSecretRuntime{
				ClusterProviderName: clusterProviderName,
				Provider:            s,
				BreakAuth: func() {
					updateKubernetesProviderServiceAccount(f, s.providerNamespace, s.providerConfigName(cfg.Name), "missing-service-account")
				},
				RepairAuth: func() {
					updateKubernetesProviderServiceAccount(f, s.providerNamespace, s.providerConfigName(cfg.Name), s.serviceAccount)
				},
			}
		},
	}
}

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

func (s *clusterProviderV2Scenario) allowRemoteAccessForScope(authScope esv1.AuthenticationScope, suffix string) {
	serviceAccountNamespace := s.workloadNamespace
	if authScope == esv1.AuthenticationScopeProviderNamespace {
		serviceAccountNamespace = s.providerNamespace
	}
	s.allowRemoteAccessFrom(serviceAccountNamespace, suffix)
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

func (s *clusterProviderV2Scenario) CreateSecret(key string, val framework.SecretEntry) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key,
			Namespace: s.remoteNamespace,
			Labels:    val.Tags,
		},
		Data: make(map[string][]byte),
	}
	stringMap := make(map[string]string)
	err := json.Unmarshal([]byte(val.Value), &stringMap)
	Expect(err).ToNot(HaveOccurred())

	for k, v := range stringMap {
		secret.Data[k] = []byte(v)
	}
	Expect(s.f.CRClient.Create(GinkgoT().Context(), secret)).To(Succeed())
}

func (s *clusterProviderV2Scenario) DeleteSecret(key string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key,
			Namespace: s.remoteNamespace,
		},
	}
	Expect(s.f.CRClient.Delete(GinkgoT().Context(), secret)).To(Succeed())
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

func externalSecretConditionHasStatus(condition *esv1.ExternalSecretStatusCondition, want corev1.ConditionStatus) bool {
	return condition != nil && condition.Status == want
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
