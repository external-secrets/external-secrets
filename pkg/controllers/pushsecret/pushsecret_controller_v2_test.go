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

package pushsecret

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	"github.com/external-secrets/external-secrets/runtime/clientmanager"
)

type pushsecretRecordingProviderServer struct {
	pb.UnimplementedSecretStoreProviderServer
	pushRequest   *pb.PushSecretRequest
	deleteRequest *pb.DeleteSecretRequest
}

const (
	pushSecretManifestNamespace = "tenant-a"
	pushSecretRemoteKey         = "remote/path"
	pushSecretProperty          = "property"
	pushSecretSecretKey         = "token"
)

func (s *pushsecretRecordingProviderServer) PushSecret(_ context.Context, req *pb.PushSecretRequest) (*pb.PushSecretResponse, error) {
	s.pushRequest = req
	return &pb.PushSecretResponse{}, nil
}

func (s *pushsecretRecordingProviderServer) DeleteSecret(_ context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	s.deleteRequest = req
	return &pb.DeleteSecretResponse{}, nil
}

func (s *pushsecretRecordingProviderServer) SecretExists(_ context.Context, _ *pb.SecretExistsRequest) (*pb.SecretExistsResponse, error) {
	return &pb.SecretExistsResponse{Exists: false}, nil
}

func TestResolvedStoreInfoSupportsProviderKinds(t *testing.T) {
	providerInfo, ok := resolvedStoreInfo(esapi.PushSecretStoreRef{
		Name: "provider",
		Kind: esv1.ProviderKindStr,
	}, &esv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "provider",
			Labels: map[string]string{"team": "a"},
		},
	})
	if !ok {
		t.Fatal("expected provider store info to resolve")
	}
	if providerInfo.Name != "provider" || providerInfo.Kind != esv1.ProviderKindStr || providerInfo.Labels["team"] != "a" {
		t.Fatalf("unexpected provider info: %#v", providerInfo)
	}

	clusterProviderInfo, ok := resolvedStoreInfo(esapi.PushSecretStoreRef{
		Name: "cluster-provider",
		Kind: esv1.ClusterProviderKindStr,
	}, &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cluster-provider",
			Labels: map[string]string{"scope": "cluster"},
		},
	})
	if !ok {
		t.Fatal("expected cluster provider store info to resolve")
	}
	if clusterProviderInfo.Name != "cluster-provider" || clusterProviderInfo.Kind != esv1.ClusterProviderKindStr || clusterProviderInfo.Labels["scope"] != "cluster" {
		t.Fatalf("unexpected cluster provider info: %#v", clusterProviderInfo)
	}
}

func TestResolvedStoreInfoInfersOmittedProviderKinds(t *testing.T) {
	providerInfo, ok := resolvedStoreInfo(esapi.PushSecretStoreRef{
		Name: "provider",
	}, &esv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "provider",
			Labels: map[string]string{"team": "a"},
		},
	})
	if !ok {
		t.Fatal("expected provider store info to resolve")
	}
	if providerInfo.Kind != esv1.ProviderKindStr {
		t.Fatalf("expected kind %q, got %#v", esv1.ProviderKindStr, providerInfo)
	}

	clusterProviderInfo, ok := resolvedStoreInfo(esapi.PushSecretStoreRef{
		Name: "cluster-provider",
	}, &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cluster-provider",
			Labels: map[string]string{"scope": "cluster"},
		},
	})
	if !ok {
		t.Fatal("expected cluster provider store info to resolve")
	}
	if clusterProviderInfo.Kind != esv1.ClusterProviderKindStr {
		t.Fatalf("expected kind %q, got %#v", esv1.ClusterProviderKindStr, clusterProviderInfo)
	}
}

