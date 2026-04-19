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
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
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

func TestResolvedStoreInfoSupportsCleanStoreKinds(t *testing.T) {
	providerStoreInfo, ok := resolvedStoreInfo(esapi.PushSecretStoreRef{
		Name: "provider-store",
		Kind: esv1.ProviderStoreKindStr,
	}, &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "provider-store",
			Labels: map[string]string{"team": "a"},
		},
	})
	if !ok {
		t.Fatal("expected provider store info to resolve")
	}
	if providerStoreInfo.Name != "provider-store" || providerStoreInfo.Kind != esv1.ProviderStoreKindStr || providerStoreInfo.Labels["team"] != "a" {
		t.Fatalf("unexpected provider store info: %#v", providerStoreInfo)
	}

	clusterProviderStoreInfo, ok := resolvedStoreInfo(esapi.PushSecretStoreRef{
		Name: "cluster-provider-store",
		Kind: esv1.ClusterProviderStoreKindStr,
	}, &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cluster-provider-store",
			Labels: map[string]string{"scope": "cluster"},
		},
	})
	if !ok {
		t.Fatal("expected cluster provider store info to resolve")
	}
	if clusterProviderStoreInfo.Name != "cluster-provider-store" || clusterProviderStoreInfo.Kind != esv1.ClusterProviderStoreKindStr || clusterProviderStoreInfo.Labels["scope"] != "cluster" {
		t.Fatalf("unexpected cluster provider store info: %#v", clusterProviderStoreInfo)
	}
}

func TestResolvedStoreInfoInfersOmittedCleanStoreKinds(t *testing.T) {
	providerStoreInfo, ok := resolvedStoreInfo(esapi.PushSecretStoreRef{
		Name: "provider-store",
	}, &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "provider-store",
			Labels: map[string]string{"team": "a"},
		},
	})
	if !ok {
		t.Fatal("expected provider store info to resolve")
	}
	if providerStoreInfo.Kind != esv1.ProviderStoreKindStr {
		t.Fatalf("expected kind %q, got %#v", esv1.ProviderStoreKindStr, providerStoreInfo)
	}

	clusterProviderStoreInfo, ok := resolvedStoreInfo(esapi.PushSecretStoreRef{
		Name: "cluster-provider-store",
	}, &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cluster-provider-store",
			Labels: map[string]string{"scope": "cluster"},
		},
	})
	if !ok {
		t.Fatal("expected cluster provider store info to resolve")
	}
	if clusterProviderStoreInfo.Kind != esv1.ClusterProviderStoreKindStr {
		t.Fatalf("expected kind %q, got %#v", esv1.ClusterProviderStoreKindStr, clusterProviderStoreInfo)
	}
}

func TestValidateDataToMatchesResolvedStoresSupportsCleanStoreKinds(t *testing.T) {
	err := validateDataToMatchesResolvedStores([]esapi.PushSecretDataTo{
		{
			StoreRef: &esapi.PushSecretStoreRef{
				Kind: esv1.ProviderStoreKindStr,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"team": "a"},
				},
			},
			RemoteKey: "bundle",
		},
	}, []storeInfo{
		{Name: "provider-store", Kind: esv1.ProviderStoreKindStr, Labels: map[string]string{"team": "a"}},
	})
	if err != nil {
		t.Fatalf("expected provider store label selector to match, got %v", err)
	}

	err = validateDataToMatchesResolvedStores([]esapi.PushSecretDataTo{
		{
			StoreRef: &esapi.PushSecretStoreRef{
				Kind: esv1.ClusterProviderStoreKindStr,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"scope": "missing"},
				},
			},
			RemoteKey: "bundle",
		},
	}, []storeInfo{
		{Name: "cluster-provider-store", Kind: esv1.ClusterProviderStoreKindStr, Labels: map[string]string{"scope": "cluster"}},
	})
	if err == nil || err.Error() != "dataTo[0]: labelSelector does not match any store in secretStoreRefs" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushSecretToProvidersV2UsesProviderStorePath(t *testing.T) {
	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	store := &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-prod",
			Namespace: pushSecretManifestNamespace,
			Labels:    map[string]string{"team": "a"},
		},
		Spec: esv2alpha1.ProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "aws"},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: "provider.aws.external-secrets.io/v2alpha1",
				Kind:       "SecretsManager",
				Name:       "backend",
			},
		},
	}
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws"},
		Spec:       esv1alpha1.ClusterProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: pushSecretManifestNamespace,
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
			Namespace: "tenant-a",
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Name: store.Name,
				Kind: esv1.ProviderStoreKindStr,
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
		{Name: store.Name, Kind: esv1.ProviderStoreKindStr}: store,
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
	if server.pushRequest.ProviderRef.Namespace != pushSecretManifestNamespace || server.pushRequest.ProviderRef.StoreRefKind != esv1.ProviderStoreKindStr {
		t.Fatalf("unexpected provider ref namespace/kind: %#v", server.pushRequest.ProviderRef)
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
	if synced["ProviderStore/aws-prod"]["remote/path/property"].Match.SecretKey != pushSecretSecretKey {
		t.Fatalf("unexpected synced map: %#v", synced)
	}
}

