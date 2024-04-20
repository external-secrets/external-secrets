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

package kubernetes

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	testingfake "github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
)

const (
	errSomethingWentWrong = "Something went wrong"
)

type fakeClient struct {
	t                   *testing.T
	secretMap           map[string]*v1.Secret
	expectedListOptions metav1.ListOptions
	err                 error
}

func (fk *fakeClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*v1.Secret, error) {
	if fk.err != nil {
		return nil, fk.err
	}

	secret, ok := fk.secretMap[name]

	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, "secret")
	}
	// return inmutable to simulate external system and avoid accidental side effects
	sCopy := secret.DeepCopy()
	// update operation requires to relate names
	sCopy.Name = name
	return sCopy, nil
}

func (fk *fakeClient) List(_ context.Context, opts metav1.ListOptions) (*v1.SecretList, error) {
	assert.Equal(fk.t, fk.expectedListOptions, opts)
	list := &v1.SecretList{}
	for _, v := range fk.secretMap {
		list.Items = append(list.Items, *v)
	}
	return list, nil
}

func (fk *fakeClient) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	if fk.err != nil {
		return fk.err
	}

	_, ok := fk.secretMap[name]

	if !ok {
		return apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, "secret")
	}
	delete(fk.secretMap, name)
	return nil
}

func (fk *fakeClient) Create(_ context.Context, secret *v1.Secret, _ metav1.CreateOptions) (*v1.Secret, error) {
	s := &v1.Secret{
		Data: secret.Data,
		Type: secret.Type,
	}
	fk.secretMap[secret.Name] = s
	return s, nil
}

func (fk *fakeClient) Update(_ context.Context, secret *v1.Secret, _ metav1.UpdateOptions) (*v1.Secret, error) {
	s, ok := fk.secretMap[secret.Name]
	if !ok {
		return nil, errors.New("error while updating secret")
	}
	s.Data = secret.Data
	return s, nil
}

var binaryTestData = []byte{0x00, 0xff, 0x00, 0xff, 0xac, 0xab, 0x28, 0x21}