func TestValidateDataToMatchesResolvedStoresSupportsProviderKinds(t *testing.T) {
	err := validateDataToMatchesResolvedStores([]esapi.PushSecretDataTo{
		{
			StoreRef: &esapi.PushSecretStoreRef{
				Kind: esv1.ProviderKindStr,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"team": "a"},
				},
			},
			RemoteKey: "bundle",
		},
	}, []storeInfo{
		{Name: "provider", Kind: esv1.ProviderKindStr, Labels: map[string]string{"team": "a"}},
	})
	if err != nil {
		t.Fatalf("expected provider label selector to match, got %v", err)
	}

	err = validateDataToMatchesResolvedStores([]esapi.PushSecretDataTo{
		{
			StoreRef: &esapi.PushSecretStoreRef{
				Kind: esv1.ClusterProviderKindStr,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"scope": "missing"},
				},
			},
			RemoteKey: "bundle",
		},
	}, []storeInfo{
		{Name: "cluster-provider", Kind: esv1.ClusterProviderKindStr, Labels: map[string]string{"scope": "cluster"}},
	})
	if err == nil || err.Error() != "dataTo[0]: labelSelector does not match any store in secretStoreRefs" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushSecretToProvidersV2UsesProviderPath(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	provider := &esv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider",
			Namespace: "tenant-a",
			Labels:    map[string]string{"team": "a"},
		},
		Spec: esv1.ProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
				},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(provider, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "external-secrets-provider-tls",
				Namespace: "tenant-a",
			},
			Data: tlsSecret,
		}).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	mgr := clientmanager.NewManager(kubeClient, "", false)
	defer func() {
		_ = mgr.Close(context.Background())
	}()

	ps := esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Name: "provider",
				Kind: esv1.ProviderKindStr,
			}},
			Data: []esapi.PushSecretData{{
				Match: esapi.PushSecretMatch{
					SecretKey: pushSecretSecretKey,
					RemoteRef: esapi.PushSecretRemoteRef{
						RemoteKey: pushSecretRemoteKey,
						Property:  pushSecretProperty,
					},
				},
				Metadata: &apiextensionsv1.JSON{Raw: []byte(`{"owner":"eso"}`)},
			}},
		},
	}

	secret := &corev1.Secret{
		Data: map[string][]byte{pushSecretSecretKey: []byte("value")},
	}

	synced, err := r.PushSecretToProvidersV2(context.Background(), map[esapi.PushSecretStoreRef]any{
		{Name: "provider", Kind: esv1.ProviderKindStr}: provider,
	}, ps, secret, mgr)
	if err != nil {
		t.Fatalf("PushSecretToProvidersV2() error = %v", err)
	}

	if server.pushRequest == nil {
		t.Fatal("expected push request to be recorded")
	}
	if server.pushRequest.SourceNamespace != pushSecretManifestNamespace {
		t.Fatalf("unexpected source namespace: %q", server.pushRequest.SourceNamespace)
	}
	if server.pushRequest.ProviderRef == nil || server.pushRequest.ProviderRef.Name != "backend" {
		t.Fatalf("unexpected provider ref: %#v", server.pushRequest.ProviderRef)
	}
	if string(server.pushRequest.SecretData[pushSecretSecretKey]) != "value" {
		t.Fatalf("unexpected secret data: %#v", server.pushRequest.SecretData)
	}
	if server.pushRequest.PushSecretData == nil || server.pushRequest.PushSecretData.RemoteKey != pushSecretRemoteKey || server.pushRequest.PushSecretData.Property != pushSecretProperty {
		t.Fatalf("unexpected push payload: %#v", server.pushRequest.PushSecretData)
	}
	if string(server.pushRequest.PushSecretData.Metadata) != `{"owner":"eso"}` {
		t.Fatalf("unexpected metadata: %q", string(server.pushRequest.PushSecretData.Metadata))
	}
	if synced["Provider/provider"]["remote/path/property"].Match.SecretKey != pushSecretSecretKey {
		t.Fatalf("unexpected synced map: %#v", synced)
	}
}

func TestPushSecretToProvidersV2UsesProviderPathWhenKindOmitted(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	provider := &esv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider",
			Namespace: pushSecretManifestNamespace,
			Labels:    map[string]string{"team": "a"},
		},
		Spec: esv1.ProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
				},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(provider, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "external-secrets-provider-tls",
				Namespace: pushSecretManifestNamespace,
			},
			Data: tlsSecret,
		}).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	mgr := clientmanager.NewManager(kubeClient, "", false)
	defer func() {
		_ = mgr.Close(context.Background())
	}()

	ps := esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Name: "provider",
			}},
			Data: []esapi.PushSecretData{{
				Match: esapi.PushSecretMatch{
					SecretKey: "token",
					RemoteRef: esapi.PushSecretRemoteRef{
						RemoteKey: "remote/path",
						Property:  "property",
					},
				},
				Metadata: &apiextensionsv1.JSON{Raw: []byte(`{"owner":"eso"}`)},
			}},
		},
	}

	secret := &corev1.Secret{
		Data: map[string][]byte{"token": []byte("value")},
	}

	synced, err := r.PushSecretToProvidersV2(context.Background(), map[esapi.PushSecretStoreRef]any{
		{Name: "provider"}: provider,
	}, ps, secret, mgr)
	if err != nil {
		t.Fatalf("PushSecretToProvidersV2() error = %v", err)
	}

	if server.pushRequest == nil {
		t.Fatal("expected push request to be recorded")
	}
	if synced["Provider/provider"]["remote/path/property"].Match.SecretKey != "token" {
		t.Fatalf("unexpected synced map: %#v", synced)
	}
}

