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

package clusterprovider

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
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	k8sv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

type recordingClusterProviderGRPCServer struct {
	pb.UnimplementedSecretStoreProviderServer

	validateRequest     *pb.ValidateRequest
	capabilitiesRequest *pb.CapabilitiesRequest
	capabilitiesResp    *pb.CapabilitiesResponse
	capabilitiesErr     error
}

func (s *recordingClusterProviderGRPCServer) Validate(_ context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	s.validateRequest = req
	return &pb.ValidateResponse{Valid: true}, nil
}

func (s *recordingClusterProviderGRPCServer) Capabilities(_ context.Context, req *pb.CapabilitiesRequest) (*pb.CapabilitiesResponse, error) {
	s.capabilitiesRequest = req
	if s.capabilitiesErr != nil {
		return nil, s.capabilitiesErr
	}
	if s.capabilitiesResp != nil {
		return s.capabilitiesResp, nil
	}
	return &pb.CapabilitiesResponse{Capabilities: pb.SecretStoreCapabilities_READ_WRITE}, nil
}

func TestValidateStoreAndGetCapabilitiesUsesCapabilitiesOnly(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))

	server, address, tlsSecret := newClusterProviderGRPCServer(t)
	store := &esv1.ClusterProvider{
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
					Namespace:  "config-ns",
				},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(store, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "external-secrets-provider-tls",
				Namespace: "config-ns",
			},
			Data: tlsSecret,
		}).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}

	caps, err := r.validateStoreAndGetCapabilities(context.Background(), store)
	if err != nil {
		t.Fatalf("validateStoreAndGetCapabilities() error = %v", err)
	}
	if caps != esv1.ProviderReadWrite {
		t.Fatalf("expected ProviderReadWrite, got %q", caps)
	}
	if server.validateRequest != nil {
		t.Fatalf("expected Validate not to be called, got %#v", server.validateRequest)
	}
	assertClusterProviderReference(t, server.capabilitiesRequest.ProviderRef, store.Spec.Config.ProviderRef, esv1.ClusterProviderKindStr)
	if server.capabilitiesRequest.SourceNamespace != "" {
		t.Fatalf("expected empty source namespace, got %q", server.capabilitiesRequest.SourceNamespace)
	}
}

func TestValidateStoreAndGetCapabilitiesFallsBackToReadOnlyOnCapabilitiesError(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))

	server, address, tlsSecret := newClusterProviderGRPCServer(t)
	server.capabilitiesErr = status.Error(codes.Unavailable, "capabilities unavailable")

	store := &esv1.ClusterProvider{
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
					Namespace:  "config-ns",
				},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(store, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "external-secrets-provider-tls",
				Namespace: "config-ns",
			},
			Data: tlsSecret,
		}).
		Build()

	r := &Reconciler{Client: kubeClient, Log: logr.Discard()}

	caps, err := r.validateStoreAndGetCapabilities(context.Background(), store)
	if err != nil {
		t.Fatalf("expected fallback to read-only, got error %v", err)
	}
	if caps != esv1.ProviderReadOnly {
		t.Fatalf("expected ProviderReadOnly, got %q", caps)
	}
}

