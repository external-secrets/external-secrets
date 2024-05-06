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

package fake

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	testingfake "github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
)

func TestNewClient(t *testing.T) {
	p := &Provider{}
	gomega.RegisterTestingT(t)

	// nil store
	_, err := p.NewClient(context.Background(), nil, nil, "")
	gomega.Expect(err).To(gomega.HaveOccurred())

	// missing provider
	_, err = p.NewClient(context.Background(), &esv1beta1.SecretStore{}, nil, "")
	gomega.Expect(err).To(gomega.HaveOccurred())
}

func TestValidateStore(t *testing.T) {
	p := &Provider{}
	gomega.RegisterTestingT(t)
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Fake: &esv1beta1.FakeProvider{
					Data: []esv1beta1.FakeProviderData{},
				},
			},
		},
	}
	// empty data must not error
	_, err := p.ValidateStore(store)
	gomega.Expect(err).To(gomega.BeNil())
	// missing key in data
	data := esv1beta1.FakeProviderData{}
	data.Version = "v1"
	store.Spec.Provider.Fake.Data = []esv1beta1.FakeProviderData{data}
	_, err = p.ValidateStore(store)
	gomega.Expect(err).To(gomega.BeEquivalentTo(fmt.Errorf(errMissingKeyField, 0)))
	// missing values in data
	data.Key = "/foo"
	store.Spec.Provider.Fake.Data = []esv1beta1.FakeProviderData{data}
	_, err = p.ValidateStore(store)
	gomega.Expect(err).To(gomega.BeEquivalentTo(fmt.Errorf(errMissingValueField, 0)))
	// spec ok
	data.Value = "bar"
	data.ValueMap = map[string]string{"foo": "bar"}
	store.Spec.Provider.Fake.Data = []esv1beta1.FakeProviderData{data}
	_, err = p.ValidateStore(store)
	gomega.Expect(err).To(gomega.BeNil())
}
func TestClose(t *testing.T) {
	p := &Provider{}
	gomega.RegisterTestingT(t)
	err := p.Close(context.TODO())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

type testCase struct {
	name     string
	input    []esv1beta1.FakeProviderData
	request  esv1beta1.ExternalSecretDataRemoteRef
	expValue string
	expErr   string
}

func TestGetAllSecrets(t *testing.T) {
	cases := []struct {
		desc        string
		data        []esv1beta1.FakeProviderData
		ref         esv1beta1.ExternalSecretFind
		expected    map[string][]byte
		expectedErr string
	}{
		{
			desc: "no matches",
			data: []esv1beta1.FakeProviderData{},
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: "some-key",
				},
			},
			expected: map[string][]byte{},
		},
		{
			desc: "matches",
			data: []esv1beta1.FakeProviderData{
				{
					Key:   "some-key1",
					Value: "some-value1",
				},
				{
					Key:   "some-key2",
					Value: "some-value2",
				},
				{
					Key:   "another-key1",
					Value: "another-value1",
				},
			},
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: "some-key.*",
				},
			},
			expected: map[string][]byte{
				"some-key1": []byte("some-value1"),
				"some-key2": []byte("some-value2"),
			},
		},
		{
			desc: "matches with version",
			data: []esv1beta1.FakeProviderData{
				{
					Key:     "some-key1",
					Value:   "some-value1-version1",
					Version: "1",
				},
				{
					Key:     "some-key1",
					Value:   "some-value1-version2",
					Version: "2",
				},
				{
					Key:     "some-key2",
					Value:   "some-value2-version1",
					Version: "1",
				},
				{
					Key:     "some-key2",
					Value:   "some-value2-version2",
					Version: "2",
				},
				{
					Key:     "some-key2",
					Value:   "some-value2-version3",
					Version: "3",
				},
				{
					Key:     "another-key1",
					Value:   "another-value1-version1",
					Version: "1",
				},
				{
					Key:     "another-key1",
					Value:   "another-value1-version2",
					Version: "2",
				},
			},
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: "some-key.*",
				},
			},
			expected: map[string][]byte{
				"some-key1": []byte("some-value1-version2"),
				"some-key2": []byte("some-value2-version3"),
			},
		},
		{
			desc: "unsupported operator",
			data: []esv1beta1.FakeProviderData{},
			ref: esv1beta1.ExternalSecretFind{
				Path: ptr.To("some-path"),
			},
			expectedErr: "unsupported find operator",
		},
	}

	for i, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			p := Provider{}

			client, err := p.NewClient(ctx, &esv1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("secret-store-%v", i),
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Fake: &esv1beta1.FakeProvider{
							Data: tc.data,
						},
					},
				},
			}, nil, "")
			if err != nil {
				t.Fatalf("failed to create a client: %v", err)
			}

			got, err := client.GetAllSecrets(ctx, tc.ref)
			if err != nil {
				if tc.expectedErr == "" {
					t.Fatalf("failed to call GetAllSecrets: %v", err)
				}

				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("%q expected to contain substring %q", err.Error(), tc.expectedErr)
				}

				return
			}

			if tc.expectedErr != "" {
				t.Fatal("expected to receive an error but got nil")
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("(-got, +want)\n%s", diff)
			}
		})
	}
}