func TestPushSecretToProvidersV2UsesProviderNamespaceAuthScope(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	const manifestNamespace = "tenant-a"
	const providerNamespace = "provider-config-ns"

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cluster-provider",
			Labels: map[string]string{"scope": "cluster"},
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
					Namespace:  providerNamespace,
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeProviderNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{{
				Namespaces: []string{manifestNamespace},
			}},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			clusterProvider,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: manifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": manifestNamespace,
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: providerNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	mgr := clientmanager.NewManager(kubeClient, "", false)
	defer func() {
		_ = mgr.Close(context.Background())
	}()

	ps := esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: manifestNamespace,
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Name: "cluster-provider",
				Kind: esv1.ClusterProviderKindStr,
			}},
			Data: []esapi.PushSecretData{{
				Match: esapi.PushSecretMatch{
					SecretKey: "token",
					RemoteRef: esapi.PushSecretRemoteRef{
						RemoteKey: "remote/path",
						Property:  "property",
					},
				},
				Metadata: &apiextensionsv1.JSON{Raw: []byte(`{"owner":"eso"}`)},
			}},
		},
	}

	secret := &corev1.Secret{
		Data: map[string][]byte{"token": []byte("value")},
	}

	synced, err := r.PushSecretToProvidersV2(context.Background(), map[esapi.PushSecretStoreRef]any{
		{Name: "cluster-provider", Kind: esv1.ClusterProviderKindStr}: clusterProvider,
	}, ps, secret, mgr)
	if err != nil {
		t.Fatalf("PushSecretToProvidersV2() error = %v", err)
	}

	if server.pushRequest == nil {
		t.Fatal("expected push request to be recorded")
	}
	if server.pushRequest.SourceNamespace != providerNamespace {
		t.Fatalf("unexpected source namespace: %q", server.pushRequest.SourceNamespace)
	}
	if server.pushRequest.ProviderRef == nil || server.pushRequest.ProviderRef.Name != "backend" || server.pushRequest.ProviderRef.Namespace != providerNamespace {
		t.Fatalf("unexpected provider ref: %#v", server.pushRequest.ProviderRef)
	}
	if synced["ClusterProvider/cluster-provider"]["remote/path/property"].Match.SecretKey != "token" {
		t.Fatalf("unexpected synced map: %#v", synced)
	}
}

func TestDeleteSecretFromProvidersV2UsesClusterProviderPath(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cluster-provider",
			Labels: map[string]string{"scope": "cluster"},
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeManifestNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{{
				Namespaces: []string{"tenant-a"},
			}},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			clusterProvider,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "tenant-a",
				Labels: map[string]string{
					"kubernetes.io/metadata.name": "tenant-a",
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: "tenant-a",
				},
				Data: tlsSecret,
			},
		).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := &esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Status: esapi.PushSecretStatus{
			SyncedPushSecrets: esapi.SyncedPushSecretsMap{
				"ClusterProvider/cluster-provider": {
					"remote/path": {
						Match: esapi.PushSecretMatch{
							SecretKey: "token",
							RemoteRef: esapi.PushSecretRemoteRef{
								RemoteKey: "remote/path",
								Property:  "property",
							},
						},
					},
				},
			},
		},
	}

	result, err := r.DeleteSecretFromProvidersV2(context.Background(), ps, esapi.SyncedPushSecretsMap{}, map[esapi.PushSecretStoreRef]any{
		{Name: "cluster-provider", Kind: esv1.ClusterProviderKindStr}: clusterProvider,
	})
	if err != nil {
		t.Fatalf("DeleteSecretFromProvidersV2() error = %v", err)
	}

	if server.deleteRequest == nil {
		t.Fatal("expected delete request to be recorded")
	}
	if server.deleteRequest.SourceNamespace != pushSecretManifestNamespace {
		t.Fatalf("unexpected source namespace: %q", server.deleteRequest.SourceNamespace)
	}
	if server.deleteRequest.RemoteRef == nil || server.deleteRequest.RemoteRef.RemoteKey != pushSecretRemoteKey || server.deleteRequest.RemoteRef.Property != pushSecretProperty {
		t.Fatalf("unexpected delete ref: %#v", server.deleteRequest.RemoteRef)
	}
	if _, ok := result["ClusterProvider/cluster-provider"]; ok {
		t.Fatalf("expected synced state to be cleaned up, got %#v", result)
	}
}

