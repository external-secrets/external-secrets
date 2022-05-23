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
	"errors"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/framework/log"
	common "github.com/external-secrets/external-secrets/e2e/suites/provider/cases/aws"
)

type Provider struct {
	ServiceAccountName      string
	ServiceAccountNamespace string

	region    string
	client    *secretsmanager.SecretsManager
	framework *framework.Framework
}

func NewProvider(f *framework.Framework, kid, sak, region, saName, saNamespace string) *Provider {
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
	prov := &Provider{
		ServiceAccountName:      saName,
		ServiceAccountNamespace: saNamespace,
		region:                  region,
		client:                  sm,
		framework:               f,
	}

	BeforeEach(func() {
		common.SetupStaticStore(f, kid, sak, region, esv1beta1.AWSServiceSecretsManager)
		prov.SetupReferencedIRSAStore()
		prov.SetupMountedIRSAStore()
	})

	AfterEach(func() {
		// Cleanup ClusterSecretStore
		err := prov.framework.CRClient.Delete(context.Background(), &esv1beta1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name: common.ReferencedIRSAStoreName(f),
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	return prov
}

func NewFromEnv(f *framework.Framework) *Provider {
	kid := os.Getenv("AWS_ACCESS_KEY_ID")
	sak := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := "eu-west-1"
	saName := os.Getenv("AWS_SA_NAME")
	saNamespace := os.Getenv("AWS_SA_NAMESPACE")
	return NewProvider(f, kid, sak, region, saName, saNamespace)
}

// CreateSecret creates a secret at the provider.
func (s *Provider) CreateSecret(key string, val framework.SecretEntry) {
	smTags := make([]*secretsmanager.Tag, 0)
	for k, v := range val.Tags {
		smTags = append(smTags, &secretsmanager.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	// we re-use some secret names throughout our test suite
	// due to the fact that there is a short delay before the secret is actually deleted
	// we have to retry creating the secret
	attempts := 20
	for {
		log.Logf("creating secret %s / attempts left: %d", key, attempts)
		_, err := s.client.CreateSecret(&secretsmanager.CreateSecretInput{
			Name:         aws.String(key),
			SecretString: aws.String(val.Value),
			Tags:         smTags,
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
func (s *Provider) DeleteSecret(key string) {
	log.Logf("deleting secret %s", key)
	_, err := s.client.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId:                   aws.String(key),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})
	var nf *secretsmanager.ResourceNotFoundException
	if errors.As(err, &nf) {
		return
	}
	Expect(err).ToNot(HaveOccurred())
}

// MountedIRSAStore is a SecretStore without auth config
// ESO relies on the pod-mounted ServiceAccount when using this store.
func (s *Provider) SetupMountedIRSAStore() {
	secretStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.MountedIRSAStoreName(s.framework),
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				AWS: &esv1beta1.AWSProvider{
					Service: esv1beta1.AWSServiceSecretsManager,
					Region:  s.region,
					Auth:    esv1beta1.AWSAuth{},
				},
			},
		},
	}
	err := s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

// ReferncedIRSAStore is a ClusterStore
// that references a (IRSA-) ServiceAccount in the default namespace.
func (s *Provider) SetupReferencedIRSAStore() {
	log.Logf("creating IRSA ClusterSecretStore %s", s.framework.Namespace.Name)
	secretStore := &esv1beta1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.ReferencedIRSAStoreName(s.framework),
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.Background(), s.framework.CRClient, secretStore, func() error {
		secretStore.Spec.Provider = &esv1beta1.SecretStoreProvider{
			AWS: &esv1beta1.AWSProvider{
				Service: esv1beta1.AWSServiceSecretsManager,
				Region:  s.region,
				Auth: esv1beta1.AWSAuth{
					JWTAuth: &esv1beta1.AWSJWTAuth{
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
