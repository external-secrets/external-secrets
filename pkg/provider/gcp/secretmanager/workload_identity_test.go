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

package secretmanager

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type workloadIdentityTest struct {
	name           string
	expTS          bool
	expToken       *oauth2.Token
	expErr         string
	genAccessToken func(context.Context, *credentialspb.GenerateAccessTokenRequest, ...gax.CallOption) (*credentialspb.GenerateAccessTokenResponse, error)
	genIDBindToken func(ctx context.Context, client *http.Client, k8sToken, idPool, idProvider string) (*oauth2.Token, error)
	genSAToken     func(c context.Context, s1 []string, s2, s3 string) (*authv1.TokenRequest, error)
	store          esv1beta1.GenericStore
	kubeObjects    []client.Object
}

func TestWorkloadIdentity(t *testing.T) {
	clusterSANamespace := "foobar"
	tbl := []*workloadIdentityTest{
		composeTestcase(
			defaultTestCase("should skip when no workload identity is configured: TokenSource and error must be nil"),
			withStore(&esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						GCPSM: &esv1beta1.GCPSMProvider{},
					},
				},
			}),
		),
		composeTestcase(
			defaultTestCase("return access token from GenerateAccessTokenRequest with SecretStore"),
			withStore(defaultStore()),
			expTokenSource(),
			expectToken(defaultGenAccessToken),
		),
		composeTestcase(
			defaultTestCase("return idBindToken when no annotation is set with SecretStore"),
			expTokenSource(),
			expectToken(defaultIDBindToken),
			withStore(defaultStore()),
			withK8sResources([]client.Object{
				&v1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "example",
						Namespace:   "default",
						Annotations: map[string]string{},
					},
				},
			}),
		),
		composeTestcase(
			defaultTestCase("ClusterSecretStore: referent auth / service account without namespace"),
			expTokenSource(),
			withStore(
				composeStore(defaultClusterStore()),
			),
			withK8sResources([]client.Object{
				&v1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "example",
						Namespace:   "default",
						Annotations: map[string]string{},
					},
				},
			}),
		),
		composeTestcase(
			defaultTestCase("ClusterSecretStore: invalid service account"),
			expErr("foobar"),
			withStore(
				composeStore(defaultClusterStore()),
			),
			withK8sResources([]client.Object{
				&v1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "does not exist",
						Namespace:   "default",
						Annotations: map[string]string{},
					},
				},
			}),
		),
		composeTestcase(
			defaultTestCase("return access token from GenerateAccessTokenRequest with ClusterSecretStore"),
			expTokenSource(),
			expectToken(defaultGenAccessToken),
			withStore(
				composeStore(defaultClusterStore(), withSANamespace(clusterSANamespace)),
			),
			withK8sResources([]client.Object{
				&v1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: clusterSANamespace,
						Annotations: map[string]string{
							gcpSAAnnotation: "example",
						},
					},
				},
			}),
		),
	}

	for _, row := range tbl {
		t.Run(row.name, func(t *testing.T) {
			fakeIam := &fakeIAMClient{generateAccessTokenFunc: row.genAccessToken}
			fakeIDBGen := &fakeIDBindTokenGen{generateFunc: row.genIDBindToken}
			fakeSATG := &fakeSATokenGen{GenerateFunc: row.genSAToken}
			w := &workloadIdentity{
				iamClient:            fakeIam,
				idBindTokenGenerator: fakeIDBGen,
				saTokenGenerator:     fakeSATG,
			}
			cb := clientfake.NewClientBuilder()
			cb.WithObjects(row.kubeObjects...)
			client := cb.Build()
			isCluster := row.store.GetTypeMeta().Kind == esv1beta1.ClusterSecretStoreKind
			ts, err := w.TokenSource(context.Background(), row.store.GetSpec().Provider.GCPSM.Auth, isCluster, client, "default")
			// assert err
			if row.expErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err, row.expErr)
			}
			// assert ts
			if row.expTS {
				assert.NotNil(t, ts)
				if row.expToken != nil {
					tk, err := ts.Token()
					assert.NoError(t, err)
					assert.EqualValues(t, tk, row.expToken)
				}
			} else {
				assert.Nil(t, ts)
			}
		})
	}
}