func TestDeleteSecretFromProvidersV2UsesClusterProviderPathWhenKindOmitted(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cluster-provider",
			Labels: map[string]string{"scope": "cluster"},
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeManifestNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{{
				Namespaces: []string{"tenant-a"},
			}},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			clusterProvider,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "tenant-a",
				Labels: map[string]string{
					"kubernetes.io/metadata.name": "tenant-a",
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: "tenant-a",
				},
				Data: tlsSecret,
			},
		).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := &esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Status: esapi.PushSecretStatus{
			SyncedPushSecrets: esapi.SyncedPushSecretsMap{
				"ClusterProvider/cluster-provider": {
					"remote/path": {
						Match: esapi.PushSecretMatch{
							SecretKey: "token",
							RemoteRef: esapi.PushSecretRemoteRef{
								RemoteKey: "remote/path",
								Property:  "property",
							},
						},
					},
				},
			},
		},
	}

	result, err := r.DeleteSecretFromProvidersV2(context.Background(), ps, esapi.SyncedPushSecretsMap{}, map[esapi.PushSecretStoreRef]any{
		{Name: "cluster-provider"}: clusterProvider,
	})
	if err != nil {
		t.Fatalf("DeleteSecretFromProvidersV2() error = %v", err)
	}

	if server.deleteRequest == nil {
		t.Fatal("expected delete request to be recorded")
	}
	if _, ok := result["ClusterProvider/cluster-provider"]; ok {
		t.Fatalf("expected synced state to be cleaned up, got %#v", result)
	}
}

func TestGetSecretStoresV2ResolvesClusterProviderWhenKindOmitted(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(clusterProvider).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Name: "cluster-provider",
			}},
		},
	}

	stores, err := r.GetSecretStoresV2(context.Background(), ps)
	if err != nil {
		t.Fatalf("GetSecretStoresV2() error = %v", err)
	}

	store, ok := stores[esapi.PushSecretStoreRef{Name: "cluster-provider"}]
	if !ok {
		t.Fatalf("expected cluster provider store, got %#v", stores)
	}
	if _, ok := store.(*esv1.ClusterProvider); !ok {
		t.Fatalf("expected ClusterProvider, got %T", store)
	}
}

func TestGetSecretStoresV2PrefersSecretStoreWhenKindOmitted(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	secretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-name",
			Namespace: "tenant-a",
		},
	}
	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "shared-name",
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secretStore, clusterProvider).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Name: "shared-name",
			}},
		},
	}

	stores, err := r.GetSecretStoresV2(context.Background(), ps)
	if err != nil {
		t.Fatalf("GetSecretStoresV2() error = %v", err)
	}

	store, ok := stores[esapi.PushSecretStoreRef{Name: "shared-name"}]
	if !ok {
		t.Fatalf("expected store to resolve, got %#v", stores)
	}
	if _, ok := store.(*esv1.SecretStore); !ok {
		t.Fatalf("expected SecretStore to take precedence, got %T", store)
	}
}

