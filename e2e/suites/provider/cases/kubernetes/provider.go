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
	"context"
	"encoding/json"
	"fmt"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
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
	err = s.framework.CRClient.Create(context.Background(), secret)
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) BeforeEach() {
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
	err := s.framework.CRClient.Delete(context.Background(), secret, &client.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func makeDefaultStore(suffix, namespace string) (*rbac.Role, *rbac.RoleBinding, *esv1beta1.SecretStore) {
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

	store := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace,
			Namespace: namespace,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Kubernetes: &esv1beta1.KubernetesProvider{
					Server: esv1beta1.KubernetesServer{
						CAProvider: &esv1beta1.CAProvider{
							Type: esv1beta1.CAProviderTypeConfigMap,
							Name: "kube-root-ca.crt",
							Key:  "ca.crt",
						},
					},
					Auth: esv1beta1.KubernetesAuth{
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
	rb, role, store := makeDefaultStore("", s.framework.Namespace.Name)

	err := s.framework.CRClient.Create(context.Background(), role)
	Expect(err).ToNot(HaveOccurred())

	err = s.framework.CRClient.Create(context.Background(), rb)
	Expect(err).ToNot(HaveOccurred())

	err = s.framework.CRClient.Create(context.Background(), store)
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) CreateReferentStore() {
	rb, role, store := makeDefaultStore("referent", s.framework.Namespace.Name)
	// ServiceAccount Namespace is not set, this will be inferred
	// from the ExternalSecret
	css := &esv1beta1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: referentStoreName(s.framework),
		},
		Spec: store.Spec,
	}
	css.Spec.Provider.Kubernetes.Server.CAProvider.Namespace = &s.framework.Namespace.Name

	err := s.framework.CRClient.Create(context.Background(), role)
	Expect(err).ToNot(HaveOccurred())

	err = s.framework.CRClient.Create(context.Background(), rb)
	Expect(err).ToNot(HaveOccurred())

	err = s.framework.CRClient.Create(context.Background(), css)
	Expect(err).ToNot(HaveOccurred())
}

func referentStoreName(f *framework.Framework) string {
	return fmt.Sprintf("%s-referent", f.Namespace.Name)
}