func TestGetSecret(t *testing.T) {
	gomega.RegisterTestingT(t)
	p := &Provider{}
	tbl := []testCase{
		{
			name:  "return err when not found",
			input: []esv1beta1.FakeProviderData{},
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "/foo",
				Version: "v2",
			},
			expErr: esv1beta1.NoSecretErr.Error(),
		},
		{
			name: "get correct value from multiple versions",
			input: []esv1beta1.FakeProviderData{
				{
					Key:     "/foo",
					Value:   "bar2",
					Version: "v2",
				},
				{
					Key:   "junk",
					Value: "xxxxx",
				},
				{
					Key:     "/foo",
					Value:   "bar1",
					Version: "v1",
				},
			},
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "/foo",
				Version: "v2",
			},
			expValue: "bar2",
		},
		{
			name: "get correct value from multiple properties",
			input: []esv1beta1.FakeProviderData{
				{
					Key:   "junk",
					Value: "xxxxx",
				},
				{
					Key:   "/foo",
					Value: `{"p1":"bar","p2":"bar2"}`,
				},
			},
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "/foo",
				Property: "p2",
			},
			expValue: "bar2",
		},
	}

	for i, row := range tbl {
		t.Run(row.name, func(t *testing.T) {
			cl, err := p.NewClient(context.Background(), &esv1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("secret-store-%v", i),
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Fake: &esv1beta1.FakeProvider{
							Data: row.input,
						},
					},
				},
			}, nil, "")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			out, err := cl.GetSecret(context.Background(), row.request)
			if row.expErr != "" {
				gomega.Expect(err).To(gomega.MatchError(row.expErr))
			} else {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
			gomega.Expect(string(out)).To(gomega.Equal(row.expValue))
		})
	}
}

type setSecretTestCase struct {
	name       string
	input      []esv1beta1.FakeProviderData
	requestKey string
	expValue   string
	expErr     string
}

func TestSetSecret(t *testing.T) {
	gomega.RegisterTestingT(t)
	p := &Provider{}
	secretKey := "secret-key"
	tbl := []setSecretTestCase{
		{
			name:       "return nil if no existing secret",
			input:      []esv1beta1.FakeProviderData{},
			requestKey: "/foo",
			expValue:   "my-secret-value",
		},
		{
			name: "return err if existing secret",
			input: []esv1beta1.FakeProviderData{
				{
					Key:   "/foo",
					Value: "bar2",
				},
			},
			requestKey: "/foo",
			expErr:     errors.New("key already exists").Error(),
		},
	}

	for i, row := range tbl {
		t.Run(row.name, func(t *testing.T) {
			cl, err := p.NewClient(context.Background(), &esv1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("secret-store-%v", i),
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Fake: &esv1beta1.FakeProvider{
							Data: row.input,
						},
					},
				},
			}, nil, "")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			secret := &corev1.Secret{
				Data: map[string][]byte{
					secretKey: []byte(row.expValue),
				},
			}
			err = cl.PushSecret(context.TODO(), secret, testingfake.PushSecretData{
				SecretKey: secretKey,
				RemoteKey: row.requestKey,
			})
			if row.expErr != "" {
				gomega.Expect(err).To(gomega.MatchError(row.expErr))
			} else {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				out, err := cl.GetSecret(context.Background(), esv1beta1.ExternalSecretDataRemoteRef{
					Key: row.requestKey,
				})
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(string(out)).To(gomega.Equal(row.expValue))
			}
		})
	}
}

type secretExistsTestCase struct {
	name      string
	input     []esv1beta1.FakeProviderData
	request   esv1alpha1.PushSecretRemoteRef
	expExists bool
}

