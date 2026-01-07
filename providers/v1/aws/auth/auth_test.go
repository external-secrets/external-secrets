/*
Copyright © 2025 ESO Maintainer Team

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

package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/assert"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakesess "github.com/external-secrets/external-secrets/providers/v1/aws/auth/fake"
)

const (
	esNamespaceKey      = "es-namespace"
	platformTeamNsKey   = "platform-team-ns"
	myServiceAccountKey = "my-service-account"
	otherNsName         = "other-ns"
)

func TestSTSResolver(t *testing.T) {
	endpointEnvKey := STSEndpointEnv
	endpointURL := "http://sts.foo"

	t.Setenv(endpointEnvKey, endpointURL)

	f, err := customEndpointResolver{}.ResolveEndpoint(context.Background(), sts.EndpointParameters{})

	assert.Nil(t, err)
	assert.Equal(t, endpointURL, f.URI.String())
}

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
			store:     &esv1.SecretStore{},
		},
		{
			name:      "store spec has no provider",
			expectErr: "storeSpec is missing provider",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{},
			},
		},
		{
			name:      "spec has no awssm field",
			expectErr: "Missing AWS field",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{},
				},
			},
		},
		{
			name: "configure aws using environment variables",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{},
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
			stsProvider: func(_ *aws.Config) STSprovider {
				return &fakesess.AssumeRoler{
					AssumeRoleFunc: func(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
						assert.Equal(t, *input.RoleArn, "foo-bar-baz")
						return &sts.AssumeRoleOutput{
							AssumedRoleUser: &ststypes.AssumedRoleUser{
								Arn:           aws.String("1123132"),
								AssumedRoleId: aws.String("xxxxx"),
							},
							Credentials: &ststypes.Credentials{
								AccessKeyId:     aws.String("3333"),
								SecretAccessKey: aws.String("4444"),
								Expiration:      aws.Time(time.Now().Add(time.Hour)),
								SessionToken:    aws.String("6666"),
							},
						}, nil
					},
				}
			},
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{
							Auth: esv1.AWSAuth{
								SecretRef: &esv1.AWSAuthSecretRef{
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{
							Auth: esv1.AWSAuth{
								SecretRef: &esv1.AWSAuthSecretRef{
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{
							Auth: esv1.AWSAuth{
								SecretRef: &esv1.AWSAuthSecretRef{
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
			expectErr: "could not fetch SecretAccessKey secret: cannot find secret data for key: \"two\"",
		},
		{
			name:      "should not be able to access secrets from different namespace",
			namespace: "foo",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{
							Auth: esv1.AWSAuth{
								SecretRef: &esv1.AWSAuthSecretRef{
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
			namespace: esNamespaceKey,
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					APIVersion: esv1.ClusterSecretStoreKindAPIVersion,
					Kind:       esv1.ClusterSecretStoreKind,
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{
							Auth: esv1.AWSAuth{
								SecretRef: &esv1.AWSAuthSecretRef{
									AccessKeyID: esmeta.SecretKeySelector{
										Name:      "onesecret",
										Namespace: aws.String(platformTeamNsKey),
										Key:       "one",
									},
									SecretAccessKey: esmeta.SecretKeySelector{
										Name:      "onesecret",
										Namespace: aws.String(platformTeamNsKey),
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
						Namespace: platformTeamNsKey,
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
			name:      "ClusterStore should use credentials from a ExternalSecret namespace (referentAuth)",
			namespace: esNamespaceKey,
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					APIVersion: esv1.ClusterSecretStoreKindAPIVersion,
					Kind:       esv1.ClusterSecretStoreKind,
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{
							Auth: esv1.AWSAuth{
								SecretRef: &esv1.AWSAuthSecretRef{
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
			secrets: []v1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "onesecret",
						Namespace: esNamespaceKey,
					},
					Data: map[string][]byte{
						"one": []byte("7777"),
						"two": []byte("4444"),
					},
				},
			},
			expectProvider:    true,
			expectedKeyID:     "7777",
			expectedSecretKey: "4444",
		},
		{
			name:      "jwt auth via cluster secret store",
			namespace: esNamespaceKey,
			sa: &v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      myServiceAccountKey,
					Namespace: otherNsName,
					Annotations: map[string]string{
						roleARNAnnotation: "my-sa-role",
					},
				},
			},
			jwtProvider: func(name, namespace, roleArn string, _ []string, _ string) (aws.CredentialsProvider, error) {
				assert.Equal(t, myServiceAccountKey, name)
				assert.Equal(t, otherNsName, namespace)
				assert.Equal(t, "my-sa-role", roleArn)
				return fakesess.CredentialsProvider{
					RetrieveFunc: func() (aws.Credentials, error) {
						return aws.Credentials{
							AccessKeyID:     "3333",
							SecretAccessKey: "4444",
							SessionToken:    "1234",
							Source:          "fake",
						}, nil
					},
				}, nil
			},
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					APIVersion: esv1.ClusterSecretStoreKindAPIVersion,
					Kind:       esv1.ClusterSecretStoreKind,
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{
							Auth: esv1.AWSAuth{
								JWTAuth: &esv1.AWSJWTAuth{
									ServiceAccountRef: &esmeta.ServiceAccountSelector{
										Name:      myServiceAccountKey,
										Namespace: aws.String(otherNsName),
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
		{
			name: "configure aws using environment variables + assume role + check external id",
			stsProvider: func(_ *aws.Config) STSprovider {
				return &fakesess.AssumeRoler{
					AssumeRoleFunc: func(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
						assert.Equal(t, *input.ExternalId, "12345678")
						return &sts.AssumeRoleOutput{
							AssumedRoleUser: &ststypes.AssumedRoleUser{
								Arn:           aws.String("1123132"),
								AssumedRoleId: aws.String("xxxxx"),
							},
							Credentials: &ststypes.Credentials{
								AccessKeyId:     aws.String("3333"),
								SecretAccessKey: aws.String("4444"),
								Expiration:      aws.Time(time.Now().Add(time.Hour)),
								SessionToken:    aws.String("6666"),
							},
						}, nil
					},
				}
			},
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{
							Role:       "foo-bar-baz",
							ExternalID: "12345678",
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
	store             esv1.GenericStore
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
		t.Setenv(k, v)
	}
	if row.sa != nil {
		err := kc.Create(context.Background(), row.sa)
		assert.Nil(t, err)
	}
	err := kc.Create(context.Background(), &authv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      myServiceAccountKey,
			Namespace: otherNsName,
		},
	})
	assert.Nil(t, err)
	s, err := New(context.Background(), Opts{
		Store:       row.store,
		Kube:        kc,
		Namespace:   row.namespace,
		AssumeRoler: row.stsProvider,
		JWTProvider: row.jwtProvider,
	})
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
	creds, _ := s.Credentials.Retrieve(context.Background())
	assert.Equal(t, row.expectedKeyID, creds.AccessKeyID)
	assert.Equal(t, row.expectedSecretKey, creds.SecretAccessKey)
}

func TestSMEnvCredentials(t *testing.T) {
	k8sClient := clientfake.NewClientBuilder().Build()
	t.Setenv("AWS_SECRET_ACCESS_KEY", "1111")
	t.Setenv("AWS_ACCESS_KEY_ID", "2222")
	s, err := New(context.Background(), Opts{
		Kube:        k8sClient,
		Namespace:   "example-ns",
		AssumeRoler: DefaultSTSProvider,
		Store: &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					// defaults
					AWS: &esv1.AWSProvider{},
				},
			},
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, s)
	creds, err := s.Credentials.Retrieve(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, creds.AccessKeyID, "2222")
	assert.Equal(t, creds.SecretAccessKey, "1111")
}

func TestSMAssumeRole(t *testing.T) {
	k8sClient := clientfake.NewClientBuilder().Build()
	sts := &fakesess.AssumeRoler{
		AssumeRoleFunc: func(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
			if *input.RoleArn == "chained-role-1" {
				return &sts.AssumeRoleOutput{
					AssumedRoleUser: &ststypes.AssumedRoleUser{
						Arn:           aws.String("1111111"),
						AssumedRoleId: aws.String("yyyyy1"),
					},
					Credentials: &ststypes.Credentials{
						AccessKeyId:     aws.String("77771"),
						SecretAccessKey: aws.String("88881"),
						Expiration:      aws.Time(time.Now().Add(time.Hour)),
						SessionToken:    aws.String("99991"),
					},
				}, nil
			} else if *input.RoleArn == "chained-role-2" {
				return &sts.AssumeRoleOutput{
					AssumedRoleUser: &ststypes.AssumedRoleUser{
						Arn:           aws.String("2222222"),
						AssumedRoleId: aws.String("yyyyy2"),
					},
					Credentials: &ststypes.Credentials{
						AccessKeyId:     aws.String("77772"),
						SecretAccessKey: aws.String("88882"),
						Expiration:      aws.Time(time.Now().Add(time.Hour)),
						SessionToken:    aws.String("99992"),
					},
				}, nil
			}

			// make sure the correct role is passed in
			assert.Equal(t, *input.RoleArn, "my-awesome-role")
			return &sts.AssumeRoleOutput{
				AssumedRoleUser: &ststypes.AssumedRoleUser{
					Arn:           aws.String("1123132"),
					AssumedRoleId: aws.String("xxxxx"),
				},
				Credentials: &ststypes.Credentials{
					AccessKeyId:     aws.String("3333"),
					SecretAccessKey: aws.String("4444"),
					Expiration:      aws.Time(time.Now().Add(time.Hour)),
					SessionToken:    aws.String("6666"),
				},
			}, nil
		},
	}
	t.Setenv("AWS_SECRET_ACCESS_KEY", "1111")
	t.Setenv("AWS_ACCESS_KEY_ID", "2222")
	s, err := New(context.Background(), Opts{
		Kube:      k8sClient,
		Namespace: "example-ns",
		Store: &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					// do assume role!
					AWS: &esv1.AWSProvider{
						Role:            "my-awesome-role",
						AdditionalRoles: []string{"chained-role-1", "chained-role-2"},
					},
				},
			},
		},
		AssumeRoler: func(cfg *aws.Config) STSprovider {
			// check if the correct temporary credentials were used
			creds, err := cfg.Credentials.Retrieve(context.Background())
			assert.Nil(t, err)
			if creds.SessionToken == "" {
				// called with credentials from envvars
				assert.Equal(t, creds.AccessKeyID, "2222")
				assert.Equal(t, creds.SecretAccessKey, "1111")
			} else if creds.SessionToken == "99991" {
				// called with chained role 1's credentials
				assert.Equal(t, creds.AccessKeyID, "77771")
				assert.Equal(t, creds.SecretAccessKey, "88881")
			} else {
				// called with chained role 2's credentials
				assert.Equal(t, creds.AccessKeyID, "77772")
				assert.Equal(t, creds.SecretAccessKey, "88882")
			}
			return sts
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, s)

	creds, err := s.Credentials.Retrieve(context.Background())
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

func TestNewGeneratorSession_DefaultCredentialChain(t *testing.T) {
	cfg, err := NewGeneratorSession(context.Background(), esv1.AWSAuth{}, "", "us-east-1", clientfake.NewClientBuilder().Build(), "test-ns", DefaultSTSProvider, DefaultJWTProvider)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "us-east-1", cfg.Region)
}

func TestNewGeneratorSession_CredentialProviderPriority(t *testing.T) {
	ctx := context.Background()
	k8sClient := clientfake.NewClientBuilder().Build()

	assert.NoError(t, k8sClient.Create(ctx, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-creds", Namespace: "test-ns"},
		Data: map[string][]byte{
			"access-key": []byte("SECRET_KEY_ID"),
			"secret-key": []byte("SECRET_ACCESS_KEY"),
		},
	}))

	assert.NoError(t, k8sClient.Create(ctx, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-sa",
			Namespace:   "test-ns",
			Annotations: map[string]string{roleARNAnnotation: "arn:aws:iam::123456789012:role/test-role"},
		},
	}))

	jwtProviderCalled := false
	cfg, err := NewGeneratorSession(ctx, esv1.AWSAuth{
		JWTAuth: &esv1.AWSJWTAuth{
			ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "test-sa"},
		},
		SecretRef: &esv1.AWSAuthSecretRef{
			AccessKeyID:     esmeta.SecretKeySelector{Name: "aws-creds", Key: "access-key"},
			SecretAccessKey: esmeta.SecretKeySelector{Name: "aws-creds", Key: "secret-key"},
		},
	}, "", "us-east-1", k8sClient, "test-ns", DefaultSTSProvider, func(name, namespace, roleArn string, _ []string, _ string) (aws.CredentialsProvider, error) {
		jwtProviderCalled = true
		assert.Equal(t, "test-sa", name)
		assert.Equal(t, "test-ns", namespace)
		assert.Equal(t, "arn:aws:iam::123456789012:role/test-role", roleArn)
		return fakesess.CredentialsProvider{
			RetrieveFunc: func() (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     "JWT_ACCESS_KEY",
					SecretAccessKey: "JWT_SECRET_KEY",
					SessionToken:    "JWT_SESSION_TOKEN",
					Source:          "jwt",
				}, nil
			},
		}, nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.True(t, jwtProviderCalled)

	creds, err := cfg.Credentials.Retrieve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "SECRET_KEY_ID", creds.AccessKeyID)
	assert.Equal(t, "SECRET_ACCESS_KEY", creds.SecretAccessKey)
}

func TestNewGeneratorSession_OnlySecretRef(t *testing.T) {
	ctx := context.Background()
	k8sClient := clientfake.NewClientBuilder().Build()

	assert.NoError(t, k8sClient.Create(ctx, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-creds", Namespace: "test-ns"},
		Data: map[string][]byte{
			"access-key": []byte("SECRET_KEY_ID"),
			"secret-key": []byte("SECRET_ACCESS_KEY"),
		},
	}))

	cfg, err := NewGeneratorSession(ctx, esv1.AWSAuth{
		SecretRef: &esv1.AWSAuthSecretRef{
			AccessKeyID:     esmeta.SecretKeySelector{Name: "aws-creds", Key: "access-key"},
			SecretAccessKey: esmeta.SecretKeySelector{Name: "aws-creds", Key: "secret-key"},
		},
	}, "", "us-east-1", k8sClient, "test-ns", DefaultSTSProvider, DefaultJWTProvider)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	creds, err := cfg.Credentials.Retrieve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "SECRET_KEY_ID", creds.AccessKeyID)
	assert.Equal(t, "SECRET_ACCESS_KEY", creds.SecretAccessKey)
}

func TestNewGeneratorSession_RegionConfiguration(t *testing.T) {
	ctx := context.Background()
	k8sClient := clientfake.NewClientBuilder().Build()

	testCases := []struct {
		name           string
		region         string
		expectedRegion string
	}{
		{"region specified", "us-east-1", "us-east-1"},
		{"empty region uses default", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewGeneratorSession(ctx, esv1.AWSAuth{}, "", tc.region, k8sClient, "test-ns", DefaultSTSProvider, DefaultJWTProvider)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			if tc.expectedRegion != "" {
				assert.Equal(t, tc.expectedRegion, cfg.Region)
			}
		})
	}
}

func TestNewGeneratorSession_AssumeRoleWithDefaultCredentials(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "BASE_ACCESS_KEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "BASE_SECRET_KEY")

	stsProviderCalled := false
	cfg, err := NewGeneratorSession(
		context.Background(),
		esv1.AWSAuth{},
		"arn:aws:iam::123456789012:role/assumed-role",
		"us-east-1",
		clientfake.NewClientBuilder().Build(),
		"test-ns",
		func(cfg *aws.Config) STSprovider {
			stsProviderCalled = true
			creds, err := cfg.Credentials.Retrieve(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, "BASE_ACCESS_KEY", creds.AccessKeyID)
			return &fakesess.AssumeRoler{
				AssumeRoleFunc: func(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
					assert.Equal(t, "arn:aws:iam::123456789012:role/assumed-role", *input.RoleArn)
					return &sts.AssumeRoleOutput{
						AssumedRoleUser: &ststypes.AssumedRoleUser{
							Arn:           aws.String("arn:aws:sts::123456789012:assumed-role/assumed-role/session"),
							AssumedRoleId: aws.String("AROA123456"),
						},
						Credentials: &ststypes.Credentials{
							AccessKeyId:     aws.String("ASSUMED_ACCESS_KEY"),
							SecretAccessKey: aws.String("ASSUMED_SECRET_KEY"),
							SessionToken:    aws.String("ASSUMED_SESSION_TOKEN"),
							Expiration:      aws.Time(time.Now().Add(time.Hour)),
						},
					}, nil
				},
			}
		},
		DefaultJWTProvider,
	)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.True(t, stsProviderCalled)

	creds, err := cfg.Credentials.Retrieve(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "ASSUMED_ACCESS_KEY", creds.AccessKeyID)
	assert.Equal(t, "ASSUMED_SECRET_KEY", creds.SecretAccessKey)
	assert.Equal(t, "ASSUMED_SESSION_TOKEN", creds.SessionToken)
}

func TestNewGeneratorSession_DefaultCredentialChainFallback(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "ENV_ACCESS_KEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ENV_SECRET_KEY")
	t.Setenv("AWS_SESSION_TOKEN", "ENV_SESSION_TOKEN")

	cfg, err := NewGeneratorSession(context.Background(), esv1.AWSAuth{}, "", "us-east-1", clientfake.NewClientBuilder().Build(), "test-ns", DefaultSTSProvider, DefaultJWTProvider)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	creds, err := cfg.Credentials.Retrieve(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, creds.AccessKeyID)
	assert.NotEmpty(t, creds.SecretAccessKey)
}