func TestDeleteSecretFromProvidersV2UsesClusterProviderStorePath(t *testing.T) {
	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	store := &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "aws-shared",
			Labels: map[string]string{"scope": "cluster"},
		},
		Spec: esv2alpha1.ClusterProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "aws"},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: "provider.aws.external-secrets.io/v2alpha1",
				Kind:       "SecretsManager",
				Name:       "backend",
			},
		},
	}
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws"},
		Spec:       esv1alpha1.ClusterProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: pushSecretManifestNamespace,
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
				"ClusterProviderStore/aws-shared": {
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
		{Name: store.Name, Kind: esv1.ClusterProviderStoreKindStr}: store,
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
	if server.deleteRequest.ProviderRef == nil ||
		server.deleteRequest.ProviderRef.Namespace != pushSecretManifestNamespace ||
		server.deleteRequest.ProviderRef.StoreRefKind != esv1.ClusterProviderStoreKindStr {
		t.Fatalf("unexpected provider ref: %#v", server.deleteRequest.ProviderRef)
	}
	if server.deleteRequest.RemoteRef == nil || server.deleteRequest.RemoteRef.RemoteKey != pushSecretRemoteKey || server.deleteRequest.RemoteRef.Property != pushSecretProperty {
		t.Fatalf("unexpected delete ref: %#v", server.deleteRequest.RemoteRef)
	}
	if _, ok := result["ClusterProviderStore/aws-shared"]; ok {
		t.Fatalf("expected synced state to be cleaned up, got %#v", result)
	}
}

func TestGetSecretStoresV2ResolvesProviderStoreWhenAPIVersionOmitted(t *testing.T) {
	scheme := newPushSecretTestScheme(t)
	store := &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-prod",
			Namespace: "tenant-a",
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(store).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Name: "aws-prod",
				Kind: esv1.ProviderStoreKindStr,
			}},
		},
	}

	stores, err := r.GetSecretStoresV2(context.Background(), ps)
	if err != nil {
		t.Fatalf("GetSecretStoresV2() error = %v", err)
	}
	if _, ok := stores[ps.Spec.SecretStoreRefs[0]].(*esv2alpha1.ProviderStore); !ok {
		t.Fatalf("expected ProviderStore, got %#v", stores)
	}
}

func TestGetSecretStoresV2PrefersProviderStoreWhenKindOmitted(t *testing.T) {
	scheme := newPushSecretTestScheme(t)
	namespacedStore := &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-shared",
			Namespace: "tenant-a",
		},
	}
	clusterStore := &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aws-shared",
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(namespacedStore, clusterStore).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Name: "aws-shared",
			}},
		},
	}

	stores, err := r.GetSecretStoresV2(context.Background(), ps)
	if err != nil {
		t.Fatalf("GetSecretStoresV2() error = %v", err)
	}

	store, ok := stores[ps.Spec.SecretStoreRefs[0]]
	if !ok {
		t.Fatalf("expected resolved store, got %#v", stores)
	}
	if _, ok := store.(*esv2alpha1.ProviderStore); !ok {
		t.Fatalf("expected ProviderStore to win omitted-kind lookup, got %T", store)
	}
}

