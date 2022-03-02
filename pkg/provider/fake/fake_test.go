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
	"testing"

	"github.com/onsi/gomega"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
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
			expErr: "secret value not found",
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
	}

	for _, row := range tbl {
		t.Run(row.name, func(t *testing.T) {
			cl, err := p.NewClient(context.Background(), &esv1beta1.SecretStore{
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
			expErr: "secret value not found",
		},
		{
			name: "get correct value from multiple versions",
			input: []esv1beta1.FakeProviderData{
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

	for _, row := range tbl {
		t.Run(row.name, func(t *testing.T) {
			cl, err := p.NewClient(context.Background(), &esv1beta1.SecretStore{
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
				gomega.Expect(err).To(gomega.MatchError(row.expErr))
			} else {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
			gomega.Expect(out).To(gomega.Equal(row.expValue))
		})
	}
}
