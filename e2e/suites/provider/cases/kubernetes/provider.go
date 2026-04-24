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

	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	kubernetesv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	// nolint
	. "github.com/onsi/gomega"
)

type Provider struct {
	framework *framework.Framework
}

func NewProvider(f *framework.Framework) *Provider {
	prov := &Provider{
		framework: f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *Provider) CreateSecret(key string, val framework.SecretEntry) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key,
			Namespace: s.framework.Namespace.Name,
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
	err = s.framework.CRClient.Create(GinkgoT().Context(), secret)
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) BeforeEach() {
	if framework.IsV2ProviderMode() {
		s.CreateStoreV2()
		s.CreateReferentStoreV2()
		return
	}

	s.CreateStore()
	s.CreateReferentStore()
}

func (s *Provider) DeleteSecret(key string) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key,
			Namespace: s.framework.Namespace.Name,
		},
	}
	err := s.framework.CRClient.Delete(GinkgoT().Context(), secret, &client.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func makeDefaultStore(suffix, namespace string) (*rbac.Role, *rbac.RoleBinding, *esv1.SecretStore) {
	roleName := fmt.Sprintf("%s-%s", "allow-eso-secret-read", suffix)
	role := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
		},
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete", "patch"},
			},
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"selfsubjectrulesreviews"},
			},
		},
	}

	rb := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", "eso-rb", suffix),
			Namespace: namespace,
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		},
		RoleRef: rbac.RoleRef{
			Kind:     "Role",
			Name:     roleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace,
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Kubernetes: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{
						CAProvider: &esv1.CAProvider{
							Type: esv1.CAProviderTypeConfigMap,
							Name: "kube-root-ca.crt",
							Key:  "ca.crt",
						},
					},
					Auth: &esv1.KubernetesAuth{
						ServiceAccount: &esmeta.ServiceAccountSelector{
							Name: "default",
						},
					},
					RemoteNamespace: namespace,
				},
			},
		},
	}

	return role, rb, store
}

func (s *Provider) CreateStore() {
	role, roleBinding, store := makeDefaultStore("", s.framework.Namespace.Name)

	err := s.framework.CRClient.Create(GinkgoT().Context(), role)
	Expect(err).ToNot(HaveOccurred())

	err = s.framework.CRClient.Create(GinkgoT().Context(), roleBinding)
	Expect(err).ToNot(HaveOccurred())

	err = s.framework.CRClient.Create(GinkgoT().Context(), store)
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) CreateReferentStore() {
	role, roleBinding, store := makeDefaultStore("referent", s.framework.Namespace.Name)
	// ServiceAccount Namespace is not set, this will be inferred
	// from the ExternalSecret
	css := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: referentStoreName(s.framework),
		},
		Spec: store.Spec,
	}
	css.Spec.Provider.Kubernetes.Server.CAProvider.Namespace = &s.framework.Namespace.Name

	err := s.framework.CRClient.Create(GinkgoT().Context(), role)
	Expect(err).ToNot(HaveOccurred())

	err = s.framework.CRClient.Create(GinkgoT().Context(), roleBinding)
	Expect(err).ToNot(HaveOccurred())

	err = s.framework.CRClient.Create(GinkgoT().Context(), css)
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) CreateStoreV2() {
	namespace := s.framework.Namespace.Name
	configName := namespace + "-config"

	frameworkv2.CreateKubernetesAccessRole(
		s.framework,
		storeAccessName(namespace, ""),
		frameworkv2.DefaultSAName,
		namespace,
		namespace,
	)

	serviceAccountNamespace := namespace
	frameworkv2.CreateRuntimeSecretStore(
		s.framework,
		namespace,
		namespace,
		frameworkv2.ProviderAddress("kubernetes"),
		createKubernetesProviderConfig(
			s.framework,
			namespace,
			configName,
			"",
			frameworkv2.NewKubernetesStoreProvider(
				namespace,
				frameworkv2.DefaultSAName,
				&serviceAccountNamespace,
				frameworkv2.GetClusterCABundle(s.framework, namespace),
			),
		),
	)
	frameworkv2.WaitForSecretStoreReady(s.framework, namespace, namespace, 30*time.Second)
}

