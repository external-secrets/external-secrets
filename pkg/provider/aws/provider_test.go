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
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager"
)

func TestProvider(t *testing.T) {
	cl := clientfake.NewClientBuilder().Build()
	p := Provider{}

	tbl := []struct {
		test    string
		store   esv1alpha1.GenericStore
		expType interface{}
		expErr  bool
	}{
		{
			test:   "should not create provider due to nil store",
			store:  nil,
			expErr: true,
		},
		{
			test:   "should not create provider due to missing provider",
			expErr: true,
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{},
			},
		},
		{
			test:   "should not create provider due to missing provider field",
			expErr: true,
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{},
				},
			},
		},
		{
			test:    "should create parameter store client",
			expErr:  false,
			expType: &parameterstore.ParameterStore{},
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Service: esv1alpha1.AWSServiceParameterStore,
						},
					},
				},
			},
		},
		{
			test:    "should create secretsmanager client",
			expErr:  false,
			expType: &secretsmanager.SecretsManager{},
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Service: esv1alpha1.AWSServiceSecretsManager,
						},
					},
				},
			},
		},
		{
			test:   "invalid service should return an error",
			expErr: true,
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Service: "HIHIHIHHEHEHEHEHEHE",
						},
					},
				},
			},
		},
		{
			test:   "newSession error should be returned",
			expErr: true,
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Service: esv1alpha1.AWSServiceParameterStore,
							Auth: esv1alpha1.AWSAuth{
								SecretRef: &esv1alpha1.AWSAuthSecretRef{
									AccessKeyID: esmeta.SecretKeySelector{
										Name:      "foo",
										Namespace: aws.String("NOOP"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for i := range tbl {
		row := tbl[i]
		t.Run(row.test, func(t *testing.T) {
			sc, err := p.NewClient(context.TODO(), row.store, cl, "foo")
			if row.expErr {
				assert.Error(t, err)
				assert.Nil(t, sc)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, sc)
				assert.IsType(t, row.expType, sc)
			}
		})
	}
}
