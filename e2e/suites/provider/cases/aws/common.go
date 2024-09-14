//Copyright External Secrets Inc. All Rights Reserved

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

	IAMRoleExternalID  = "arn:aws:iam::783882199045:role/eso-e2e-external-id"
	IAMRoleSessionTags = "arn:aws:iam::783882199045:role/eso-e2e-session-tags"

	IAMTrustedExternalID = "eso-e2e-ext-id"
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
	ExternalIDStoreName   = "aws-ext-id"
	SessionTagsStoreName  = "aws-sess-tags"
	staticKeyID           = "kid"
	staticSecretAccessKey = "sak"
	staticySessionToken   = "st"
)

func newStaticStoreProvider(serviceType esv1beta1.AWSServiceType, region, secretName, role, externalID string, sessionTags []*esv1beta1.Tag) *esv1beta1.SecretStoreProvider {
	return &esv1beta1.SecretStoreProvider{
		AWS: &esv1beta1.AWSProvider{
			Service:     serviceType,
			Region:      region,
			Role:        role,
			ExternalID:  externalID,
			SessionTags: sessionTags,
			Auth: esv1beta1.AWSAuth{
				SecretRef: &esv1beta1.AWSAuthSecretRef{
					AccessKeyID: esmetav1.SecretKeySelector{
						Name: secretName,
						Key:  staticKeyID,
					},
					SecretAccessKey: esmetav1.SecretKeySelector{
						Name: secretName,
						Key:  staticSecretAccessKey,
					},
					SessionToken: &esmetav1.SecretKeySelector{
						Name: secretName,
						Key:  staticySessionToken,
					},
				},
			},
		},
	}
}

// SessionTagsStore is namespaced and references
// static credentials from a secret. It assumes a role and specifies session tags
func SetupSessionTagsStore(f *framework.Framework, kid, sak, st, region, role string, sessionTags []*esv1beta1.Tag, serviceType esv1beta1.AWSServiceType) {
	credsName := "provider-secret-sess-tags"
	awsCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credsName,
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
			Name:      SessionTagsStoreName,
			Namespace: f.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: newStaticStoreProvider(serviceType, region, credsName, role, "", sessionTags),
		},
	}
	err = f.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

// ExternalIDStore is namespaced and references
// static credentials from a secret. It assumes a role and specifies an externalID
func SetupExternalIDStore(f *framework.Framework, kid, sak, st, region, role, externalID string, sessionTags []*esv1beta1.Tag, serviceType esv1beta1.AWSServiceType) {
	credsName := "provider-secret-ext-id"
	awsCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credsName,
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
			Name:      ExternalIDStoreName,
			Namespace: f.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: newStaticStoreProvider(serviceType, region, credsName, role, externalID, sessionTags),
		},
	}
	err = f.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
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
			Provider: newStaticStoreProvider(serviceType, region, StaticCredentialsSecretName, "", "", nil),
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
			Provider: newStaticStoreProvider(serviceType, region, StaticReferentCredentialsSecretName, "", "", nil),
		},
	}
	err = f.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func ReferentSecretStoreName(f *framework.Framework) string {
	return "referent-auth" + f.Namespace.Name
}