func TestGetSecret(t *testing.T) {
	tests := []struct {
		desc      string
		secrets   map[string]*v1.Secret
		clientErr error
		ref       esv1beta1.ExternalSecretDataRemoteRef
		want      []byte
		wantErr   string
	}{
		{
			desc: "secret data with correct property",
			secrets: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`foobar`),
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "token",
			},
			want: []byte(`foobar`),
		},
		{
			desc: "secret data with multi level property",
			secrets: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"foo": []byte(`{"huga":{"bar":"val"}}`),
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "foo.huga.bar",
			},
			want: []byte(`val`),
		},
		{
			desc: "secret data with property containing .",
			secrets: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"foo.png": []byte(`correct`),
						"foo":     []byte(`{"png":"wrong"}`),
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "foo.png",
			},
			want: []byte(`correct`),
		},
		{
			desc: "secret data contains html characters",
			secrets: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"html": []byte(`<foobar>`),
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "mysec",
			},
			want: []byte(`{"html":"<foobar>"}`),
		},
		{
			desc: "secret metadata contains html characters",
			secrets: map[string]*v1.Secret{
				"mysec": {
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"date": "today"},
						Labels:      map[string]string{"dev": "<seb>"},
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Key:            "mysec",
			},
			want: []byte(`{"annotations":{"date":"today"},"labels":{"dev":"<seb>"}}`),
		},
		{
			desc: "secret data contains binary",
			secrets: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"bindata": binaryTestData,
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "bindata",
			},
			want: binaryTestData,
		},
		{
			desc: "secret data without property",
			secrets: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`foobar`),
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "mysec",
			},
			want: []byte(`{"token":"foobar"}`),
		},
		{
			desc: "secret metadata without property",
			secrets: map[string]*v1.Secret{
				"mysec": {
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"date": "today"},
						Labels:      map[string]string{"dev": "seb"},
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Key:            "mysec",
			},
			want: []byte(`{"annotations":{"date":"today"},"labels":{"dev":"seb"}}`),
		},
		{
			desc: "secret metadata with single level property",
			secrets: map[string]*v1.Secret{
				"mysec": {
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"date": "today"},
						Labels:      map[string]string{"dev": "seb"},
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Key:            "mysec",
				Property:       "labels",
			},
			want: []byte(`{"dev":"seb"}`),
		},
		{
			desc: "secret metadata with multiple level property",
			secrets: map[string]*v1.Secret{
				"mysec": {
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"date": "today"},
						Labels:      map[string]string{"dev": "seb"},
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Key:            "mysec",
				Property:       "labels.dev",
			},
			want: []byte(`seb`),
		},
		{
			desc:      "secret is not found",
			clientErr: apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, "secret"),
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "token",
			},
			wantErr: `Secret "secret" not found`,
		},
		{
			desc: "secret data with wrong property",
			secrets: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`foobar`),
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "not-the-token",
			},
			wantErr: "property not-the-token does not exist in data of secret",
		},
		{
			desc: "secret metadata with wrong property",
			secrets: map[string]*v1.Secret{
				"mysec": {
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"date": "today"},
						Labels:      map[string]string{"dev": "seb"},
					},
				},
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Key:            "mysec",
				Property:       "foo",
			},
			wantErr: "property foo does not exist in metadata of secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			p := &Client{
				userSecretClient: &fakeClient{t: t, secretMap: tt.secrets, err: tt.clientErr},
				namespace:        "default",
			}
			got, err := p.GetSecret(context.Background(), tt.ref)
			if err != nil {
				if tt.wantErr == "" {
					t.Fatalf("failed to call GetSecret: %v", err)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("received an unexpected error: %q should have contained %q", err.Error(), tt.wantErr)
				}

				return
			}

			if tt.wantErr != "" {
				t.Fatalf("expected to receive an error but got nil")
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("received an unexpected secret: got: %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	type fields struct {
		Client       KClient
		ReviewClient RClient
		Namespace    string
	}
	tests := []struct {
		name   string
		fields fields
		ref    esv1beta1.ExternalSecretDataRemoteRef

		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "successful case metadata without property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{"date": "today"},
								Labels:      map[string]string{"dev": "seb"},
							},
						},
					},
				},
				Namespace: "default",
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Key:            "mysec",
			},
			want: map[string][]byte{"annotations": []byte("{\"date\":\"today\"}"), "labels": []byte("{\"dev\":\"seb\"}")},
		},
		{
			name: "successful case metadata with single property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{"date": "today"},
								Labels:      map[string]string{"dev": "seb"},
							},
						},
					},
				},
				Namespace: "default",
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Key:            "mysec",
				Property:       "labels",
			},
			want: map[string][]byte{"dev": []byte("\"seb\"")},
		},
		{
			name: "error case metadata with wrong property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{"date": "today"},
								Labels:      map[string]string{"dev": "seb"},
							},
						},
					},
				},
				Namespace: "default",
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				MetadataPolicy: esv1beta1.ExternalSecretMetadataPolicyFetch,
				Key:            "mysec",
				Property:       "foo",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Client{
				userSecretClient: tt.fields.Client,
				userReviewClient: tt.fields.ReviewClient,
				namespace:        tt.fields.Namespace,
			}
			got, err := p.GetSecretMap(context.Background(), tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderKubernetes.GetSecretMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProviderKubernetes.GetSecretMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllSecrets(t *testing.T) {
	type fields struct {
		Client       KClient
		ReviewClient RClient
		Namespace    string
	}
	type args struct {
		ctx context.Context
		ref esv1beta1.ExternalSecretFind
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "use regex",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							ObjectMeta: metav1.ObjectMeta{
								Name: "mysec",
							},
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
						"other": {
							ObjectMeta: metav1.ObjectMeta{
								Name: "other",
							},
							Data: map[string][]byte{
								"token": []byte(`bar`),
							},
						},
					},
				},
			},
			args: args{
				ref: esv1beta1.ExternalSecretFind{
					Name: &esv1beta1.FindName{
						RegExp: "other",
					},
				},
			},
			want: map[string][]byte{
				"other": []byte(`{"token":"bar"}`),
			},
		},
		{
			name: "use tags/labels",
			fields: fields{
				Client: &fakeClient{
					t: t,
					expectedListOptions: metav1.ListOptions{
						LabelSelector: "app=foobar",
					},
					secretMap: map[string]*v1.Secret{
						"mysec": {
							ObjectMeta: metav1.ObjectMeta{
								Name: "mysec",
							},
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
						"other": {
							ObjectMeta: metav1.ObjectMeta{
								Name: "other",
							},
							Data: map[string][]byte{
								"token": []byte(`bar`),
							},
						},
					},
				},
			},
			args: args{
				ref: esv1beta1.ExternalSecretFind{
					Tags: map[string]string{
						"app": "foobar",
					},
				},
			},
			want: map[string][]byte{
				"mysec": []byte(`{"token":"foo"}`),
				"other": []byte(`{"token":"bar"}`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Client{
				userSecretClient: tt.fields.Client,
				userReviewClient: tt.fields.ReviewClient,
				namespace:        tt.fields.Namespace,
			}
			got, err := p.GetAllSecrets(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderKubernetes.GetAllSecrets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProviderKubernetes.GetAllSecrets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	type fields struct {
		Client KClient
	}
	tests := []struct {
		name   string
		fields fields
		ref    esv1beta1.PushSecretRemoteRef

		wantSecretMap map[string]*v1.Secret
		wantErr       bool
	}{
		{
			name: "refuse to delete without property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foobar`),
							},
						},
					},
				},
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
			},
			wantErr: true,
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`foobar`),
					},
				},
			},
		},
		{
			name: "gracefully ignore not found secret",
			fields: fields{
				Client: &fakeClient{
					t:         t,
					secretMap: map[string]*v1.Secret{},
				},
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
				Property:  "token",
			},
			wantErr:       false,
			wantSecretMap: map[string]*v1.Secret{},
		},
		{
			name: "gracefully ignore not found property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foobar`),
							},
						},
					},
				},
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
				Property:  "secret",
			},
			wantErr: false,
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`foobar`),
					},
				},
			},
		},
		{
			name: "unexpected lookup error",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foobar`),
							},
						},
					},
					err: errors.New(errSomethingWentWrong),
				},
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
			},
			wantErr: true,
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`foobar`),
					},
				},
			},
		},
		{
			name: "delete whole secret if only property should be removed",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foobar`),
							},
						},
					},
				},
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
				Property:  "token",
			},
			wantErr:       false,
			wantSecretMap: map[string]*v1.Secret{},
		},
		{
			name: "multiple properties, just remove that one",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token":  []byte(`foo`),
								"secret": []byte(`bar`),
							},
						},
					},
				},
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
				Property:  "token",
			},
			wantErr: false,
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"secret": []byte(`bar`),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Client{
				userSecretClient: tt.fields.Client,
			}
			err := p.DeleteSecret(context.Background(), tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderKubernetes.DeleteSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			fClient := tt.fields.Client.(*fakeClient)
			if diff := cmp.Diff(tt.wantSecretMap, fClient.secretMap); diff != "" {
				t.Errorf("Unexpected resulting secrets map:  -want, +got :\n%s\n", diff)
			}
		})
	}
}

