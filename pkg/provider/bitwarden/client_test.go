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

package bitwarden

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var projectID = "e8fc8f9c-2208-446e-9e89-9bc358f39b47"

func TestProviderDeleteSecret(t *testing.T) {
	type fields struct {
		kube       client.Client
		namespace  string
		store      v1beta1.GenericStore
		mock       func(c *FakeClient)
		assertMock func(t *testing.T, c *FakeClient)
	}
	type args struct {
		ctx context.Context
		ref v1beta1.PushSecretRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "delete secret is successfully with UUID",
			fields: fields{
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.DeleteSecretReturnsOnCallN(0, &SecretsDeleteResponse{})
				},
				assertMock: func(t *testing.T, c *FakeClient) {
					assert.Equal(t, 1, c.deleteSecretCalledN)
				},
			},
			args: args{
				ctx: context.TODO(),
				ref: v1alpha1.PushSecretRemoteRef{
					RemoteKey: "d8f29773-3019-4973-9bbc-66327d077fe2",
				},
			},
		},
		{
			name: "delete secret by name",
			fields: fields{
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.ListSecretReturnsOnCallN(0, &SecretIdentifiersResponse{
						Data: []SecretIdentifierResponse{
							{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "this-is-a-name",
								OrganizationID: "orgid",
							},
						},
					})

					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "key",
						Note:           "note",
						OrganizationID: "org",
						Value:          "value",
						ProjectID:      &projectID,
					})
					c.DeleteSecretReturnsOnCallN(0, &SecretsDeleteResponse{})
				},
				assertMock: func(t *testing.T, c *FakeClient) {
					assert.Equal(t, 1, c.deleteSecretCalledN)
				},
			},
			args: args{
				ctx: context.TODO(),
				ref: v1alpha1.PushSecretRemoteRef{
					RemoteKey: "d8f29773-3019-4973-9bbc-66327d077fe2",
				},
			},
		},
		{
			name: "delete secret by name will not delete if something doesn't match",
			fields: fields{
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.ListSecretReturnsOnCallN(0, &SecretIdentifiersResponse{
						Data: []SecretIdentifierResponse{
							{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "this-is-a-name",
								OrganizationID: "orgid",
							},
						},
					})

					projectID := "another-project"
					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "this-is-a-name",
						Note:           "note",
						OrganizationID: "orgid",
						Value:          "value",
						ProjectID:      &projectID,
					})
				},
				assertMock: func(t *testing.T, c *FakeClient) {
					assert.Equal(t, 0, c.deleteSecretCalledN)
				},
			},
			wantErr: true, // no secret found
			args: args{
				ctx: context.TODO(),
				ref: v1alpha1.PushSecretRemoteRef{
					RemoteKey: "this-is-a-name",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := &FakeClient{}
			tt.fields.mock(fakeClient)

			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: fakeClient,
			}
			if err := p.DeleteSecret(tt.args.ctx, tt.args.ref); (err != nil) != tt.wantErr {
				t.Errorf("DeleteSecret() error = %v, wantErr %v", err, tt.wantErr)
			}

			tt.fields.assertMock(t, fakeClient)
		})
	}
}

