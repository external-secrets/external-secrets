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

package auth

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awssess "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/stretchr/testify/assert"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakesess "github.com/external-secrets/external-secrets/pkg/provider/aws/auth/fake"
)

func TestNewSession(t *testing.T) {
	rows := []TestSessionRow{
		{
			name:      "nil store",
			expectErr: "found nil store",
			store:     nil,
		},
		{
			name:      "not store spec",
			expectErr: "storeSpec is missing provider",
			store:     &esv1alpha1.SecretStore{},
		},
		{
			name:      "store spec has no provider",
			expectErr: "storeSpec is missing provider",
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{},
			},
		},
		{
			name:      "spec has no awssm field",
			expectErr: "Missing AWS field",
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{},
				},
			},
		},
		{
			name: "configure aws using environment variables",
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{},
					},
				},
			},
			env: map[string]string{
				"AWS_ACCESS_KEY_ID":     "1111",
				"AWS_SECRET_ACCESS_KEY": "2222",
			},
			expectProvider:    true,
			expectedKeyID:     "1111",
			expectedSecretKey: "2222",
		},
		{
			name: "configure aws using environment variables + assume role",
			stsProvider: func(*awssess.Session) stsiface.STSAPI {
				return &fakesess.AssumeRoler{
					AssumeRoleFunc: func(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
						assert.Equal(t, *input.RoleArn, "foo-bar-baz")
						return &sts.AssumeRoleOutput{
							AssumedRoleUser: &sts.AssumedRoleUser{
								Arn:           aws.String("1123132"),
								AssumedRoleId: aws.String("xxxxx"),
							},
							Credentials: &sts.Credentials{
								AccessKeyId:     aws.String("3333"),
								SecretAccessKey: aws.String("4444"),
								Expiration:      aws.Time(time.Now().Add(time.Hour)),
								SessionToken:    aws.String("6666"),
							},
						}, nil
					},
				}
			},
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Role: "foo-bar-baz",
						},
					},
				},
			},
			env: map[string]string{
				"AWS_ACCESS_KEY_ID":     "1111",
				"AWS_SECRET_ACCESS_KEY": "2222",
			},
			expectProvider:    true,
			expectedKeyID:     "3333",
			expectedSecretKey: "4444",
		},
		{
			name:      "error out when secret with credentials does not exist",
			namespace: "foo",
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Auth: esv1alpha1.AWSAuth{
								SecretRef: &esv1alpha1.AWSAuthSecretRef{
									AccessKeyID: esmeta.SecretKeySelector{
										Name: "othersecret",
										Key:  "one",
									},
									SecretAccessKey: esmeta.SecretKeySelector{
										Name: "othersecret",
										Key:  "two",
									},
								},
							},
						},
					},
				},
			},
			expectErr: `secrets "othersecret" not found`,
		},
		{
			name:      "use credentials from secret to configure aws",
			namespace: "foo",
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Auth: esv1alpha1.AWSAuth{
								SecretRef: &esv1alpha1.AWSAuthSecretRef{
									AccessKeyID: esmeta.SecretKeySelector{
										Name: "onesecret",
										// Namespace is not set
										Key: "one",
									},
									SecretAccessKey: esmeta.SecretKeySelector{
										Name: "onesecret",
										// Namespace is not set
										Key: "two",
									},
								},
							},
						},
					},
				},
			},
			secrets: []v1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "onesecret",
						Namespace: "foo",
					},
					Data: map[string][]byte{
						"one": []byte("1111"),
						"two": []byte("2222"),
					},
				},
			},
			expectProvider:    true,
			expectedKeyID:     "1111",
			expectedSecretKey: "2222",
		},
		{
			name:      "error out when secret key does not exist",
			namespace: "foo",
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Auth: esv1alpha1.AWSAuth{
								SecretRef: &esv1alpha1.AWSAuthSecretRef{
									AccessKeyID: esmeta.SecretKeySelector{
										Name: "brokensecret",
										Key:  "one",
									},
									SecretAccessKey: esmeta.SecretKeySelector{
										Name: "brokensecret",
										Key:  "two",
									},
								},
							},
						},
					},
				},
			},
			secrets: []v1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "brokensecret",
						Namespace: "foo",
					},
					Data: map[string][]byte{},
				},
			},
			expectErr: "missing SecretAccessKey",
		},
		{
			name:      "should not be able to access secrets from different namespace",
			namespace: "foo",
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Auth: esv1alpha1.AWSAuth{
								SecretRef: &esv1alpha1.AWSAuthSecretRef{
									AccessKeyID: esmeta.SecretKeySelector{
										Name:      "onesecret",
										Namespace: aws.String("evil"), // this should not be possible!
										Key:       "one",
									},
									SecretAccessKey: esmeta.SecretKeySelector{
										Name:      "onesecret",
										Namespace: aws.String("evil"),
										Key:       "two",
									},
								},
							},
						},
					},
				},
			},
			secrets: []v1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "onesecret",
						Namespace: "evil",
					},
					Data: map[string][]byte{
						"one": []byte("1111"),
						"two": []byte("2222"),
					},
				},
			},
			expectErr: `secrets "onesecret" not found`,
		},
		{
			name:      "ClusterStore should use credentials from a specific namespace",
			namespace: "es-namespace",
			store: &esv1alpha1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					APIVersion: esv1alpha1.ClusterSecretStoreKindAPIVersion,
					Kind:       esv1alpha1.ClusterSecretStoreKind,
				},
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Auth: esv1alpha1.AWSAuth{
								SecretRef: &esv1alpha1.AWSAuthSecretRef{
									AccessKeyID: esmeta.SecretKeySelector{
										Name:      "onesecret",
										Namespace: aws.String("platform-team-ns"),
										Key:       "one",
									},
									SecretAccessKey: esmeta.SecretKeySelector{
										Name:      "onesecret",
										Namespace: aws.String("platform-team-ns"),
										Key:       "two",
									},
								},
							},
						},
					},
				},
			},
			secrets: []v1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "onesecret",
						Namespace: "platform-team-ns",
					},
					Data: map[string][]byte{
						"one": []byte("1111"),
						"two": []byte("2222"),
					},
				},
			},
			expectProvider:    true,
			expectedKeyID:     "1111",
			expectedSecretKey: "2222",
		},
		{
			name:      "namespace is mandatory when using ClusterStore with SecretKeySelector",
			namespace: "es-namespace",
			store: &esv1alpha1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					APIVersion: esv1alpha1.ClusterSecretStoreKindAPIVersion,
					Kind:       esv1alpha1.ClusterSecretStoreKind,
				},
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Auth: esv1alpha1.AWSAuth{
								SecretRef: &esv1alpha1.AWSAuthSecretRef{
									AccessKeyID: esmeta.SecretKeySelector{
										Name: "onesecret",
										Key:  "one",
									},
									SecretAccessKey: esmeta.SecretKeySelector{
										Name: "onesecret",
										Key:  "two",
									},
								},
							},
						},
					},
				},
			},
			expectErr: "invalid ClusterSecretStore: missing AWS AccessKeyID Namespace",
		},
		{
			name:      "jwt auth via cluster secret store",
			namespace: "es-namespace",
			sa: &v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-service-account",
					Namespace: "other-ns",
					Annotations: map[string]string{
						roleARNAnnotation: "my-sa-role",
					},
				},
			},
			jwtProvider: func(name, namespace, roleArn, region string) (credentials.Provider, error) {
				assert.Equal(t, "my-service-account", name)
				assert.Equal(t, "other-ns", namespace)
				assert.Equal(t, "my-sa-role", roleArn)
				return fakesess.CredentialsProvider{
					RetrieveFunc: func() (credentials.Value, error) {
						return credentials.Value{
							AccessKeyID:     "3333",
							SecretAccessKey: "4444",
							SessionToken:    "1234",
							ProviderName:    "fake",
						}, nil
					},
					IsExpiredFunc: func() bool { return false },
				}, nil
			},
			store: &esv1alpha1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					APIVersion: esv1alpha1.ClusterSecretStoreKindAPIVersion,
					Kind:       esv1alpha1.ClusterSecretStoreKind,
				},
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Auth: esv1alpha1.AWSAuth{
								JWTAuth: &esv1alpha1.AWSJWTAuth{
									ServiceAccountRef: &esmeta.ServiceAccountSelector{
										Name:      "my-service-account",
										Namespace: aws.String("other-ns"),
									},
								},
							},
						},
					},
				},
			},
			expectProvider:    true,
			expectedKeyID:     "3333",
			expectedSecretKey: "4444",
		},
	}
	for i := range rows {
		row := rows[i]
		t.Run(row.name, func(t *testing.T) {
			testRow(t, row)
		})
	}
}