func TestGetSecretStoresV2SupportsSecretStoreLabelSelectors(t *testing.T) {
	scheme := newPushSecretTestScheme(t)
	selectedStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selected",
			Namespace: "tenant-a",
			Labels:    map[string]string{"env": "test"},
		},
	}
	otherNamespaceStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-namespace",
			Namespace: "tenant-b",
			Labels:    map[string]string{"env": "test"},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(selectedStore, otherNamespaceStore).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Kind: esv1.SecretStoreKind,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "test"},
				},
			}},
		},
	}

	stores, err := r.GetSecretStoresV2(context.Background(), ps)
	if err != nil {
		t.Fatalf("GetSecretStoresV2() error = %v", err)
	}

	if len(stores) != 1 {
		t.Fatalf("expected one resolved store, got %#v", stores)
	}

	store, ok := stores[esapi.PushSecretStoreRef{Name: "selected", Kind: esv1.SecretStoreKind}]
	if !ok {
		t.Fatalf("expected selected store, got %#v", stores)
	}
	if _, ok := store.(*esv1.SecretStore); !ok {
		t.Fatalf("expected SecretStore, got %T", store)
	}
}

func TestRemoveUnmanagedStoresSupportsOmittedKindRefs(t *testing.T) {
	scheme := newPushSecretTestScheme(t)
	secretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-name",
			Namespace: "tenant-a",
		},
		Spec: esv1.SecretStoreSpec{
			Controller: "eso",
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secretStore).
		Build()

	r := &Reconciler{
		Client:          kubeClient,
		ControllerClass: "eso",
		Log:             logr.Discard(),
	}

	stores, err := removeUnmanagedStores(context.Background(), "tenant-a", r, map[esapi.PushSecretStoreRef]esv1.GenericStore{
		{Name: "shared-name"}: secretStore,
	})
	if err != nil {
		t.Fatalf("removeUnmanagedStores() error = %v", err)
	}

	if _, ok := stores[esapi.PushSecretStoreRef{Name: "shared-name"}]; !ok {
		t.Fatalf("expected omitted-kind store ref to be retained, got %#v", stores)
	}
}

func TestDeleteSecretFromProvidersV2DeletesRemovedStoreEvenWhenNoLongerReferenced(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeManifestNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{{
				Namespaces: []string{"tenant-a"},
			}},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			clusterProvider,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "tenant-a",
				Labels: map[string]string{
					"kubernetes.io/metadata.name": "tenant-a",
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: "tenant-a",
				},
				Data: tlsSecret,
			},
		).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := &esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Status: esapi.PushSecretStatus{
			SyncedPushSecrets: esapi.SyncedPushSecretsMap{
				"ClusterProvider/cluster-provider": {
					"remote/path": {
						Match: esapi.PushSecretMatch{
							SecretKey: "token",
							RemoteRef: esapi.PushSecretRemoteRef{
								RemoteKey: "remote/path",
								Property:  "property",
							},
						},
					},
				},
			},
		},
	}

	result, err := r.DeleteSecretFromProvidersV2(context.Background(), ps, esapi.SyncedPushSecretsMap{}, map[esapi.PushSecretStoreRef]any{})
	if err != nil {
		t.Fatalf("DeleteSecretFromProvidersV2() error = %v", err)
	}

	if server.deleteRequest == nil {
		t.Fatal("expected delete request to be recorded")
	}
	if server.deleteRequest.RemoteRef == nil || server.deleteRequest.RemoteRef.RemoteKey != "remote/path" {
		t.Fatalf("unexpected delete ref: %#v", server.deleteRequest.RemoteRef)
	}
	if _, ok := result["ClusterProvider/cluster-provider"]; ok {
		t.Fatalf("expected synced state to be cleaned up, got %#v", result)
	}
}

func TestDeleteSecretFromProvidersV2UsesProviderNamespaceAuthScope(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	const manifestNamespace = "tenant-a"
	const providerNamespace = "provider-config-ns"

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cluster-provider",
			Labels: map[string]string{"scope": "cluster"},
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
					Namespace:  providerNamespace,
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeProviderNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{{
				Namespaces: []string{manifestNamespace},
			}},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			clusterProvider,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: manifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": manifestNamespace,
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: providerNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := &esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: manifestNamespace,
		},
		Status: esapi.PushSecretStatus{
			SyncedPushSecrets: esapi.SyncedPushSecretsMap{
				"ClusterProvider/cluster-provider": {
					"remote/path": {
						Match: esapi.PushSecretMatch{
							SecretKey: "token",
							RemoteRef: esapi.PushSecretRemoteRef{
								RemoteKey: "remote/path",
								Property:  "property",
							},
						},
					},
				},
			},
		},
	}

	result, err := r.DeleteSecretFromProvidersV2(context.Background(), ps, esapi.SyncedPushSecretsMap{}, map[esapi.PushSecretStoreRef]any{
		{Name: "cluster-provider", Kind: esv1.ClusterProviderKindStr}: clusterProvider,
	})
	if err != nil {
		t.Fatalf("DeleteSecretFromProvidersV2() error = %v", err)
	}

	if server.deleteRequest == nil {
		t.Fatal("expected delete request to be recorded")
	}
	if server.deleteRequest.SourceNamespace != providerNamespace {
		t.Fatalf("unexpected source namespace: %q", server.deleteRequest.SourceNamespace)
	}
	if server.deleteRequest.ProviderRef == nil || server.deleteRequest.ProviderRef.Namespace != providerNamespace {
		t.Fatalf("unexpected provider ref: %#v", server.deleteRequest.ProviderRef)
	}
	if _, ok := result["ClusterProvider/cluster-provider"]; ok {
		t.Fatalf("expected synced state to be cleaned up, got %#v", result)
	}
}