func TestPushSecret(t *testing.T) {
	secretKey := "secret-key"
	type fields struct {
		Client KClient
	}
	tests := []struct {
		name   string
		fields fields
		data   testingfake.PushSecretData
		secret *v1.Secret

		wantSecretMap map[string]*v1.Secret
		wantErr       bool
	}{
		{
			name: "refuse to work without property if secret key is provided",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
					},
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				RemoteKey: "mysec",
			},
			secret: &v1.Secret{
				Data: map[string][]byte{secretKey: []byte("bar")},
			},
			wantErr: true,
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`foo`),
					},
				},
			},
		},
		{
			name: "push the whole secret if neither remote property or secretKey is defined but keep existing keys",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
					},
				},
			},
			data: testingfake.PushSecretData{
				RemoteKey: "mysec",
			},
			secret: &v1.Secret{
				Data: map[string][]byte{"token2": []byte("foo")},
			},
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token":  []byte(`foo`),
						"token2": []byte(`foo`),
					},
				},
			},
		},
		{
			name: "push the whole secret while secret exists into a single property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
					},
				},
			},
			data: testingfake.PushSecretData{
				RemoteKey: "mysec",
				Property:  "token",
			},
			secret: &v1.Secret{
				Data: map[string][]byte{"foo": []byte("bar")},
			},
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`{"foo":"bar"}`),
					},
				},
			},
		},
		{
			name: "push the whole secret while secret exists but new property is defined should update the secret and keep existing key",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
					},
				},
			},
			data: testingfake.PushSecretData{
				RemoteKey: "mysec",
				Property:  "token2",
			},
			secret: &v1.Secret{
				Data: map[string][]byte{"foo": []byte("bar")},
			},
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token":  []byte(`foo`),
						"token2": []byte(`{"foo":"bar"}`),
					},
				},
			},
		},
		{
			name: "push the whole secret as json if remote property is defined but secret key is not given",
			fields: fields{
				Client: &fakeClient{
					t:         t,
					secretMap: map[string]*v1.Secret{},
				},
			},
			data: testingfake.PushSecretData{
				RemoteKey: "mysec",
				Property:  "marshaled",
			},
			secret: &v1.Secret{
				Data: map[string][]byte{
					"token":  []byte("foo"),
					"token2": []byte("2"),
				},
			},
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"marshaled": []byte(`{"token":"foo","token2":"2"}`),
					},
					Type: "Opaque",
				},
			},
		},
		{
			name: "add missing property to existing secret",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
					},
				},
			},
			secret: &v1.Secret{
				Data: map[string][]byte{secretKey: []byte("bar")},
			},
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				RemoteKey: "mysec",
				Property:  "secret",
			},
			wantErr: false,
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token":  []byte(`foo`),
						"secret": []byte(`bar`),
					},
				},
			},
		},
		{
			name: "replace existing property in existing secret",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
					},
				},
			},
			secret: &v1.Secret{
				Data: map[string][]byte{secretKey: []byte("bar")},
			},
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				RemoteKey: "mysec",
				Property:  "token",
			},
			wantErr: false,
			wantSecretMap: map[string]*v1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`bar`),
					},
				},
			},
		},
		{
			name: "create new secret",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"yoursec": {
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
					},
				},
			},
			secret: &v1.Secret{
				Data: map[string][]byte{secretKey: []byte("bar")},
			},
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				RemoteKey: "mysec",
				Property:  "secret",
			},
			wantErr: false,
			wantSecretMap: map[string]*v1.Secret{
				"yoursec": {
					Data: map[string][]byte{
						"token": []byte(`foo`),
					},
				},
				"mysec": {
					Data: map[string][]byte{
						"secret": []byte(`bar`),
					},
					Type: v1.SecretTypeOpaque,
				},
			},
		},
		{
			name: "create new dockerconfigjson secret",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*v1.Secret{
						"yoursec": {
							Data: map[string][]byte{
								"token": []byte(`foo`),
							},
						},
					},
				},
			},
			secret: &v1.Secret{
				Type: v1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{secretKey: []byte(`{"auths": {"myregistry.localhost": {"username": "{{ .username }}", "password": "{{ .password }}"}}}`)},
			},
			data: testingfake.PushSecretData{
				SecretKey: secretKey,
				RemoteKey: "mysec",
				Property:  "config.json",
			},
			wantErr: false,
			wantSecretMap: map[string]*v1.Secret{
				"yoursec": {
					Data: map[string][]byte{
						"token": []byte(`foo`),
					},
				},
				"mysec": {
					Data: map[string][]byte{
						"config.json": []byte(`{"auths": {"myregistry.localhost": {"username": "{{ .username }}", "password": "{{ .password }}"}}}`),
					},
					Type: v1.SecretTypeDockerConfigJson,
				},
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Client{
				userSecretClient: tt.fields.Client,
				store:            &esv1beta1.KubernetesProvider{},
			}
			err := p.PushSecret(context.Background(), tt.secret, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderKubernetes error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			fClient := tt.fields.Client.(*fakeClient)
			if diff := cmp.Diff(tt.wantSecretMap, fClient.secretMap); diff != "" {
				t.Errorf("Unexpected resulting secrets map:  -want, +got :\n%s\n", diff)
			}
		})
	}
}