func TestProviderGetAllSecrets(t *testing.T) {
	type fields struct {
		kube      client.Client
		namespace string
		store     v1beta1.GenericStore
		mock      func(c *FakeClient)
	}
	type args struct {
		ctx context.Context
		ref v1beta1.ExternalSecretFind
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "get all secrets",
			fields: fields{
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.ListSecretReturnsOnCallN(0, &SecretIdentifiersResponse{
						Data: []SecretIdentifierResponse{
							{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "key1",
								OrganizationID: "orgid",
							},
							{
								ID:             "7c0d21ec-10d9-4972-bdf8-ec52df99cc86",
								Key:            "key2",
								OrganizationID: "orgid",
							},
						},
					})

					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:    "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:   "key1",
						Value: "value1",
					})
					c.GetSecretReturnsOnCallN(1, &SecretResponse{
						ID:    "7c0d21ec-10d9-4972-bdf8-ec52df99cc86",
						Key:   "key2",
						Value: "value2",
					})
				},
			},
			args: args{
				ctx: context.TODO(),
				ref: v1beta1.ExternalSecretFind{},
			},
			want: map[string][]byte{
				"d8f29773-3019-4973-9bbc-66327d077fe2": []byte("value1"),
				"7c0d21ec-10d9-4972-bdf8-ec52df99cc86": []byte("value2"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := &FakeClient{}
			tt.fields.mock(fakeClient)

			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: fakeClient,
			}
			got, err := p.GetAllSecrets(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllSecrets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllSecrets() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderGetSecret(t *testing.T) {
	type fields struct {
		kube      func() client.Client
		namespace string
		store     v1beta1.GenericStore
		mock      func(c *FakeClient)
	}
	type args struct {
		ctx context.Context
		ref v1beta1.ExternalSecretDataRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "get secret with UUID",
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store:     &v1beta1.SecretStore{},
				mock: func(c *FakeClient) {
					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "id",
						Key:            "key",
						Note:           "note",
						OrganizationID: "org",
						Value:          "value",
					})
				},
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key: "d8f29773-3019-4973-9bbc-66327d077fe2",
				},
			},
			want: []byte("value"),
		},
		{
			name: "get secret by name",
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.ListSecretReturnsOnCallN(0, &SecretIdentifiersResponse{
						Data: []SecretIdentifierResponse{
							{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "this-is-a-name",
								OrganizationID: "orgid",
							},
						},
					})

					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "key",
						Note:           "note",
						OrganizationID: "org",
						Value:          "value",
						ProjectID:      &projectID,
					})
				},
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key: "this-is-a-name",
				},
			},
			want: []byte("value"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := &FakeClient{}
			tt.fields.mock(fakeClient)

			p := &Provider{
				kube:               tt.fields.kube(),
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: fakeClient,
			}
			got, err := p.GetSecret(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderPushSecret(t *testing.T) {
	type fields struct {
		kube       func() client.Client
		namespace  string
		store      v1beta1.GenericStore
		mock       func(c *FakeClient)
		assertMock func(t *testing.T, c *FakeClient)
	}
	type args struct {
		ctx    context.Context
		secret *corev1.Secret
		data   v1beta1.PushSecretData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "push secret is successful for a none existent remote secret",
			args: args{
				ctx: context.Background(),
				secret: &corev1.Secret{
					Data: map[string][]byte{
						"key": []byte("value"),
					},
				},
				data: v1alpha1.PushSecretData{
					Match: v1alpha1.PushSecretMatch{
						SecretKey: "key",
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: "this-is-a-name",
						},
					},
				},
			},
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.ListSecretReturnsOnCallN(0, &SecretIdentifiersResponse{
						Data: []SecretIdentifierResponse{
							{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "this-is-a-name",
								OrganizationID: "orgid",
							},
						},
					})
					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "no-match", // if this is this-is-a-name it would match
						Note:           "",
						OrganizationID: "orgid",
						Value:          "value",
						ProjectID:      &projectID,
					})
					c.CreateSecretReturnsOnCallN(0, &SecretResponse{})
				},
				assertMock: func(t *testing.T, c *FakeClient) {
					cargs := c.createSecretCallArguments[0]
					assert.Equal(t, cargs, SecretCreateRequest{
						Key:            "this-is-a-name",
						Note:           "",
						OrganizationID: "orgid",
						ProjectIDS:     []string{projectID},
						Value:          "value",
					})
				},
			},
		},
		{
			name: "push secret is successful for a existing remote secret but only the value differs will call update",
			args: args{
				ctx: context.Background(),
				secret: &corev1.Secret{
					Data: map[string][]byte{
						"key": []byte("new-value"),
					},
				},
				data: v1alpha1.PushSecretData{
					Match: v1alpha1.PushSecretMatch{
						SecretKey: "key",
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: "this-is-a-name",
						},
					},
				},
			},
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.ListSecretReturnsOnCallN(0, &SecretIdentifiersResponse{
						Data: []SecretIdentifierResponse{
							{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "this-is-a-name",
								OrganizationID: "orgid",
							},
						},
					})
					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "this-is-a-name",
						Note:           "",
						OrganizationID: "orgid",
						Value:          "value",
						ProjectID:      &projectID,
					})
					c.UpdateSecretReturnsOnCallN(0, &SecretResponse{})
				},
				assertMock: func(t *testing.T, c *FakeClient) {
					pargs := c.updateSecretCallArguments[0]
					assert.Equal(t, pargs, SecretPutRequest{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "this-is-a-name",
						Note:           "",
						OrganizationID: "orgid",
						ProjectIDS:     []string{projectID},
						Value:          "new-value",
					})
				},
			},
		},
		{
			name: "push secret will not push if the same secret already exists",
			args: args{
				ctx: context.Background(),
				secret: &corev1.Secret{
					Data: map[string][]byte{
						"key": []byte("value"),
					},
				},
				data: v1alpha1.PushSecretData{
					Match: v1alpha1.PushSecretMatch{
						SecretKey: "key",
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: "this-is-a-name",
						},
					},
				},
			},
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.ListSecretReturnsOnCallN(0, &SecretIdentifiersResponse{
						Data: []SecretIdentifierResponse{
							{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "this-is-a-name",
								OrganizationID: "orgid",
							},
						},
					})
					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "this-is-a-name",
						OrganizationID: "orgid",
						Value:          "value",
						ProjectID:      &projectID,
					})
					c.UpdateSecretReturnsOnCallN(0, &SecretResponse{})
				},
				assertMock: func(t *testing.T, c *FakeClient) {
					assert.Equal(t, 0, c.createSecretCalledN)
					assert.Equal(t, 0, c.updateSecretCalledN)
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := &FakeClient{}
			tt.fields.mock(fakeClient)

			p := &Provider{
				kube:               tt.fields.kube(),
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: fakeClient,
			}

			if err := p.PushSecret(tt.args.ctx, tt.args.secret, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("PushSecret() error = %v, wantErr %v", err, tt.wantErr)
			}

			tt.fields.assertMock(t, fakeClient)
		})
	}
}