type TestSessionRow struct {
	name              string
	store             esv1alpha1.GenericStore
	secrets           []v1.Secret
	sa                *v1.ServiceAccount
	jwtProvider       jwtProviderFactory
	namespace         string
	stsProvider       STSProvider
	expectProvider    bool
	expectErr         string
	expectedKeyID     string
	expectedSecretKey string
	env               map[string]string
}

func testRow(t *testing.T, row TestSessionRow) {
	kc := clientfake.NewClientBuilder().Build()
	for i := range row.secrets {
		err := kc.Create(context.Background(), &row.secrets[i])
		assert.Nil(t, err)
	}
	for k, v := range row.env {
		os.Setenv(k, v)
	}
	if row.sa != nil {
		err := kc.Create(context.Background(), row.sa)
		assert.Nil(t, err)
	}
	err := kc.Create(context.Background(), &authv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-service-account",
			Namespace: "other-ns",
		},
	})
	assert.Nil(t, err)
	defer func() {
		for k := range row.env {
			os.Unsetenv(k)
		}
	}()
	s, err := New(context.Background(), row.store, kc, row.namespace, row.stsProvider, row.jwtProvider)
	if !ErrorContains(err, row.expectErr) {
		t.Errorf("expected error %s but found %s", row.expectErr, err.Error())
	}
	// pass test on expected error
	if err != nil {
		return
	}
	if row.expectProvider && s == nil {
		t.Errorf("expected provider object, found nil")
		return
	}
	creds, _ := s.Config.Credentials.Get()
	assert.Equal(t, row.expectedKeyID, creds.AccessKeyID)
	assert.Equal(t, row.expectedSecretKey, creds.SecretAccessKey)
}