func TestDeleteSecretFromProvidersV2DeletesOnlyRemovedEntriesForManifestScope(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeManifestNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{{
				Namespaces: []string{"tenant-a"},
			}},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			clusterProvider,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "tenant-a",
				Labels: map[string]string{
					"kubernetes.io/metadata.name": "tenant-a",
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: "tenant-a",
				},
				Data: tlsSecret,
			},
		).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := &esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Status: esapi.PushSecretStatus{
			SyncedPushSecrets: esapi.SyncedPushSecretsMap{
				"ClusterProvider/cluster-provider": {
					"remote/keep/property": {
						Match: esapi.PushSecretMatch{
							SecretKey: "keep",
							RemoteRef: esapi.PushSecretRemoteRef{
								RemoteKey: "remote/keep",
								Property:  "property",
							},
						},
					},
					"remote/delete/property": {
						Match: esapi.PushSecretMatch{
							SecretKey: "delete",
							RemoteRef: esapi.PushSecretRemoteRef{
								RemoteKey: "remote/delete",
								Property:  "property",
							},
						},
					},
				},
			},
		},
	}

	newMap := esapi.SyncedPushSecretsMap{
		"ClusterProvider/cluster-provider": {
			"remote/keep/property": ps.Status.SyncedPushSecrets["ClusterProvider/cluster-provider"]["remote/keep/property"],
		},
	}

	result, err := r.DeleteSecretFromProvidersV2(context.Background(), ps, newMap, map[esapi.PushSecretStoreRef]any{
		{Name: "cluster-provider", Kind: esv1.ClusterProviderKindStr}: clusterProvider,
	})
	if err != nil {
		t.Fatalf("DeleteSecretFromProvidersV2() error = %v", err)
	}

	if server.deleteRequest == nil {
		t.Fatal("expected delete request to be recorded")
	}
	if server.deleteRequest.SourceNamespace != "tenant-a" {
		t.Fatalf("unexpected source namespace: %q", server.deleteRequest.SourceNamespace)
	}
	if server.deleteRequest.RemoteRef == nil || server.deleteRequest.RemoteRef.RemoteKey != "remote/delete" || server.deleteRequest.RemoteRef.Property != "property" {
		t.Fatalf("unexpected delete ref: %#v", server.deleteRequest.RemoteRef)
	}

	storeState, ok := result["ClusterProvider/cluster-provider"]
	if !ok {
		t.Fatalf("expected synced state for cluster provider, got %#v", result)
	}
	if len(storeState) != 1 {
		t.Fatalf("expected one remaining synced entry, got %#v", storeState)
	}
	if _, ok := storeState["remote/keep/property"]; !ok {
		t.Fatalf("expected keep entry to remain, got %#v", storeState)
	}
	if _, ok := storeState["remote/delete/property"]; ok {
		t.Fatalf("expected delete entry to be removed, got %#v", storeState)
	}
}