func TestGetSecretStoresV2ResolvesClusterProviderStoreBySelector(t *testing.T) {
	scheme := newPushSecretTestScheme(t)
	store := &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "aws-shared",
			Labels: map[string]string{"team": "shared"},
		},
	}
	otherKindStore := &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-tenant",
			Namespace: "tenant-a",
			Labels:    map[string]string{"team": "shared"},
		},
	}
	nonMatchingStore := &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "aws-other",
			Labels: map[string]string{"team": "other"},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(store, otherKindStore, nonMatchingStore).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}
	ps := esapi.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pushsecret",
			Namespace: "tenant-a",
		},
		Spec: esapi.PushSecretSpec{
			SecretStoreRefs: []esapi.PushSecretStoreRef{{
				Kind: esv1.ClusterProviderStoreKindStr,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"team": "shared"},
				},
			}},
		},
	}

	stores, err := r.GetSecretStoresV2(context.Background(), ps)
	if err != nil {
		t.Fatalf("GetSecretStoresV2() error = %v", err)
	}
	if len(stores) != 1 {
		t.Fatalf("expected one resolved store, got %d", len(stores))
	}

	selectedStore, ok := stores[esapi.PushSecretStoreRef{Name: "aws-shared", Kind: esv1.ClusterProviderStoreKindStr}]
	if !ok {
		t.Fatalf("expected selected cluster provider store, got %#v", stores)
	}
	if _, ok := selectedStore.(*esv2alpha1.ClusterProviderStore); !ok {
		t.Fatalf("expected ClusterProviderStore, got %T", selectedStore)
	}
	if _, ok := stores[esapi.PushSecretStoreRef{Name: "aws-tenant", Kind: esv1.ProviderStoreKindStr}]; ok {
		t.Fatalf("expected selector to stay within cluster provider store kind, got %#v", stores)
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

func TestDeleteSecretFromProvidersV2DeletesRemovedStoreEvenWhenNoLongerReferenced(t *testing.T) {
	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	store := &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aws-shared",
		},
		Spec: esv2alpha1.ClusterProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "aws"},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: "provider.aws.external-secrets.io/v2alpha1",
				Kind:       "SecretsManager",
				Name:       "backend",
			},
		},
	}
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws"},
		Spec:       esv1alpha1.ClusterProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: pushSecretManifestNamespace,
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
				"ClusterProviderStore/aws-shared": {
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
	if _, ok := result["ClusterProviderStore/aws-shared"]; ok {
		t.Fatalf("expected synced state to be cleaned up, got %#v", result)
	}
}

func TestDeleteSecretFromProvidersV2DeletesOnlyRemovedEntriesForClusterProviderStore(t *testing.T) {
	scheme := newPushSecretTestScheme(t)
	server, address, tlsSecret := newPushSecretProviderServer(t)

	store := &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aws-shared",
		},
		Spec: esv2alpha1.ClusterProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "aws"},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: "provider.aws.external-secrets.io/v2alpha1",
				Kind:       "SecretsManager",
				Name:       "backend",
			},
		},
	}
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws"},
		Spec:       esv1alpha1.ClusterProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: pushSecretManifestNamespace,
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
				"ClusterProviderStore/aws-shared": {
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
		"ClusterProviderStore/aws-shared": {
			"remote/keep/property": ps.Status.SyncedPushSecrets["ClusterProviderStore/aws-shared"]["remote/keep/property"],
		},
	}

	result, err := r.DeleteSecretFromProvidersV2(context.Background(), ps, newMap, map[esapi.PushSecretStoreRef]any{
		{Name: store.Name, Kind: esv1.ClusterProviderStoreKindStr}: store,
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
	if server.deleteRequest.RemoteRef == nil || server.deleteRequest.RemoteRef.RemoteKey != "remote/delete" || server.deleteRequest.RemoteRef.Property != "property" {
		t.Fatalf("unexpected delete ref: %#v", server.deleteRequest.RemoteRef)
	}

	storeState, ok := result["ClusterProviderStore/aws-shared"]
	if !ok {
		t.Fatalf("expected synced state for cluster provider store, got %#v", result)
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
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))
	utilruntime.Must(esv2alpha1.AddToScheme(scheme))
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