func TestClusterProjectID(t *testing.T) {
	clusterID, err := clusterProjectID(defaultStore().GetSpec())
	assert.Nil(t, err)
	assert.Equal(t, clusterID, "1234")
	externalClusterID, err := clusterProjectID(defaultExternalStore().GetSpec())
	assert.Nil(t, err)
	assert.Equal(t, externalClusterID, "5678")
}

func TestSATokenGen(t *testing.T) {
	corev1 := &fakeK8sV1{}
	g := &k8sSATokenGenerator{
		corev1: corev1,
	}
	token, err := g.Generate(context.Background(), []string{"my-fake-audience"}, "bar", "default")
	assert.Nil(t, err)
	assert.Equal(t, token.Status.Token, defaultSAToken)
	assert.Equal(t, token.Spec.Audiences[0], "my-fake-audience")
}

func TestIDBTokenGen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		payload := make(map[string]string)
		rb, err := io.ReadAll(r.Body)
		assert.Nil(t, err)
		err = json.Unmarshal(rb, &payload)
		assert.Nil(t, err)
		assert.Equal(t, payload["audience"], "identitynamespace:some-idpool:some-id-provider")

		bt, err := json.Marshal(&oauth2.Token{
			AccessToken: "12345",
		})
		assert.Nil(t, err)
		rw.WriteHeader(http.StatusOK)
		rw.Write(bt)
	}))
	defer srv.Close()
	gen := &gcpIDBindTokenGenerator{
		targetURL: srv.URL,
	}
	token, err := gen.Generate(context.Background(), http.DefaultClient, "some-token", "some-idpool", "some-id-provider")
	assert.Nil(t, err)
	assert.Equal(t, token.AccessToken, "12345")
}

type testCaseMutator func(tc *workloadIdentityTest)

func composeTestcase(tc *workloadIdentityTest, mutators ...testCaseMutator) *workloadIdentityTest {
	for _, m := range mutators {
		m(tc)
	}
	return tc
}

func withStore(store esv1beta1.GenericStore) testCaseMutator {
	return func(tc *workloadIdentityTest) {
		tc.store = store
	}
}

func expTokenSource() testCaseMutator {
	return func(tc *workloadIdentityTest) {
		tc.expTS = true
	}
}

func expectToken(token string) testCaseMutator {
	return func(tc *workloadIdentityTest) {
		tc.expToken = &oauth2.Token{
			AccessToken: token,
		}
	}
}

func expErr(err string) testCaseMutator {
	return func(tc *workloadIdentityTest) {
		tc.expErr = err
	}
}

func withK8sResources(objs []client.Object) testCaseMutator {
	return func(tc *workloadIdentityTest) {
		tc.kubeObjects = objs
	}
}

var (
	defaultGenAccessToken = "default-gen-access-token"
	defaultIDBindToken    = "default-id-bind-token"
	defaultSAToken        = "default-k8s-sa-token"
)

func defaultTestCase(name string) *workloadIdentityTest {
	return &workloadIdentityTest{
		name: name,
		genAccessToken: func(c context.Context, gatr *credentialspb.GenerateAccessTokenRequest, co ...gax.CallOption) (*credentialspb.GenerateAccessTokenResponse, error) {
			return &credentialspb.GenerateAccessTokenResponse{
				AccessToken: defaultGenAccessToken,
			}, nil
		},
		genIDBindToken: func(ctx context.Context, client *http.Client, k8sToken, idPool, idProvider string) (*oauth2.Token, error) {
			return &oauth2.Token{
				AccessToken: defaultIDBindToken,
			}, nil
		},
		genSAToken: func(c context.Context, s1 []string, s2, s3 string) (*authv1.TokenRequest, error) {
			return &authv1.TokenRequest{
				Status: authv1.TokenRequestStatus{
					Token: defaultSAToken,
				},
			}, nil
		},
		kubeObjects: []client.Object{
			&v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example",
					Namespace: "default",
					Annotations: map[string]string{
						gcpSAAnnotation: "example",
					},
				},
			},
		},
	}
}

func defaultStore() *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foobar",
			Namespace: "default",
		},
		Spec: defaultStoreSpec(),
	}
}

func defaultExternalStore() *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foobar",
			Namespace: "default",
		},
		Spec: defaultExternalStoreSpec(),
	}
}

