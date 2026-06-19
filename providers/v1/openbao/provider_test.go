/*
Copyright © The ESO Authors

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

package openbao_test

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/openbao"

	. "github.com/onsi/gomega"
)

const recordDir = "testdata/http"
const fakeToken = "s.fakeTOKEN123"

var (
	requestIdReg     = regexp.MustCompile(`id":"[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"`)
	timeReg          = regexp.MustCompile(`_time":"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d+Z"`)
	namedAccessorReg = regexp.MustCompile(`accessor":"([a-z]+)_[0-9a-f]+"`)
	accessorReg      = regexp.MustCompile(`accessor":"([A-Za-z0-9]+)"`)
	tokenReg         = regexp.MustCompile(`client_token":"s\.([A-Za-z0-9]+)"`)
)

func getRecorder(t *testing.T) *recorder.Recorder {
	// this hook makes the "git diff" of a rerecord smaller by removing unneeded metadata
	cleanupHook := recorder.WithHook(func(i *cassette.Interaction) error {
		delete(i.Response.Headers, "Date")
		i.Response.Duration = 0
		i.Response.Body = requestIdReg.ReplaceAllString(i.Response.Body, `id":"00000000-0000-0000-0000-000000000000"`)
		i.Response.Body = timeReg.ReplaceAllString(i.Response.Body, `_time":"2099-09-09T09:09:09.09Z"`)
		i.Response.Body = namedAccessorReg.ReplaceAllString(i.Response.Body, `accessor":"${1}_01234567"`)
		i.Response.Body = accessorReg.ReplaceAllString(i.Response.Body, `accessor":"AbCdEfGHiJk123"`)
		i.Response.Body = tokenReg.ReplaceAllString(i.Response.Body, `client_token":"`+fakeToken+`"`)

		token := i.Request.Headers.Get("X-Vault-Token")
		if token != "" && token != "root" {
			i.Request.Headers.Set("X-Vault-Token", fakeToken)
		}

		var body map[string]any
		err := json.Unmarshal([]byte(i.Response.Body), &body)
		if err != nil {
			return err
		}
		indentedBody, err := json.MarshalIndent(body, "", "  ")
		if err != nil {
			return err
		}
		i.Response.Body = string(indentedBody)

		return nil
	}, recorder.BeforeSaveHook)

	r, err := recorder.New(filepath.Join(recordDir, strings.ReplaceAll(t.Name(), "/", "_")), cleanupHook)
	Expect(err).NotTo(HaveOccurred())

	t.Cleanup(func() {
		if err := r.Stop(); err != nil {
			t.Log(err)
		}
	})

	return r
}

func makeValidSecretStoreWithVersion(v esv1.OpenBaoKVStoreVersion) *esv1.SecretStore {
	path := "secret"
	if v == esv1.OpenBaoKVStoreV1 {
		path = "secret_v1"
	}
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openbao-store",
			Namespace: "default",
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				OpenBao: &esv1.OpenBaoProvider{
					Server:  "http://localhost:8200",
					Path:    &path,
					Version: v,
					Auth: &esv1.OpenBaoAuth{
						TokenSecretRef: &esmeta.SecretKeySelector{
							Name: "bao-token",
							Key:  "token",
						},
					},
				},
			},
		},
	}
}

func TestMain(m *testing.M) {
	record := os.Getenv("ESO_PROVIDER_OPENBAO_RERECORD") == "true"
	if record {
		bao := exec.Command("bao", "server", "-dev", "-dev-root-token-id=root")
		bao.Stderr = os.Stderr
		bao.Stdout = os.Stdin
		err := bao.Start()
		if err != nil {
			panic(err)
		}
		defer func() {
			err := bao.Process.Signal(os.Interrupt)
			if err != nil {
				panic(err)
			}

			err = bao.Wait()
			if err != nil {
				panic(err)
			}
		}()

		err = os.RemoveAll(recordDir)
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Second)

		initBao := exec.Command("./testdata/init-bao.sh")
		initBao.Stderr = os.Stderr
		initBao.Stdout = os.Stdin
		err = initBao.Run()
		if err != nil {
			panic(err)
		}

		fmt.Println("started bao, running test")
	}

	m.Run()
}

func TestProvider_NewClient_TokenNotFound(t *testing.T) {
	RegisterTestingT(t)

	kube := clientfake.NewClientBuilder().Build()

	provider := openbao.NewProvider()
	client, err := provider.NewClient(t.Context(), makeValidSecretStoreWithVersion(esv1.OpenBaoKVStoreV2), kube, "default")

	Expect(err).To(MatchError(ContainSubstring(`secrets "bao-token" not found`)))
	Expect(client).To(BeNil())
}

func TestProvider_KVv2(t *testing.T) {
	v := esv1.OpenBaoKVStoreV2

	t.Run("GetSecret_Property", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		data, err := client.GetSecret(t.Context(), esv1.ExternalSecretDataRemoteRef{
			Key:      "foo",
			Property: "bar",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(BeEquivalentTo("bazz"))

		data, err = client.GetSecret(t.Context(), esv1.ExternalSecretDataRemoteRef{
			Key:      "foo",
			Property: "does-not-exist",
		})
		Expect(err).To(MatchError(`cannot find secret data for key: "does-not-exist"`))
		Expect(data).To(BeNil())
	})

	t.Run("GetSecret_Versioned", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		data, err := client.GetSecret(t.Context(), esv1.ExternalSecretDataRemoteRef{
			Key:      "foo",
			Property: "bar",
			Version:  "1",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(BeEquivalentTo("old_bazz"))

		data, err = client.GetSecret(t.Context(), esv1.ExternalSecretDataRemoteRef{
			Key:      "foo",
			Property: "bar",
			Version:  "invalid",
		})
		Expect(err).To(MatchError(`invalid Ref.Version: strconv.Atoi: parsing "invalid": invalid syntax`))
		Expect(data).To(BeNil())
	})

	t.Run("GetSecret_Full", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		data, err := client.GetSecret(t.Context(), esv1.ExternalSecretDataRemoteRef{
			Key: "foo",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(MatchJSON(`{
			"bar": "bazz",
			"lorem": "ipsum"
		}`))
	})

	t.Run("GetSecretMap", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		data, err := client.GetSecretMap(t.Context(), esv1.ExternalSecretDataRemoteRef{
			Key: "foo",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(Equal(map[string][]byte{
			"bar":   []byte("bazz"),
			"lorem": []byte("ipsum"),
		}))
	})

	t.Run("GetAllSecrets", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		allData, err := client.GetAllSecrets(t.Context(), esv1.ExternalSecretFind{
			Name: &esv1.FindName{
				RegExp: "fo+",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(allData).To(HaveLen(1))
		Expect(allData).To(HaveKeyWithValue("foo", MatchJSON(`{
			"bar": "bazz",
			"lorem": "ipsum"
		}`)))
	})

	t.Run("GetAllSecrets_NoMatch", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		allData, err := client.GetAllSecrets(t.Context(), esv1.ExternalSecretFind{
			Path: new("empty"),
			Name: &esv1.FindName{
				RegExp: "nomatch",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(allData).To(HaveLen(0))
	})
}

func TestProvider_KVv1(t *testing.T) {
	v := esv1.OpenBaoKVStoreV1

	t.Run("GetSecret_Property", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)
		data, err := client.GetSecret(t.Context(), esv1.ExternalSecretDataRemoteRef{
			Key:      "foo",
			Property: "bar",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(BeEquivalentTo("bazz_v1"))
	})

	t.Run("GetSecret_Versioned", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		data, err := client.GetSecret(t.Context(), esv1.ExternalSecretDataRemoteRef{
			Key:      "foo",
			Property: "bar",
			Version:  "1",
		})
		Expect(err).To(MatchError("OpenBao KVv1 secrets do not support versioning (use KVv2)"))
		Expect(data).To(BeNil())
	})

	t.Run("GetSecret_Full", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		data, err := client.GetSecret(t.Context(), esv1.ExternalSecretDataRemoteRef{
			Key: "foo",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(MatchJSON(`{
			"bar": "bazz_v1",
			"lorem": "ipsum_v1"
		}`))
	})

	t.Run("GetAllSecrets", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		allData, err := client.GetAllSecrets(t.Context(), esv1.ExternalSecretFind{
			Name: &esv1.FindName{
				RegExp: "fo+",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(allData).To(HaveLen(1))
		Expect(allData).To(HaveKeyWithValue("foo", MatchJSON(`{
			"bar": "bazz_v1",
			"lorem": "ipsum_v1"
		}`)))
	})

	t.Run("GetAllSecrets_NoMatch", func(t *testing.T) {
		RegisterTestingT(t)
		client := setupClient(t, v)

		allData, err := client.GetAllSecrets(t.Context(), esv1.ExternalSecretFind{
			Path: new("empty"),
			Name: &esv1.FindName{
				RegExp: "nomatch",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(allData).To(HaveLen(0))
	})
}

func TestProvider_Auth_UserPass(t *testing.T) {
	RegisterTestingT(t)
	kube, provider := setupProvider(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "password-of-alice",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"pw": []byte("bob4ever"),
		},
	})

	store := makeValidSecretStoreWithVersion(esv1.OpenBaoKVStoreV2)
	store.Spec.Provider.OpenBao.Auth = &esv1.OpenBaoAuth{
		UserPass: &esv1.OpenBaoUserPassAuth{
			Path:     "customuserpasspath",
			Username: "alice",
			SecretRef: esmeta.SecretKeySelector{
				Name: "password-of-alice",
				Key:  "pw",
			},
		},
	}

	client, err := provider.NewClient(t.Context(), store, kube, "default")
	Expect(err).NotTo(HaveOccurred())
	Expect(client).NotTo(BeNil())
	t.Cleanup(func() {
		client.Close(t.Context())
	})

	data, err := client.GetSecret(t.Context(), esv1.ExternalSecretDataRemoteRef{
		Key:      "foo",
		Property: "bar",
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(data).To(BeEquivalentTo("bazz"))
}

func TestProvider_Validate(t *testing.T) {
	RegisterTestingT(t)

	kube, provider := setupProvider(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bao-token",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("root"),
		},
	})

	store := makeValidSecretStoreWithVersion(esv1.OpenBaoKVStoreV1)
	client, err := provider.NewClient(t.Context(), store, kube, "default")
	Expect(err).NotTo(HaveOccurred())
	Expect(client).NotTo(BeNil())

	Expect(client.Validate()).To(Equal(esv1.ValidationResultReady))

	// make version it invalid
	store.Spec.Provider.OpenBao.Version = esv1.OpenBaoKVStoreV2
	client, err = provider.NewClient(t.Context(), store, kube, "default")
	Expect(err).NotTo(HaveOccurred())
	Expect(client).NotTo(BeNil())

	result, err := client.Validate()
	Expect(err).To(MatchError("expected kv engine version 2 found version 1"))
	Expect(result).To(Equal(esv1.ValidationResultError))

	// make engine it invalid
	store.Spec.Provider.OpenBao.Path = new("sys")
	client, err = provider.NewClient(t.Context(), store, kube, "default")
	Expect(err).NotTo(HaveOccurred())
	Expect(client).NotTo(BeNil())

	result, err = client.Validate()
	Expect(err).To(MatchError(`expected mount type "kv" found "system"`))
	Expect(result).To(Equal(esv1.ValidationResultError))

	// make it a cluster store
	clusterStore := &esv1.ClusterSecretStore{}
	Expect(mapstructure.Decode(store, clusterStore)).NotTo(HaveOccurred())
	client, err = provider.NewClient(t.Context(), clusterStore, kube, "")
	Expect(err).NotTo(HaveOccurred())
	Expect(client).NotTo(BeNil())

	Expect(client.Validate()).To(Equal(esv1.ValidationResultUnknown))
}

var dummyCA = []byte(`-----BEGIN CERTIFICATE-----
MIIBgDCCATKgAwIBAgIRAOzjpCdp42oW5MoccLpRXpAwBQYDK2VwMBIxEDAOBgNV
BAMTB3Jvb3QtY2EwHhcNMjIwMjA5MTAyNTMxWhcNMzIwMjA3MTAyNTMxWjAaMRgw
FgYDVQQDEw9pbnRlcm1lZGlhdGUtY2EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNC
AATekdyX6cZe0Ajmme363TQoWnrQwXnARzeWEf4FRQE8BGWgf8z7wljjpb4M4S4f
+CJAYYY/6x38UnlsxXEeBTofo2YwZDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/
BAgwBgEB/wIBADAdBgNVHQ4EFgQUIuDzQn9tkFs535jz5X3iXnEzbMQwHwYDVR0j
BBgwFoAUa2fUac2OZ3pzE6EydVq7UvwiQa0wBQYDK2VwA0EA4gntaGs/3ME6q1y9
gO4ntri2qwoC25l3q7q9BiFBmeBmvS6I1w9HCZHtB3JnVC/IYDTCYDNTbpGWEOjl
aCKLCA==
-----END CERTIFICATE-----`)

func TestProvider_CustomCA(t *testing.T) {
	cases := []struct {
		name          string
		spec          esv1.OpenBaoProvider
		k8sObjects    []client.Object
		expectedError string
	}{
		{
			name: "CABundle",
			spec: esv1.OpenBaoProvider{
				CABundle: dummyCA,
			},
		},
		{
			name: "CABundle_invalid",
			spec: esv1.OpenBaoProvider{
				CABundle: []byte("invalid"),
			},
			expectedError: "cannot set OpenBao CA certificate: failed to decode ca bundle: failed to parse the new certificate, not valid pem data",
		},
		{
			name: "CAProvider",
			spec: esv1.OpenBaoProvider{
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeSecret,
					Name: "dummy-ca",
					Key:  "ca.pem",
				},
			},
			k8sObjects: []client.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dummy-ca",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"ca.pem": dummyCA,
				},
			}},
		},
		{
			name: "CAProvider_not_found",
			spec: esv1.OpenBaoProvider{
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeSecret,
					Name: "dummy-ca",
					Key:  "ca.pem",
				},
			},
			expectedError: `cannot set OpenBao CA certificate: failed to get cert from secret: failed to resolve secret key ref: cannot get Kubernetes secret "dummy-ca" from namespace "default": secrets "dummy-ca" not found`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			RegisterTestingT(t)

			kube := clientfake.NewClientBuilder().WithObjects(tc.k8sObjects...).Build()
			provider := openbao.NewProvider().(*openbao.Provider)

			originalFactory := provider.HTTPClientFactory
			var httpClient atomic.Pointer[http.Client]
			var factoryCallCount atomic.Int64
			provider.HTTPClientFactory = func() *http.Client {
				c := originalFactory()
				httpClient.Store(c)
				factoryCallCount.Add(1)
				return c
			}

			store := makeValidSecretStoreWithVersion(esv1.OpenBaoKVStoreV2)
			store.Spec.Provider.OpenBao = &tc.spec

			client, err := provider.NewClient(t.Context(), store, kube, "default")

			if tc.expectedError != "" {
				Expect(err).To(MatchError(tc.expectedError))
				Expect(client).To(BeNil())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())

				expectedPool := x509.NewCertPool()
				Expect(expectedPool.AppendCertsFromPEM(dummyCA)).To(BeTrue())

				Expect(factoryCallCount.Load()).To(BeEquivalentTo(1))
				transport := httpClient.Load().Transport
				Expect(transport).To(BeAssignableToTypeOf(&http.Transport{}))
				tls := transport.(*http.Transport).TLSClientConfig
				Expect(tls.RootCAs.Equal(expectedPool)).To(BeTrueBecause("root CAs should equal expected cert pool"))

				client.Close(t.Context())
			}
		})
	}
}

func setupClient(t *testing.T, v esv1.OpenBaoKVStoreVersion) esv1.SecretsClient {
	kube, provider := setupProvider(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bao-token",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("root"),
		},
	})

	client, err := provider.NewClient(t.Context(), makeValidSecretStoreWithVersion(v), kube, "default")
	Expect(err).NotTo(HaveOccurred())
	Expect(client).NotTo(BeNil())
	t.Cleanup(func() {
		client.Close(t.Context())
	})
	return client
}

func setupProvider(t *testing.T, objects ...client.Object) (client.WithWatch, *openbao.Provider) {
	r := getRecorder(t)

	kube := clientfake.NewClientBuilder().WithObjects(objects...).Build()

	provider := openbao.NewProvider().(*openbao.Provider)
	provider.HTTPClientFactory = r.GetDefaultClient
	return kube, provider
}