func TestSecretExists(t *testing.T) {
	gomega.RegisterTestingT(t)
	p := &Provider{}
	tbl := []secretExistsTestCase{
		{
			name:  "return false, nil if no existing secret",
			input: []esv1beta1.FakeProviderData{},
			request: esv1alpha1.PushSecretRemoteRef{
				RemoteKey: "/foo",
			},
			expExists: false,
		},
		{
			name: "return true, nil if existing secret",
			input: []esv1beta1.FakeProviderData{
				{
					Key:   "/foo",
					Value: "bar",
				},
			},
			request: esv1alpha1.PushSecretRemoteRef{
				RemoteKey: "/foo",
			},
			expExists: true,
		},
	}

	for i, row := range tbl {
		t.Run(row.name, func(t *testing.T) {
			cl, err := p.NewClient(context.Background(), &esv1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("secret-store-%v", i),
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Fake: &esv1beta1.FakeProvider{
							Data: row.input,
						},
					},
				},
			}, nil, "")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			exists, err := cl.SecretExists(context.TODO(), row.request)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(exists).To(gomega.Equal(row.expExists))
		})
	}
}

type testMapCase struct {
	name     string
	input    []esv1beta1.FakeProviderData
	request  esv1beta1.ExternalSecretDataRemoteRef
	expValue map[string][]byte
	expErr   string
}

func TestGetSecretMap(t *testing.T) {
	gomega.RegisterTestingT(t)
	p := &Provider{}
	tbl := []testMapCase{
		{
			name:  "return err when not found",
			input: []esv1beta1.FakeProviderData{},
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "/foo",
				Version: "v2",
			},
			expErr: esv1beta1.NoSecretErr.Error(),
		},
		{
			name: "get correct map from multiple versions by using Value only",
			input: []esv1beta1.FakeProviderData{
				{
					Key:     "/bar",
					Version: "v1",
					Value:   `{"john":"doe"}`,
				},
			},
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "/bar",
				Version: "v1",
			},
			expValue: map[string][]byte{
				"john": []byte("doe"),
			},
		},
		{
			name: "get correct maps from multiple versions by using Value only",
			input: []esv1beta1.FakeProviderData{
				{
					Key:     "/bar",
					Version: "v3",
					Value:   `{"john":"doe", "foo": "bar"}`,
				},
			},
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "/bar",
				Version: "v3",
			},
			expValue: map[string][]byte{
				"john": []byte("doe"),
				"foo":  []byte("bar"),
			},
		},
		{
			name: "invalid marshal",
			input: []esv1beta1.FakeProviderData{
				{
					Key:     "/bar",
					Version: "v3",
					Value:   `---------`,
				},
			},
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "/bar",
				Version: "v3",
			},
			expErr: "unable to unmarshal secret: invalid character '-' in numeric literal",
		},
		{
			name: "get correct value from ValueMap due to retrocompatibility",
			input: []esv1beta1.FakeProviderData{
				{
					Key:     "/foo/bar",
					Version: "v3",
					ValueMap: map[string]string{
						"john": "doe",
						"baz":  "bang",
					},
				},
			},
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "/foo/bar",
				Version: "v3",
			},
			expValue: map[string][]byte{
				"john": []byte("doe"),
				"baz":  []byte("bang"),
			},
		},
		{
			name: "get correct value from multiple versions",
			input: []esv1beta1.FakeProviderData{
				{
					Key:     "john",
					Value:   "doe",
					Version: "v2",
				},
				{
					Key: "junk",
					ValueMap: map[string]string{
						"junk": "ok",
					},
				},
				{
					Key: "/foo",
					ValueMap: map[string]string{
						"foo": "bar",
						"baz": "bang",
					},
					Version: "v1",
				},
				{
					Key: "/foo",
					ValueMap: map[string]string{
						"foo": "bar",
						"baz": "bang",
					},
					Version: "v2",
				},
			},
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "/foo",
				Version: "v2",
			},
			expValue: map[string][]byte{
				"foo": []byte("bar"),
				"baz": []byte("bang"),
			},
		},
	}

	for i, row := range tbl {
		t.Run(row.name, func(t *testing.T) {
			cl, err := p.NewClient(context.Background(), &esv1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("secret-store-%v", i),
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Fake: &esv1beta1.FakeProvider{
							Data: row.input,
						},
					},
				},
			}, nil, "")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			out, err := cl.GetSecretMap(context.Background(), row.request)
			if row.expErr != "" {
				gomega.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(row.expErr)))
			} else {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
			gomega.Expect(out).To(gomega.Equal(row.expValue))
		})
	}
}