func TestProviderSecretExists(t *testing.T) {
	type fields struct {
		kube       client.Client
		namespace  string
		store      v1beta1.GenericStore
		mock       func(c *FakeClient)
		assertMock func(t *testing.T, c *FakeClient)
	}
	type args struct {
		ctx context.Context
		ref v1alpha1.PushSecretData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "secret exists",
			fields: fields{
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.GetSecretReturnsOnCallN(0, &SecretResponse{})
				},
				assertMock: func(t *testing.T, c *FakeClient) {
					assert.Equal(t, 0, c.listSecretsCalledN)
				},
			},
			args: args{
				ctx: nil,
				ref: v1alpha1.PushSecretData{
					Match: v1alpha1.PushSecretMatch{
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: "d8f29773-3019-4973-9bbc-66327d077fe2",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "secret exists by name",
			fields: fields{
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.ListSecretReturnsOnCallN(0, &SecretIdentifiersResponse{
						Data: []SecretIdentifierResponse{
							{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "name",
								OrganizationID: "orgid",
							},
						},
					})
					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "name",
						OrganizationID: "orgid",
						Value:          "value",
						ProjectID:      &projectID,
					})
				},
			},
			args: args{
				ctx: nil,
				ref: v1alpha1.PushSecretData{
					Match: v1alpha1.PushSecretMatch{
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: "name",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "secret not found by name",
			fields: fields{
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
					c.ListSecretReturnsOnCallN(0, &SecretIdentifiersResponse{
						Data: []SecretIdentifierResponse{
							{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "name",
								OrganizationID: "orgid",
							},
						},
					})
					projectIDDifferent := "different-project"
					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "name",
						OrganizationID: "orgid",
						Value:          "value",
						ProjectID:      &projectIDDifferent,
					})
				},
			},
			args: args{
				ctx: nil,
				ref: v1alpha1.PushSecretData{
					Match: v1alpha1.PushSecretMatch{
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: "name",
						},
					},
				},
			},
			want:    false,
			wantErr: true, // secret not found
		},
		{
			name: "invalid name format should error",
			fields: fields{
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
								ProjectID:      projectID,
							},
						},
					},
				},
				mock: func(c *FakeClient) {
				},
				assertMock: func(t *testing.T, c *FakeClient) {
					assert.Equal(t, 0, c.listSecretsCalledN)
				},
			},
			args: args{
				ctx: nil,
				ref: v1alpha1.PushSecretData{
					Match: v1alpha1.PushSecretMatch{
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: "name",
						},
					},
				},
			},
			want:    false,
			wantErr: true, // invalid remote key format
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := &FakeClient{}
			tt.fields.mock(fakeClient)

			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: fakeClient,
			}
			got, err := p.SecretExists(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecretExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SecretExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderGetSecretMap(t *testing.T) {
	type fields struct {
		kube      func() client.Client
		namespace string
		store     v1beta1.GenericStore
		mock      func(c *FakeClient)
	}
	type args struct {
		ctx context.Context
		ref v1beta1.ExternalSecretDataRemoteRef
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "get secret map",
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store:     &v1beta1.SecretStore{},
				mock: func(c *FakeClient) {
					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "key",
						Note:           "note",
						OrganizationID: "org",
						Value:          `{"key": "value"}`,
					})
				},
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key:      "d8f29773-3019-4973-9bbc-66327d077fe2",
					Property: "key",
				},
				key: "key",
			},
			want: []byte("value"),
		},
		{
			name: "get secret map - missing key",
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store:     &v1beta1.SecretStore{},
				mock: func(c *FakeClient) {
					c.GetSecretReturnsOnCallN(0, &SecretResponse{
						ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
						Key:            "key",
						Note:           "note",
						OrganizationID: "org",
						Value:          `{"key": "value"}`,
					})
				},
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key:      "d8f29773-3019-4973-9bbc-66327d077fe2",
					Property: "nope",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := &FakeClient{}
			tt.fields.mock(fakeClient)

			p := &Provider{
				kube:               tt.fields.kube(),
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: fakeClient,
			}
			got, err := p.GetSecretMap(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got[tt.args.key])
		})
	}
}
