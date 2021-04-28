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
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	awssess "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager"
	session "github.com/external-secrets/external-secrets/pkg/provider/aws/session"
	fakesess "github.com/external-secrets/external-secrets/pkg/provider/aws/session/fake"
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
							Auth: &esv1alpha1.AWSAuth{
								SecretRef: esv1alpha1.AWSAuthSecretRef{
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

			stsProvider: func(*awssess.Session) stscreds.AssumeRoler {
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
							Auth: &esv1alpha1.AWSAuth{
								SecretRef: esv1alpha1.AWSAuthSecretRef{
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
							Auth: &esv1alpha1.AWSAuth{
								SecretRef: esv1alpha1.AWSAuthSecretRef{
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
							Auth: &esv1alpha1.AWSAuth{
								SecretRef: esv1alpha1.AWSAuthSecretRef{
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
							Auth: &esv1alpha1.AWSAuth{
								SecretRef: esv1alpha1.AWSAuthSecretRef{
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
							Auth: &esv1alpha1.AWSAuth{
								SecretRef: esv1alpha1.AWSAuthSecretRef{
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
							Auth: &esv1alpha1.AWSAuth{
								SecretRef: esv1alpha1.AWSAuthSecretRef{
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
	namespace         string
	stsProvider       session.STSProvider
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
	defer func() {
		for k := range row.env {
			os.Unsetenv(k)
		}
	}()
	s, err := newSession(context.Background(), row.store, kc, row.namespace, row.stsProvider)
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
	assert.Equal(t, creds.AccessKeyID, row.expectedKeyID)
	assert.Equal(t, creds.SecretAccessKey, row.expectedSecretKey)
}

func TestSMEnvCredentials(t *testing.T) {
	k8sClient := clientfake.NewClientBuilder().Build()
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1111")
	os.Setenv("AWS_ACCESS_KEY_ID", "2222")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	s, err := newSession(context.Background(), &esv1alpha1.SecretStore{
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				// defaults
				AWS: &esv1alpha1.AWSProvider{},
			},
		},
	}, k8sClient, "example-ns", session.DefaultSTSProvider)
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
	s, err := newSession(context.Background(), &esv1alpha1.SecretStore{
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				// do assume role!
				AWS: &esv1alpha1.AWSProvider{
					Role: "my-awesome-role",
				},
			},
		},
	}, k8sClient, "example-ns", func(se *awssess.Session) stscreds.AssumeRoler {
		// check if the correct temporary credentials were used
		creds, err := se.Config.Credentials.Get()
		assert.Nil(t, err)
		assert.Equal(t, creds.AccessKeyID, "2222")
		assert.Equal(t, creds.SecretAccessKey, "1111")
		return sts
	})
	assert.Nil(t, err)
	assert.NotNil(t, s)

	creds, err := s.Config.Credentials.Get()
	assert.Nil(t, err)
	assert.Equal(t, creds.AccessKeyID, "3333")
	assert.Equal(t, creds.SecretAccessKey, "4444")
}

func TestResolver(t *testing.T) {
	tbl := []struct {
		env     string
		service string
		url     string
	}{
		{
			env:     SecretsManagerEndpointEnv,
			service: "secretsmanager",
			url:     "http://sm.foo",
		},
		{
			env:     SSMEndpointEnv,
			service: "ssm",
			url:     "http://ssm.foo",
		},
		{
			env:     STSEndpointEnv,
			service: "sts",
			url:     "http://sts.foo",
		},
	}

	for _, item := range tbl {
		os.Setenv(item.env, item.url)
		defer os.Unsetenv(item.env)
	}

	f := ResolveEndpoint()

	for _, item := range tbl {
		ep, err := f.EndpointFor(item.service, "")
		assert.Nil(t, err)
		assert.Equal(t, item.url, ep.URL)
	}
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