func TestReconcileValidationFailureClearsStaleCapabilitiesAndUpdatesCondition(t *testing.T) {
	previousMetrics := gaugeVecMetrics
	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ClusterProviderReconcileDurationKey: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Subsystem: ClusterProviderSubsystem,
			Name:      ClusterProviderReconcileDurationKey,
		}, []string{"name"}),
		StatusConditionKey: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Subsystem: ClusterProviderSubsystem,
			Name:      StatusConditionKey,
		}, []string{"name", "condition", "status"}),
	}
	t.Cleanup(func() {
		gaugeVecMetrics = previousMetrics
	})

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))

	server, address, tlsSecret := newClusterProviderGRPCServer(t)
	server.capabilitiesErr = status.Error(codes.InvalidArgument, "invalid configuration")

	store := &esv1.ClusterProvider{
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
					Namespace:  "config-ns",
				},
			},
		},
		Status: esv1.ProviderStatus{
			Capabilities: esv1.ProviderReadWrite,
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(store, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "external-secrets-provider-tls",
				Namespace: "config-ns",
			},
			Data: tlsSecret,
		}).
		WithStatusSubresource(store).
		Build()

	r := &Reconciler{
		Client:          kubeClient,
		Log:             logr.Discard(),
		RequeueInterval: 29 * time.Second,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Name: "cluster-provider"},
	})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.RequeueAfter != 29*time.Second {
		t.Fatalf("expected requeue interval, got %#v", result)
	}

	var updated esv1.ClusterProvider
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Name: "cluster-provider"}, &updated); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if updated.Status.Capabilities != esv1.ProviderReadOnly {
		t.Fatalf("expected fallback read-only capabilities, got %q", updated.Status.Capabilities)
	}
	if len(updated.Status.Conditions) != 1 {
		t.Fatalf("expected a single condition, got %#v", updated.Status.Conditions)
	}

	condition := updated.Status.Conditions[0]
	if condition.Type != esv1.ProviderReady || condition.Status != metav1.ConditionTrue {
		t.Fatalf("unexpected condition: %#v", condition)
	}
}

func TestReconcileHardValidationFailureClearsStaleCapabilitiesAndUpdatesCondition(t *testing.T) {
	previousMetrics := gaugeVecMetrics
	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ClusterProviderReconcileDurationKey: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Subsystem: ClusterProviderSubsystem,
			Name:      ClusterProviderReconcileDurationKey,
		}, []string{"name"}),
		StatusConditionKey: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Subsystem: ClusterProviderSubsystem,
			Name:      StatusConditionKey,
		}, []string{"name", "condition", "status"}),
	}
	t.Cleanup(func() {
		gaugeVecMetrics = previousMetrics
	})

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))

	store := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "backend",
					Namespace:  "config-ns",
				},
			},
		},
		Status: esv1.ProviderStatus{
			Capabilities: esv1.ProviderReadWrite,
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(store).
		WithStatusSubresource(store).
		Build()

	r := &Reconciler{
		Client:          kubeClient,
		Log:             logr.Discard(),
		RequeueInterval: 31 * time.Second,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Name: "cluster-provider"},
	})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.RequeueAfter != 31*time.Second {
		t.Fatalf("expected requeue interval, got %#v", result)
	}

	var updated esv1.ClusterProvider
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Name: "cluster-provider"}, &updated); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if updated.Status.Capabilities != "" {
		t.Fatalf("expected capabilities to be cleared, got %q", updated.Status.Capabilities)
	}
	if len(updated.Status.Conditions) != 1 {
		t.Fatalf("expected a single condition, got %#v", updated.Status.Conditions)
	}

	condition := updated.Status.Conditions[0]
	if condition.Type != esv1.ProviderReady || condition.Status != metav1.ConditionFalse {
		t.Fatalf("unexpected condition: %#v", condition)
	}
	if condition.Reason != "ValidationFailed" {
		t.Fatalf("unexpected condition reason: %q", condition.Reason)
	}
	if condition.Message != "provider address is required" {
		t.Fatalf("unexpected condition message: %q", condition.Message)
	}
}

func TestSetNotReadyConditionUpdatesReasonAndMessageWithoutChangingTransitionTime(t *testing.T) {
	previousMetrics := gaugeVecMetrics
	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		StatusConditionKey: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Subsystem: ClusterProviderSubsystem,
			Name:      StatusConditionKey,
		}, []string{"name", "condition", "status"}),
	}
	t.Cleanup(func() {
		gaugeVecMetrics = previousMetrics
	})

	previousTransition := metav1.NewTime(time.Unix(1700000000, 0))
	store := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
		Status: esv1.ProviderStatus{
			Conditions: []esv1.ProviderCondition{{
				Type:               esv1.ProviderReady,
				Status:             metav1.ConditionFalse,
				LastTransitionTime: previousTransition,
				Reason:             "OldReason",
				Message:            "old message",
			}},
		},
	}

	r := &Reconciler{Log: logr.Discard()}
	r.setNotReadyCondition(store, "ValidationFailed", "new message")

	if len(store.Status.Conditions) != 1 {
		t.Fatalf("expected a single condition, got %#v", store.Status.Conditions)
	}

	condition := store.Status.Conditions[0]
	if condition.Status != metav1.ConditionFalse {
		t.Fatalf("expected false status, got %q", condition.Status)
	}
	if condition.Reason != "ValidationFailed" {
		t.Fatalf("expected updated reason, got %q", condition.Reason)
	}
	if condition.Message != "new message" {
		t.Fatalf("expected updated message, got %q", condition.Message)
	}
	if !condition.LastTransitionTime.Equal(&previousTransition) {
		t.Fatalf("expected transition time to remain %v, got %v", previousTransition, condition.LastTransitionTime)
	}
}