func TestSMEnvCredentials(t *testing.T) {
	k8sClient := clientfake.NewClientBuilder().Build()
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1111")
	os.Setenv("AWS_ACCESS_KEY_ID", "2222")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	s, err := New(context.Background(), &esv1alpha1.SecretStore{
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				// defaults
				AWS: &esv1alpha1.AWSProvider{},
			},
		},
	}, k8sClient, "example-ns", DefaultSTSProvider, nil)
	assert.Nil(t, err)
	assert.NotNil(t, s)
	creds, err := s.Config.Credentials.Get()
	assert.Nil(t, err)
	assert.Equal(t, creds.AccessKeyID, "2222")
	assert.Equal(t, creds.SecretAccessKey, "1111")
}

func TestSMAssumeRole(t *testing.T) {
	k8sClient := clientfake.NewClientBuilder().Build()
	sts := &fakesess.AssumeRoler{
		AssumeRoleFunc: func(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
			// make sure the correct role is passed in
			assert.Equal(t, *input.RoleArn, "my-awesome-role")
			return &sts.AssumeRoleOutput{
				AssumedRoleUser: &sts.AssumedRoleUser{
					Arn:           aws.String("1123132"),
					AssumedRoleId: aws.String("xxxxx"),
				},
				Credentials: &sts.Credentials{
					AccessKeyId:     aws.String("3333"),
					SecretAccessKey: aws.String("4444"),
					Expiration:      aws.Time(time.Now().Add(time.Hour)),
					SessionToken:    aws.String("6666"),
				},
			}, nil
		},
	}
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1111")
	os.Setenv("AWS_ACCESS_KEY_ID", "2222")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	s, err := New(context.Background(), &esv1alpha1.SecretStore{
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				// do assume role!
				AWS: &esv1alpha1.AWSProvider{
					Role: "my-awesome-role",
				},
			},
		},
	}, k8sClient, "example-ns", func(se *awssess.Session) stsiface.STSAPI {
		// check if the correct temporary credentials were used
		creds, err := se.Config.Credentials.Get()
		assert.Nil(t, err)
		assert.Equal(t, creds.AccessKeyID, "2222")
		assert.Equal(t, creds.SecretAccessKey, "1111")
		return sts
	}, nil)
	assert.Nil(t, err)
	assert.NotNil(t, s)

	creds, err := s.Config.Credentials.Get()
	assert.Nil(t, err)
	assert.Equal(t, creds.AccessKeyID, "3333")
	assert.Equal(t, creds.SecretAccessKey, "4444")
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
