package bitwarden

import (
	"context"
	"reflect"
	"testing"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestProvider_DeleteSecret(t *testing.T) {
	type fields struct {
		kube               client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient *SdkClient
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
			name: "delete secret is successfully",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: tt.fields.bitwardenSdkClient,
			}
			if err := p.DeleteSecret(tt.args.ctx, tt.args.ref); (err != nil) != tt.wantErr {
				t.Errorf("DeleteSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProvider_GetAllSecrets(t *testing.T) {
	type fields struct {
		kube               client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient *SdkClient
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
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: tt.fields.bitwardenSdkClient,
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

func TestProvider_GetSecret(t *testing.T) {
	type fields struct {
		kube               client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient *SdkClient
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
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: tt.fields.bitwardenSdkClient,
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

func TestProvider_PushSecret(t *testing.T) {
	type fields struct {
		kube               client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient *SdkClient
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
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: tt.fields.bitwardenSdkClient,
			}
			if err := p.PushSecret(tt.args.ctx, tt.args.secret, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("PushSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProvider_SecretExists(t *testing.T) {
	type fields struct {
		kube               client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient *SdkClient
	}
	type args struct {
		ctx context.Context
		ref v1beta1.PushSecretRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: tt.fields.bitwardenSdkClient,
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
