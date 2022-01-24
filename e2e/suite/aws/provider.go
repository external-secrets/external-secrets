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

package aws

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	//nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/framework/log"
)

type SMProvider struct {
	ServiceAccountName      string
	ServiceAccountNamespace string

	kid       string
	sak       string
	region    string
	client    *secretsmanager.SecretsManager
	framework *framework.Framework
}

const (
	staticCredentialsSecretName = "provider-secret"
)

func NewSMProvider(f *framework.Framework, kid, sak, region, saName, saNamespace string) *SMProvider {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials: credentials.NewStaticCredentials(kid, sak, ""),
			Region:      aws.String(region),
		},
	})
	if err != nil {
		Fail(err.Error())
	}
	sm := secretsmanager.New(sess)
	prov := &SMProvider{
		ServiceAccountName:      saName,
		ServiceAccountNamespace: saNamespace,
		kid:                     kid,
		sak:                     sak,
		region:                  region,
		client:                  sm,
		framework:               f,
	}

	BeforeEach(func() {
		prov.SetupStaticStore()
		prov.SetupReferencedIRSAStore()
		prov.SetupMountedIRSAStore()
	})

	AfterEach(func() {
		// Cleanup ClusterSecretStore
		err := prov.framework.CRClient.Delete(context.Background(), &esv1alpha1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name: prov.ReferencedIRSAStoreName(),
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	return prov
}

func NewFromEnv(f *framework.Framework) *SMProvider {
	kid := os.Getenv("AWS_ACCESS_KEY_ID")
	sak := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := "eu-west-1"
	saName := os.Getenv("AWS_SA_NAME")
	saNamespace := os.Getenv("AWS_SA_NAMESPACE")
	return NewSMProvider(f, kid, sak, region, saName, saNamespace)
}

// CreateSecret creates a secret at the provider.
func (s *SMProvider) CreateSecret(key, val string) {
	// we re-use some secret names throughout our test suite
	// due to the fact that there is a short delay before the secret is actually deleted
	// we have to retry creating the secret
	attempts := 20
	for {
		log.Logf("creating secret %s / attempts left: %d", key, attempts)
		_, err := s.client.CreateSecret(&secretsmanager.CreateSecretInput{
			Name:         aws.String(key),
			SecretString: aws.String(val),
		})
		if err == nil {
			return
		}
		attempts--
		if attempts < 0 {
			Fail("unable to create secret: " + err.Error())
		}
		<-time.After(time.Second * 5)
	}
}

// DeleteSecret deletes a secret at the provider.
// There may be a short delay between calling this function
// and the removal of the secret on the provider side.
func (s *SMProvider) DeleteSecret(key string) {
	log.Logf("deleting secret %s", key)
	_, err := s.client.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId:                   aws.String(key),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})
	Expect(err).ToNot(HaveOccurred())
}

// MountedIRSAStore is a SecretStore without auth config
// ESO relies on the pod-mounted ServiceAccount when using this store.
func (s *SMProvider) SetupMountedIRSAStore() {
	secretStore := &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.MountedIRSAStoreName(),
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				AWS: &esv1alpha1.AWSProvider{
					Service: esv1alpha1.AWSServiceSecretsManager,
					Region:  s.region,
					Auth:    esv1alpha1.AWSAuth{},
				},
			},
		},
	}
	err := s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s *SMProvider) MountedIRSAStoreName() string {
	return "irsa-mounted-" + s.framework.Namespace.Name
}

// ReferncedIRSAStore is a ClusterStore
// that references a (IRSA-) ServiceAccount in the default namespace.
func (s *SMProvider) SetupReferencedIRSAStore() {
	log.Logf("creating IRSA ClusterSecretStore %s", s.framework.Namespace.Name)
	secretStore := &esv1alpha1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.ReferencedIRSAStoreName(),
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.Background(), s.framework.CRClient, secretStore, func() error {
		secretStore.Spec.Provider = &esv1alpha1.SecretStoreProvider{
			AWS: &esv1alpha1.AWSProvider{
				Service: esv1alpha1.AWSServiceSecretsManager,
				Region:  s.region,
				Auth: esv1alpha1.AWSAuth{
					JWTAuth: &esv1alpha1.AWSJWTAuth{
						ServiceAccountRef: &esmetav1.ServiceAccountSelector{
							Name:      s.ServiceAccountName,
							Namespace: &s.ServiceAccountNamespace,
						},
					},
				},
			},
		}
		return nil
	})
	Expect(err).ToNot(HaveOccurred())
}

func (s *SMProvider) ReferencedIRSAStoreName() string {
	return "irsa-ref-" + s.framework.Namespace.Name
}

// StaticStore is namespaced and references
// static credentials from a secret.
func (s *SMProvider) SetupStaticStore() {
	awsCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      staticCredentialsSecretName,
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			"kid": s.kid,
			"sak": s.sak,
		},
	}
	err := s.framework.CRClient.Create(context.Background(), awsCreds)
	Expect(err).ToNot(HaveOccurred())

	secretStore := &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				AWS: &esv1alpha1.AWSProvider{
					Service: esv1alpha1.AWSServiceSecretsManager,
					Region:  s.region,
					Auth: esv1alpha1.AWSAuth{
						SecretRef: &esv1alpha1.AWSAuthSecretRef{
							AccessKeyID: esmetav1.SecretKeySelector{
								Name: staticCredentialsSecretName,
								Key:  "kid",
							},
							SecretAccessKey: esmetav1.SecretKeySelector{
								Name: staticCredentialsSecretName,
								Key:  "sak",
							},
						},
					},
				},
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
