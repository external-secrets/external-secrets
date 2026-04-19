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

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"

	. "github.com/onsi/gomega"
)

const (
	StaticAccessKeyIDKey     = "kid"
	StaticSecretAccessKeyKey = "sak"
	StaticSessionTokenKey    = "st"
)

type V2ClusterProviderScenario struct {
	AuthScope            esv1.AuthenticationScope
	ConfigName           string
	ConfigNamespace      string
	NamePrefix           string
	ProviderNamespace    string
	ProviderRefNamespace string
	WorkloadNamespace    string
}

func CredentialsSecretName(name string) string {
	return name + "-credentials"
}

func StaticCredentialsSecretData(kid, sak, st string) map[string]string {
	return map[string]string{
		StaticAccessKeyIDKey:     kid,
		StaticSecretAccessKeyKey: sak,
		StaticSessionTokenKey:    st,
	}
}

func ProviderConfigNamespace(authScope esv1.AuthenticationScope, providerNamespace, workloadNamespace string) string {
	if authScope == esv1.AuthenticationScopeProviderNamespace {
		return providerNamespace
	}
	return workloadNamespace
}

func ProviderReferenceNamespace(authScope esv1.AuthenticationScope, providerNamespace string) string {
	if authScope == esv1.AuthenticationScopeProviderNamespace {
		return providerNamespace
	}
	return ""
}

func NewV2ClusterProviderScenario(workloadNamespace, prefix string, authScope esv1.AuthenticationScope, createProviderNamespace func(prefix string) string) V2ClusterProviderScenario {
	providerNamespace := workloadNamespace
	if authScope == esv1.AuthenticationScopeProviderNamespace && createProviderNamespace != nil {
		providerNamespace = createProviderNamespace(prefix + "-provider")
	}

	return V2ClusterProviderScenario{
		AuthScope:            authScope,
		ConfigName:           fmt.Sprintf("%s-config", prefix),
		ConfigNamespace:      ProviderConfigNamespace(authScope, providerNamespace, workloadNamespace),
		NamePrefix:           fmt.Sprintf("%s-%s", workloadNamespace, prefix),
		ProviderNamespace:    providerNamespace,
		ProviderRefNamespace: ProviderReferenceNamespace(authScope, providerNamespace),
		WorkloadNamespace:    workloadNamespace,
	}
}

func (s V2ClusterProviderScenario) ClusterProviderName() string {
	return fmt.Sprintf("%s-cluster-provider", s.NamePrefix)
}

func WaitForPushSecretStatus(f *framework.Framework, namespace, name string, status corev1.ConditionStatus) {
	Eventually(func(g Gomega) {
		var ps esv1alpha1.PushSecret
		g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, &ps)).To(Succeed())
		g.Expect(ps.Status.Conditions).NotTo(BeEmpty())
		for _, condition := range ps.Status.Conditions {
			if condition.Type == esv1alpha1.PushSecretReady && condition.Status == status {
				return
			}
		}
		g.Expect(false).To(BeTrue())
	}, time.Minute, 5*time.Second).Should(Succeed())
}

func ExpectPushSecretEventMessage(f *framework.Framework, namespace, objectName, expectedMessage string) {
	Eventually(func() string {
		events, err := f.KubeClientSet.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{
			FieldSelector: "involvedObject.name=" + objectName + ",involvedObject.kind=PushSecret",
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

func PushSecretMetadataWithRemoteNamespace(namespace string) *apiextensionsv1.JSON {
	return &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"remoteNamespace":"%s"}}`, namespace))}
}