func TestFindClusterProvidersForKubernetesProviderEnqueuesMatchingProviders(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&esv1.ClusterProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name: "match",
				},
				Spec: esv1.ClusterProviderSpec{
					Config: esv1.ProviderConfig{
						ProviderRef: esv1.ProviderReference{
							APIVersion: k8sv2alpha1.GroupVersion.String(),
							Kind:       k8sv2alpha1.Kind,
							Name:       "backend",
							Namespace:  "config-ns",
						},
					},
				},
			},
			&esv1.ClusterProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wrong-name",
				},
				Spec: esv1.ClusterProviderSpec{
					Config: esv1.ProviderConfig{
						ProviderRef: esv1.ProviderReference{
							APIVersion: k8sv2alpha1.GroupVersion.String(),
							Kind:       k8sv2alpha1.Kind,
							Name:       "other",
							Namespace:  "config-ns",
						},
					},
				},
			},
			&esv1.ClusterProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wrong-namespace",
				},
				Spec: esv1.ClusterProviderSpec{
					Config: esv1.ProviderConfig{
						ProviderRef: esv1.ProviderReference{
							APIVersion: k8sv2alpha1.GroupVersion.String(),
							Kind:       k8sv2alpha1.Kind,
							Name:       "backend",
							Namespace:  "other-ns",
						},
					},
				},
			},
		).
		Build()

	requests := findClusterProvidersForKubernetesProvider(context.Background(), kubeClient, &k8sv2alpha1.Kubernetes{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backend",
			Namespace: "config-ns",
		},
	})

	if len(requests) != 1 {
		t.Fatalf("expected one reconcile request, got %#v", requests)
	}
	if requests[0].NamespacedName != (client.ObjectKey{Name: "match"}) {
		t.Fatalf("unexpected reconcile request: %#v", requests[0])
	}
}

func newClusterProviderGRPCServer(t *testing.T) (*recordingClusterProviderGRPCServer, string, map[string][]byte) {
	t.Helper()

	serverCert, serverKey, clientCert, clientKey, caCert := newClusterProviderTLSArtifacts(t, "127.0.0.1")

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

	server := &recordingClusterProviderGRPCServer{}
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

func assertClusterProviderReference(t *testing.T, got *pb.ProviderReference, want esv1.ProviderReference, wantStoreRefKind string) {
	t.Helper()

	if got == nil {
		t.Fatal("expected provider reference to be set")
	}
	if got.ApiVersion != want.APIVersion || got.Kind != want.Kind || got.Name != want.Name || got.Namespace != want.Namespace {
		t.Fatalf("unexpected provider ref: got=%#v want=%#v", got, want)
	}
	if got.StoreRefKind != wantStoreRefKind {
		t.Fatalf("unexpected store ref kind: got=%q want=%q", got.StoreRefKind, wantStoreRefKind)
	}
}

func newClusterProviderTLSArtifacts(t *testing.T, host string) (serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM []byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "cluster-provider-controller-test-ca"},
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

	serverCertPEM, serverKeyPEM = newClusterProviderSignedTLSCert(t, caCert, caKey, 2, host, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
	clientCertPEM, clientKeyPEM = newClusterProviderSignedTLSCert(t, caCert, caKey, 3, host, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth})
	caCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	return serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM
}

func newClusterProviderSignedTLSCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey, serial int64, host string, usages []x509.ExtKeyUsage) ([]byte, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  usages,
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
