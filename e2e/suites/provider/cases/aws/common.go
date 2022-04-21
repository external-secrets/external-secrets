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
package common

import (
	"context"

	// nolint
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

const (
	WithReferencedIRSA          = "with referenced IRSA"
	WithMountedIRSA             = "with mounted IRSA"
	StaticCredentialsSecretName = "provider-secret"
)

func ReferencedIRSAStoreName(f *framework.Framework) string {
	return "irsa-ref-" + f.Namespace.Name
}

func MountedIRSAStoreName(f *framework.Framework) string {
	return "irsa-mounted-" + f.Namespace.Name
}

func UseClusterSecretStore(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1beta1.ClusterSecretStoreKind
	tc.ExternalSecret.Spec.SecretStoreRef.Name = ReferencedIRSAStoreName(tc.Framework)
}

func UseMountedIRSAStore(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1beta1.SecretStoreKind
	tc.ExternalSecret.Spec.SecretStoreRef.Name = MountedIRSAStoreName(tc.Framework)
}

// StaticStore is namespaced and references
// static credentials from a secret.
func SetupStaticStore(f *framework.Framework, kid, sak, region string, serviceType esv1beta1.AWSServiceType) {
	awsCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StaticCredentialsSecretName,
			Namespace: f.Namespace.Name,
		},
		StringData: map[string]string{
			"kid": kid,
			"sak": sak,
		},
	}
	err := f.CRClient.Create(context.Background(), awsCreds)
	Expect(err).ToNot(HaveOccurred())

	secretStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Namespace.Name,
			Namespace: f.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				AWS: &esv1beta1.AWSProvider{
					Service: serviceType,
					Region:  region,
					Auth: esv1beta1.AWSAuth{
						SecretRef: &esv1beta1.AWSAuthSecretRef{
							AccessKeyID: esmetav1.SecretKeySelector{
								Name: StaticCredentialsSecretName,
								Key:  "kid",
							},
							SecretAccessKey: esmetav1.SecretKeySelector{
								Name: StaticCredentialsSecretName,
								Key:  "sak",
							},
						},
					},
				},
			},
		},
	}
	err = f.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
