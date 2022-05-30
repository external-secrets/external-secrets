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

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
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

func (s *Provider) CreateStore() {
	By("creating a role binding to allow default service account to read secrets")
	role := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-eso-secret-read",
			Namespace: s.framework.Namespace.Name,
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
				Resources: []string{"selfsubjectaccessreviews", "selfsubjectrulesreviews"},
			},
		},
	}
	err := s.framework.CRClient.Create(context.Background(), role)
	Expect(err).ToNot(HaveOccurred())

	rb := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eso-rb",
			Namespace: s.framework.Namespace.Name,
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: s.framework.Namespace.Name,
			},
		},
		RoleRef: rbac.RoleRef{
			Kind:     "Role",
			Name:     "allow-eso-secret-read",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	err = s.framework.CRClient.Create(context.Background(), rb)
	Expect(err).ToNot(HaveOccurred())

	By("creating a secret store for credentials")
	fakeStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
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
					RemoteNamespace: s.framework.Namespace.Name,
				},
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), fakeStore)
	Expect(err).ToNot(HaveOccurred())
}
