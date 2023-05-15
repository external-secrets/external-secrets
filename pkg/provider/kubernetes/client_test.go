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
	"testing"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	errSomethingWentWrong = "Something went wrong"
)

type fakeClient struct {
	t                   *testing.T
	secretMap           map[string]*corev1.Secret
	expectedListOptions metav1.ListOptions
	err                 error
}

func (fk *fakeClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error) {
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

func (fk *fakeClient) List(_ context.Context, opts metav1.ListOptions) (*corev1.SecretList, error) {
	assert.Equal(fk.t, fk.expectedListOptions, opts)
	list := &corev1.SecretList{}
	for _, v := range fk.secretMap {
		list.Items = append(list.Items, *v)
	}
	return list, nil
}

func (fk *fakeClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
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

func (fk *fakeClient) Create(ctx context.Context, secret *v1.Secret, opts metav1.CreateOptions) (*v1.Secret, error) {
	s := &corev1.Secret{
		Data: secret.Data,
	}
	fk.secretMap[secret.Name] = s
	return s, nil
}

func (fk *fakeClient) Update(ctx context.Context, secret *v1.Secret, opts metav1.UpdateOptions) (*v1.Secret, error) {
	s, ok := fk.secretMap[secret.Name]
	if !ok {
		return nil, errors.New("error while updating secret")
	}
	s.Data = secret.Data
	return s, nil
}

func TestGetSecret(t *testing.T) {
	type fields struct {
		Client       KClient
		ReviewClient RClient
		Namespace    string
	}
	tests := []struct {
		name   string
		fields fields
		ref    esv1beta1.ExternalSecretDataRemoteRef

		want    []byte
		wantErr bool
	}{
		{
			name: "secretNotFound",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foobar`),
							},
						},
					},
					err: apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, "secret"),
				},
				Namespace: "default",
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "token",
			},
			wantErr: true,
		},
		{
			name: "err GetSecretMap",
			fields: fields{
				Client: &fakeClient{
					t:         t,
					secretMap: map[string]*corev1.Secret{},
				},
				Namespace: "default",
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "token",
			},
			wantErr: true,
		},
		{
			name: "wrong property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foobar`),
							},
						},
					},
				},
				Namespace: "default",
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "not-the-token",
			},
			wantErr: true,
		},
		{
			name: "successful case",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foobar`),
							},
						},
					},
				},
				Namespace: "default",
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "mysec",
				Property: "token",
			},
			want: []byte(`foobar`),
		},
		{
			name: "successful case without property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
						"mysec": {
							Data: map[string][]byte{
								"token": []byte(`foobar`),
							},
						},
					},
				},
				Namespace: "default",
			},
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "mysec",
			},
			want: []byte(`{"token":"foobar"}`),
		},
		{
			name: "successful case metadata without property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
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
			want: []byte(`{"annotations":{"date":"today"},"labels":{"dev":"seb"}}`),
		},
		{
			name: "successful case metadata with single property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
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
			want: []byte(`{"dev":"seb"}`),
		},
		{
			name: "successful case metadata with multiple properties",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
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
				Property:       "labels.dev",
			},
			want: []byte(`seb`),
		},
		{
			name: "error case metadata with wrong property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
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
			got, err := p.GetSecret(context.Background(), tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderKubernetes.GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProviderKubernetes.GetSecret() = %v, want %v", got, tt.want)
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
					secretMap: map[string]*corev1.Secret{
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
					secretMap: map[string]*corev1.Secret{
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
					secretMap: map[string]*corev1.Secret{
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
					secretMap: map[string]*corev1.Secret{
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
					secretMap: map[string]*corev1.Secret{
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
		ref    esv1beta1.PushRemoteRef

		wantSecretMap map[string]*corev1.Secret
		wantErr       bool
	}{
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
			wantSecretMap: map[string]*corev1.Secret{},
		},
		{
			name: "gracefully ignore not found property",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
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
			wantSecretMap: map[string]*corev1.Secret{
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
			name: "single property, delete whole secret",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
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
			wantSecretMap: map[string]*corev1.Secret{},
		},
		{
			name: "multiple properties, just remove that one",
			fields: fields{
				Client: &fakeClient{
					t: t,
					secretMap: map[string]*corev1.Secret{
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
			wantSecretMap: map[string]*corev1.Secret{
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
	type fields struct {
		Client    KClient
		PushValue string
	}
	tests := []struct {
		name   string
		fields fields
		ref    esv1beta1.PushRemoteRef

		wantSecretMap map[string]*corev1.Secret
		wantErr       bool
	}{
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
				PushValue: "bar",
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
				Property:  "secret",
			},
			wantErr: false,
			wantSecretMap: map[string]*corev1.Secret{
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
				PushValue: "bar",
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
				Property:  "token",
			},
			wantErr: false,
			wantSecretMap: map[string]*corev1.Secret{
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
				PushValue: "bar",
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
				Property:  "secret",
			},
			wantErr: false,
			wantSecretMap: map[string]*corev1.Secret{
				"yoursec": {
					Data: map[string][]byte{
						"token": []byte(`foo`),
					},
				},
				"mysec": {
					Data: map[string][]byte{
						"secret": []byte(`bar`),
					},
				},
			},
		},
		{
			name: "refuse to work without property",
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
				PushValue: "bar",
			},
			ref: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "mysec",
			},
			wantErr: true,
			wantSecretMap: map[string]*corev1.Secret{
				"mysec": {
					Data: map[string][]byte{
						"token": []byte(`foo`),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Client{
				userSecretClient: tt.fields.Client,
				store:            &esv1beta1.KubernetesProvider{},
			}
			err := p.PushSecret(context.Background(), []byte(tt.fields.PushValue), tt.ref)
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
