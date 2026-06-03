package openbao_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/external-secrets/external-secrets/providers/v1/openbao"
	"github.com/go-viper/mapstructure/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const recordDir = "testdata/http"

var (
	requestIdReg = regexp.MustCompile(`id":"[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"`)
	timeReg      = regexp.MustCompile(`_time":"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d+Z"`)
	accessorReg  = regexp.MustCompile(`accessor":"([a-z]+)_[0-9a-f]+"`)
)

func getRecorder(t *testing.T) *recorder.Recorder {
	// this hook makes the "git diff" of a rerecord smaller by removing unneeded metadata
	cleanupHook := recorder.WithHook(func(i *cassette.Interaction) error {
		delete(i.Response.Headers, "Date")
		i.Response.Duration = 0
		i.Response.Body = requestIdReg.ReplaceAllString(i.Response.Body, `id":"00000000-0000-0000-0000-000000000000"`)
		i.Response.Body = timeReg.ReplaceAllString(i.Response.Body, `_time":"2099-09-09T09:09:09.09Z"`)
		i.Response.Body = accessorReg.ReplaceAllString(i.Response.Body, `accessor":"${1}_01234567"`)

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

func TestProvider_Validate(t *testing.T) {
	RegisterTestingT(t)

	kube, provider := setupProvider(t)

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

	result, err = client.Validate()
	Expect(client.Validate()).To(Equal(esv1.ValidationResultUnknown))
}

func setupClient(t *testing.T, v esv1.OpenBaoKVStoreVersion) esv1.SecretsClient {
	kube, provider := setupProvider(t)

	client, err := provider.NewClient(t.Context(), makeValidSecretStoreWithVersion(v), kube, "default")
	Expect(err).NotTo(HaveOccurred())
	Expect(client).NotTo(BeNil())
	return client
}

func setupProvider(t *testing.T) (client.WithWatch, *openbao.Provider) {
	r := getRecorder(t)

	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bao-token",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("root"),
		},
	}).Build()

	provider := openbao.NewProvider().(*openbao.Provider)
	provider.HTTPClient = r.GetDefaultClient()
	return kube, provider
}
