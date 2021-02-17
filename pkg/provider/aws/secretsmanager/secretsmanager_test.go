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
package secretsmanager

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	awsprovider "github.com/external-secrets/external-secrets/pkg/provider/aws"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager/fake"
)

func TestConstructor(t *testing.T) {
	rows := []ConstructorRow{
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
			expectErr: "Missing AWSSM field",
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
						AWSSM: &esv1alpha1.AWSSMProvider{},
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
			stsProvider: func(*session.Session) stscreds.AssumeRoler {
				return &fakesm.AssumeRoler{
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
						AWSSM: &esv1alpha1.AWSSMProvider{
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
						AWSSM: &esv1alpha1.AWSSMProvider{
							Auth: &esv1alpha1.AWSSMAuth{
								SecretRef: esv1alpha1.AWSSMAuthSecretRef{
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
						AWSSM: &esv1alpha1.AWSSMProvider{
							Auth: &esv1alpha1.AWSSMAuth{
								SecretRef: esv1alpha1.AWSSMAuthSecretRef{
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
						AWSSM: &esv1alpha1.AWSSMProvider{
							Auth: &esv1alpha1.AWSSMAuth{
								SecretRef: esv1alpha1.AWSSMAuthSecretRef{
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
						AWSSM: &esv1alpha1.AWSSMProvider{
							Auth: &esv1alpha1.AWSSMAuth{
								SecretRef: esv1alpha1.AWSSMAuthSecretRef{
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
						AWSSM: &esv1alpha1.AWSSMProvider{
							Auth: &esv1alpha1.AWSSMAuth{
								SecretRef: esv1alpha1.AWSSMAuthSecretRef{
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
						AWSSM: &esv1alpha1.AWSSMProvider{
							Auth: &esv1alpha1.AWSSMAuth{
								SecretRef: esv1alpha1.AWSSMAuthSecretRef{
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
			expectErr: "invalid ClusterSecretStore: missing AWSSM AccessKeyID Namespace",
		},
	}
	for i := range rows {
		row := rows[i]
		t.Run(row.name, func(t *testing.T) {
			testRow(t, row)
		})
	}
}

type ConstructorRow struct {
	name              string
	store             esv1alpha1.GenericStore
	secrets           []v1.Secret
	namespace         string
	stsProvider       awsprovider.STSProvider
	expectProvider    bool
	expectErr         string
	expectedKeyID     string
	expectedSecretKey string
	env               map[string]string
}

func testRow(t *testing.T, row ConstructorRow) {
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
	sm := SecretsManager{
		stsProvider: row.stsProvider,
	}
	newsm, err := sm.New(context.Background(), row.store, kc, row.namespace)
	if !ErrorContains(err, row.expectErr) {
		t.Errorf("expected error %s but found %s", row.expectErr, err.Error())
	}
	// pass test on expected error
	if err != nil {
		return
	}
	if row.expectProvider && newsm == nil {
		t.Errorf("expected provider object, found nil")
		return
	}
	creds, _ := newsm.(*SecretsManager).session.Config.Credentials.Get()
	assert.Equal(t, creds.AccessKeyID, row.expectedKeyID)
	assert.Equal(t, creds.SecretAccessKey, row.expectedSecretKey)
}

func TestSMEnvCredentials(t *testing.T) {
	k8sClient := clientfake.NewClientBuilder().Build()
	sm := &SecretsManager{}
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1111")
	os.Setenv("AWS_ACCESS_KEY_ID", "2222")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	smi, err := sm.New(context.Background(), &esv1alpha1.SecretStore{
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				// defaults
				AWSSM: &esv1alpha1.AWSSMProvider{},
			},
		},
	}, k8sClient, "example-ns")
	assert.Nil(t, err)
	assert.NotNil(t, smi)

	creds, err := sm.session.Config.Credentials.Get()
	assert.Nil(t, err)
	assert.Equal(t, creds.AccessKeyID, "2222")
	assert.Equal(t, creds.SecretAccessKey, "1111")
}

func TestSMAssumeRole(t *testing.T) {
	k8sClient := clientfake.NewClientBuilder().Build()
	sts := &fakesm.AssumeRoler{
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
	sm := &SecretsManager{
		stsProvider: func(se *session.Session) stscreds.AssumeRoler {
			// check if the correct temporary credentials were used
			creds, err := se.Config.Credentials.Get()
			assert.Nil(t, err)
			assert.Equal(t, creds.AccessKeyID, "2222")
			assert.Equal(t, creds.SecretAccessKey, "1111")
			return sts
		},
	}
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1111")
	os.Setenv("AWS_ACCESS_KEY_ID", "2222")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	smi, err := sm.New(context.Background(), &esv1alpha1.SecretStore{
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				// do assume role!
				AWSSM: &esv1alpha1.AWSSMProvider{
					Role: "my-awesome-role",
				},
			},
		},
	}, k8sClient, "example-ns")
	assert.Nil(t, err)
	assert.NotNil(t, smi)

	creds, err := sm.session.Config.Credentials.Get()
	assert.Nil(t, err)
	assert.Equal(t, creds.AccessKeyID, "3333")
	assert.Equal(t, creds.SecretAccessKey, "4444")
}

// test the sm<->aws interface
// make sure correct values are passed and errors are handled accordingly.
func TestGetSecret(t *testing.T) {
	fake := &fakesm.Client{}
	p := &SecretsManager{
		client: fake,
	}
	for i, row := range []struct {
		apiInput       *awssm.GetSecretValueInput
		apiOutput      *awssm.GetSecretValueOutput
		rr             esv1alpha1.ExternalSecretDataRemoteRef
		apiErr         error
		expectError    string
		expectedSecret string
	}{
		{
			// good case: default version is set
			// key is passed in, output is sent back
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/baz",
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String("RRRRR"),
			},
			apiErr:         nil,
			expectError:    "",
			expectedSecret: "RRRRR",
		},
		{
			// good case: extract property
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key:      "/baz",
				Property: "/shmoo",
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String(`{"/shmoo": "bang"}`),
			},
			apiErr:         nil,
			expectError:    "",
			expectedSecret: "bang",
		},
		{
			// bad case: missing property
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key:      "/baz",
				Property: "DOES NOT EXIST",
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String(`{"/shmoo": "bang"}`),
			},
			apiErr:         nil,
			expectError:    "has no property",
			expectedSecret: "",
		},
		{
			// bad case: extract property failure due to invalid json
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key:      "/baz",
				Property: "/shmoo",
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String(`------`),
			},
			apiErr:         nil,
			expectError:    "unable to unmarshal secret",
			expectedSecret: "",
		},
		{
			// should pass version
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/foo/bar"),
				VersionStage: aws.String("1234"),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key:     "/foo/bar",
				Version: "1234",
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String("FOOBA!"),
			},
			apiErr:         nil,
			expectError:    "",
			expectedSecret: "FOOBA!",
		},
		{
			// should return err
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/foo/bar"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/foo/bar",
			},
			apiOutput:   &awssm.GetSecretValueOutput{},
			apiErr:      fmt.Errorf("oh no"),
			expectError: "oh no",
		},
	} {
		fake.WithValue(row.apiInput, row.apiOutput, row.apiErr)
		out, err := p.GetSecret(context.Background(), row.rr)
		if !ErrorContains(err, row.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", i, err.Error(), row.expectError)
		}
		if string(out) != row.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", i, row.expectedSecret, string(out))
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	fake := &fakesm.Client{}
	p := &SecretsManager{
		client: fake,
	}
	for i, row := range []struct {
		apiInput     *awssm.GetSecretValueInput
		apiOutput    *awssm.GetSecretValueOutput
		rr           esv1alpha1.ExternalSecretDataRemoteRef
		expectedData map[string]string
		apiErr       error
		expectError  string
	}{
		{
			// good case: default version & deserialization
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String(`{"foo":"bar"}`),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/baz",
			},
			expectedData: map[string]string{
				"foo": "bar",
			},
			apiErr:      nil,
			expectError: "",
		},
		{
			// bad case: api error returned
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String(`{"foo":"bar"}`),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/baz",
			},
			expectedData: map[string]string{
				"foo": "bar",
			},
			apiErr:      fmt.Errorf("some api err"),
			expectError: "some api err",
		},
		{
			// bad case: invalid json
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String(`-----------------`),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/baz",
			},
			expectedData: map[string]string{},
			apiErr:       nil,
			expectError:  "unable to unmarshal secret",
		},
	} {
		fake.WithValue(row.apiInput, row.apiOutput, row.apiErr)
		out, err := p.GetSecretMap(context.Background(), row.rr)
		if !ErrorContains(err, row.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", i, err.Error(), row.expectError)
		}
		if cmp.Equal(out, row.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", i, row.expectedData, out)
		}
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