func defaultClusterStore() *esv1beta1.ClusterSecretStore {
	return &esv1beta1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: esv1beta1.ClusterSecretStoreKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "foobar",
		},
		Spec: defaultStoreSpec(),
	}
}

func defaultStoreSpec() esv1beta1.SecretStoreSpec {
	return esv1beta1.SecretStoreSpec{
		Provider: &esv1beta1.SecretStoreProvider{
			GCPSM: &esv1beta1.GCPSMProvider{
				Auth: esv1beta1.GCPSMAuth{
					WorkloadIdentity: &esv1beta1.GCPWorkloadIdentity{
						ServiceAccountRef: esmeta.ServiceAccountSelector{
							Name: "example",
						},
						ClusterLocation: "example",
						ClusterName:     "foobar",
					},
				},
				ProjectID: "1234",
			},
		},
	}
}

func defaultExternalStoreSpec() esv1beta1.SecretStoreSpec {
	return esv1beta1.SecretStoreSpec{
		Provider: &esv1beta1.SecretStoreProvider{
			GCPSM: &esv1beta1.GCPSMProvider{
				Auth: esv1beta1.GCPSMAuth{
					WorkloadIdentity: &esv1beta1.GCPWorkloadIdentity{
						ServiceAccountRef: esmeta.ServiceAccountSelector{
							Name: "example",
						},
						ClusterLocation:  "example",
						ClusterName:      "foobar",
						ClusterProjectID: "5678",
					},
				},
				ProjectID: "1234",
			},
		},
	}
}

type storeMutator func(spc esv1beta1.GenericStore)

func composeStore(store esv1beta1.GenericStore, mutators ...storeMutator) esv1beta1.GenericStore {
	for _, m := range mutators {
		m(store)
	}
	return store
}

func withSANamespace(namespace string) storeMutator {
	return func(store esv1beta1.GenericStore) {
		spc := store.GetSpec()
		spc.Provider.GCPSM.Auth.WorkloadIdentity.ServiceAccountRef.Namespace = &namespace
	}
}

// fake IDBindToken Generator.
type fakeIDBindTokenGen struct {
	generateFunc func(ctx context.Context, client *http.Client, k8sToken, idPool, idProvider string) (*oauth2.Token, error)
}

func (g *fakeIDBindTokenGen) Generate(ctx context.Context, client *http.Client, k8sToken, idPool, idProvider string) (*oauth2.Token, error) {
	return g.generateFunc(ctx, client, k8sToken, idPool, idProvider)
}

// fake IAM Client.
type fakeIAMClient struct {
	generateAccessTokenFunc func(context.Context, *credentialspb.GenerateAccessTokenRequest, ...gax.CallOption) (*credentialspb.GenerateAccessTokenResponse, error)
}

func (f *fakeIAMClient) GenerateAccessToken(ctx context.Context, req *credentialspb.GenerateAccessTokenRequest, opts ...gax.CallOption) (*credentialspb.GenerateAccessTokenResponse, error) {
	return f.generateAccessTokenFunc(ctx, req, opts...)
}

func (f *fakeIAMClient) Close() error {
	return nil
}

// fake SA Token Generator.
type fakeSATokenGen struct {
	GenerateFunc func(context.Context, []string, string, string) (*authv1.TokenRequest, error)
}

func (f *fakeSATokenGen) Generate(ctx context.Context, idPool []string, namespace, name string) (*authv1.TokenRequest, error) {
	return f.GenerateFunc(ctx, idPool, namespace, name)
}

// fake k8s client for creating tokens.
type fakeK8sV1 struct {
	k8sv1.CoreV1Interface
}

func (m *fakeK8sV1) ServiceAccounts(_ string) k8sv1.ServiceAccountInterface {
	return &fakeK8sV1SA{v1mock: m}
}

// Mock the K8s service account client.
type fakeK8sV1SA struct {
	k8sv1.ServiceAccountInterface
	v1mock *fakeK8sV1
}

func (ma *fakeK8sV1SA) CreateToken(
	_ context.Context,
	_ string,
	tokenRequest *authv1.TokenRequest,
	_ metav1.CreateOptions,
) (*authv1.TokenRequest, error) {
	tokenRequest.Status.Token = defaultSAToken
	return tokenRequest, nil
}