func (s *Provider) CreateReferentStoreV2() {
	namespace := s.framework.Namespace.Name
	referentName := referentStoreName(s.framework)
	configName := referentName + "-config"

	frameworkv2.CreateKubernetesAccessRole(
		s.framework,
		storeAccessName(namespace, "referent"),
		frameworkv2.DefaultSAName,
		namespace,
		namespace,
	)

	frameworkv2.CreateRuntimeClusterSecretStore(
		s.framework,
		referentName,
		frameworkv2.ProviderAddress("kubernetes"),
		createKubernetesProviderConfig(
			s.framework,
			namespace,
			configName,
			"",
			frameworkv2.NewKubernetesStoreProvider(
				namespace,
				frameworkv2.DefaultSAName,
				nil,
				frameworkv2.GetClusterCABundle(s.framework, namespace),
			),
		),
		nil,
	)
	frameworkv2.WaitForClusterSecretStoreReady(s.framework, referentName, 30*time.Second)
}

func referentStoreName(f *framework.Framework) string {
	return fmt.Sprintf("%s-referent", f.Namespace.Name)
}

func updateKubernetesStoreServiceAccount(f *framework.Framework, ref esv1.SecretStoreRef, namespace, serviceAccountName string) {
	switch ref.Kind {
	case esv1.ClusterSecretStoreKind:
		var store esv1.ClusterSecretStore
		Expect(f.CRClient.Get(context.Background(), client.ObjectKey{Name: ref.Name}, &store)).To(Succeed())
		updateKubernetesProviderConfigServiceAccount(f, store.Spec.ProviderRef, namespace, serviceAccountName)
	default:
		var store esv1.SecretStore
		Expect(f.CRClient.Get(context.Background(), client.ObjectKey{Name: ref.Name, Namespace: namespace}, &store)).To(Succeed())
		updateKubernetesProviderConfigServiceAccount(f, store.Spec.ProviderRef, store.Namespace, serviceAccountName)
	}
}

func createKubernetesProviderConfig(f *framework.Framework, namespace, name, providerRefNamespace string, provider *esv1.SecretStoreProvider) *esv1.StoreProviderRef {
	Expect(provider).NotTo(BeNil())
	Expect(provider.Kubernetes).NotTo(BeNil())
	Expect(f.CreateObjectWithRetry(&kubernetesv2alpha1.Kubernetes{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "provider.external-secrets.io/v2alpha1",
			Kind:       "Kubernetes",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *provider.Kubernetes,
	})).To(Succeed())
	return kubernetesStoreProviderRef(name, providerRefNamespace)
}

func kubernetesStoreProviderRef(name, namespace string) *esv1.StoreProviderRef {
	return &esv1.StoreProviderRef{
		APIVersion: "provider.external-secrets.io/v2alpha1",
		Kind:       "Kubernetes",
		Name:       name,
		Namespace:  namespace,
	}
}

func updateKubernetesProviderConfigServiceAccount(f *framework.Framework, providerRef *esv1.StoreProviderRef, fallbackNamespace, serviceAccountName string) {
	Expect(providerRef).NotTo(BeNil())
	resolvedNamespace := providerRef.Namespace
	if resolvedNamespace == "" {
		resolvedNamespace = fallbackNamespace
	}

	var providerConfig kubernetesv2alpha1.Kubernetes
	Expect(f.CRClient.Get(context.Background(), client.ObjectKey{Name: providerRef.Name, Namespace: resolvedNamespace}, &providerConfig)).To(Succeed())
	if providerConfig.Spec.Auth == nil {
		providerConfig.Spec.Auth = &esv1.KubernetesAuth{}
	}
	if providerConfig.Spec.Auth.ServiceAccount == nil {
		providerConfig.Spec.Auth.ServiceAccount = &esmeta.ServiceAccountSelector{}
	}
	providerConfig.Spec.Auth.ServiceAccount.Name = serviceAccountName
	Expect(f.CRClient.Update(context.Background(), &providerConfig)).To(Succeed())
}

func storeAccessName(namespace, suffix string) string {
	if suffix == "" {
		return fmt.Sprintf("%s-provider-access", namespace)
	}
	return fmt.Sprintf("%s-provider-access-%s", namespace, suffix)
}
