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

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	WithReferencedIRSA                  = "with referenced IRSA"
	WithMountedIRSA                     = "with mounted IRSA"
	StaticCredentialsSecretName         = "provider-secret"
	StaticReferentCredentialsSecretName = "referent-provider-secret"
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

const (
	StaticStoreName       = "aws-static-creds"
	staticKeyID           = "kid"
	staticSecretAccessKey = "sak"
	staticySessionToken   = "st"
)

func newStaticStoreProvider(serviceType esv1beta1.AWSServiceType, region, secretName string) *esv1beta1.SecretStoreProvider {
	return &esv1beta1.SecretStoreProvider{
		AWS: &esv1beta1.AWSProvider{
			Service: serviceType,
			Region:  region,
			Auth: esv1beta1.AWSAuth{
				SecretRef: &esv1beta1.AWSAuthSecretRef{
					AccessKeyID: esmetav1.SecretKeySelector{
						Name: StaticReferentCredentialsSecretName,
						Key:  staticKeyID,
					},
					SecretAccessKey: esmetav1.SecretKeySelector{
						Name: StaticReferentCredentialsSecretName,
						Key:  staticSecretAccessKey,
					},
					SessionToken: &esmetav1.SecretKeySelector{
						Name: StaticReferentCredentialsSecretName,
						Key:  staticySessionToken,
					},
				},
			},
		},
	}
}

// StaticStore is namespaced and references
// static credentials from a secret.
func SetupStaticStore(f *framework.Framework, kid, sak, st, region string, serviceType esv1beta1.AWSServiceType) {
	awsCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StaticCredentialsSecretName,
			Namespace: f.Namespace.Name,
		},
		StringData: map[string]string{
			staticKeyID:           kid,
			staticSecretAccessKey: sak,
			staticySessionToken:   st,
		},
	}
	err := f.CRClient.Create(context.Background(), awsCreds)
	Expect(err).ToNot(HaveOccurred())

	secretStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StaticStoreName,
			Namespace: f.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: newStaticStoreProvider(serviceType, region, StaticCredentialsSecretName),
		},
	}
	err = f.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

// CreateReferentStaticStore creates a CSS with referent auth and
// creates a secret with static authentication credentials in the ExternalSecret namespace.
func CreateReferentStaticStore(f *framework.Framework, kid, sak, st, region string, serviceType esv1beta1.AWSServiceType) {
	ns := f.Namespace.Name

	awsCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StaticReferentCredentialsSecretName,
			Namespace: ns,
		},
		StringData: map[string]string{
			staticKeyID:           kid,
			staticSecretAccessKey: sak,
			staticySessionToken:   st,
		},
	}
	err := f.CRClient.Create(context.Background(), awsCreds)
	Expect(err).ToNot(HaveOccurred())

	secretStore := &esv1beta1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: ReferentSecretStoreName(f),
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: newStaticStoreProvider(serviceType, region, StaticReferentCredentialsSecretName),
		},
	}
	err = f.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func ReferentSecretStoreName(f *framework.Framework) string {
	return "referent-auth" + f.Namespace.Name
}