func TestDeleteSecretFromProvidersV2DeletesOnlyRemovedEntriesForProviderNamespaceScope(t *testing.T) {
	previous := clientmanager.V2ProvidersEnabled()
	clientmanager.SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		clientmanager.SetV2ProvidersEnabled(previous)
	})

	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	const manifestNamespace = "tenant-a"
	const providerNamespace = "provider-config-ns"

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
					Namespace:  providerNamespace,
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeProviderNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{{
				Namespaces: []string{manifestNamespace},
			}},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			clusterProvider,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: manifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": manifestNamespace,
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: providerNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := &esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: manifestNamespace,
		},
		Status: esapi.PushSecretStatus{
			SyncedPushSecrets: esapi.SyncedPushSecretsMap{
				"ClusterProvider/cluster-provider": {
					"remote/keep/property": {
						Match: esapi.PushSecretMatch{
							SecretKey: "keep",
							RemoteRef: esapi.PushSecretRemoteRef{
								RemoteKey: "remote/keep",
								Property:  "property",
							},
						},
					},
					"remote/delete/property": {
						Match: esapi.PushSecretMatch{
							SecretKey: "delete",
							RemoteRef: esapi.PushSecretRemoteRef{
								RemoteKey: "remote/delete",
								Property:  "property",
							},
						},
					},
				},
			},
		},
	}

	newMap := esapi.SyncedPushSecretsMap{
		"ClusterProvider/cluster-provider": {
			"remote/keep/property": ps.Status.SyncedPushSecrets["ClusterProvider/cluster-provider"]["remote/keep/property"],
		},
	}

	result, err := r.DeleteSecretFromProvidersV2(context.Background(), ps, newMap, map[esapi.PushSecretStoreRef]any{
		{Name: "cluster-provider", Kind: esv1.ClusterProviderKindStr}: clusterProvider,
	})
	if err != nil {
		t.Fatalf("DeleteSecretFromProvidersV2() error = %v", err)
	}

	if server.deleteRequest == nil {
		t.Fatal("expected delete request to be recorded")
	}
	if server.deleteRequest.SourceNamespace != providerNamespace {
		t.Fatalf("unexpected source namespace: %q", server.deleteRequest.SourceNamespace)
	}
	if server.deleteRequest.ProviderRef == nil || server.deleteRequest.ProviderRef.Namespace != providerNamespace {
		t.Fatalf("unexpected provider ref: %#v", server.deleteRequest.ProviderRef)
	}
	if server.deleteRequest.RemoteRef == nil || server.deleteRequest.RemoteRef.RemoteKey != "remote/delete" || server.deleteRequest.RemoteRef.Property != "property" {
		t.Fatalf("unexpected delete ref: %#v", server.deleteRequest.RemoteRef)
	}

	storeState, ok := result["ClusterProvider/cluster-provider"]
	if !ok {
		t.Fatalf("expected synced state for cluster provider, got %#v", result)
	}
	if len(storeState) != 1 {
		t.Fatalf("expected one remaining synced entry, got %#v", storeState)
	}
	if _, ok := storeState["remote/keep/property"]; !ok {
		t.Fatalf("expected keep entry to remain, got %#v", storeState)
	}
	if _, ok := storeState["remote/delete/property"]; ok {
		t.Fatalf("expected delete entry to be removed, got %#v", storeState)
	}
}

func newPushSecretTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esapi.AddToScheme(scheme))
	return scheme
}

func newPushSecretProviderServer(t *testing.T) (*pushsecretRecordingProviderServer, string, map[string][]byte) {
	t.Helper()

	serverCert, serverKey, clientCert, clientKey, caCert := newPushSecretTLSArtifacts(t, "127.0.0.1")

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		t.Fatal("failed to append CA cert")
	}
	tlsCert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		t.Fatalf("X509KeyPair() error = %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	server := &pushsecretRecordingProviderServer{}
	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{tlsCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})))
	pb.RegisterSecretStoreProviderServer(grpcServer, server)
	go func() {
		_ = grpcServer.Serve(lis)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = lis.Close()
	})

	return server, lis.Addr().String(), map[string][]byte{
		"ca.crt":     caCert,
		"client.crt": clientCert,
		"client.key": clientKey,
	}
}

func newPushSecretTLSArtifacts(t *testing.T, host string) (serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM []byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "pushsecret-test-ca",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}

	serverCertPEM, serverKeyPEM = newPushSecretSignedTLSCert(t, caCert, caKey, 2, host, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
	clientCertPEM, clientKeyPEM = newPushSecretSignedTLSCert(t, caCert, caKey, 3, host, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth})
	caCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	return serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM
}

func newPushSecretSignedTLSCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey, serial int64, host string, usages []x509.ExtKeyUsage) ([]byte, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: usages,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}
