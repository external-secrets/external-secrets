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

package providerstore

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

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

func providerStoreScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))
	utilruntime.Must(esv2alpha1.AddToScheme(scheme))
	return scheme
}

func newProviderStoreReconciler(t *testing.T, objects ...client.Object) *StoreReconciler {
	t.Helper()

	builder := fakeclient.NewClientBuilder().
		WithScheme(providerStoreScheme(t)).
		WithStatusSubresource(&esv2alpha1.ProviderStore{}, &esv2alpha1.ClusterProviderStore{}, &esv1alpha1.ClusterProviderClass{}).
		WithObjects(objects...)

	return &StoreReconciler{
		Client:          builder.Build(),
		Log:             testr.New(t),
		Scheme:          providerStoreScheme(t),
		RequeueInterval: time.Minute,
	}
}

func newClusterProviderStoreReconciler(t *testing.T, objects ...client.Object) *ClusterStoreReconciler {
	t.Helper()

	builder := fakeclient.NewClientBuilder().
		WithScheme(providerStoreScheme(t)).
		WithStatusSubresource(&esv2alpha1.ProviderStore{}, &esv2alpha1.ClusterProviderStore{}, &esv1alpha1.ClusterProviderClass{}).
		WithObjects(objects...)

	return &ClusterStoreReconciler{
		Client:          builder.Build(),
		Log:             testr.New(t),
		Scheme:          providerStoreScheme(t),
		RequeueInterval: time.Minute,
	}
}

func providerStoreReady(status esv2alpha1.ProviderStoreStatus) bool {
	for _, condition := range status.Conditions {
		if condition.Type == esv2alpha1.ProviderStoreReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

type recordingProviderServer struct {
	pb.UnimplementedSecretStoreProviderServer
}

func newProviderStoreGRPCServer(t *testing.T) (*grpc.Server, string, map[string][]byte) {
	t.Helper()

	serverCert, serverKey, clientCert, clientKey, caCert := newMutualTLSArtifacts(t, "127.0.0.1")

	caPool := x509.NewCertPool()
	require.True(t, caPool.AppendCertsFromPEM(caCert))

	tlsCert, err := tls.X509KeyPair(serverCert, serverKey)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{tlsCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})))
	pb.RegisterSecretStoreProviderServer(grpcServer, &recordingProviderServer{})

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = lis.Close()
	})

	return grpcServer, lis.Addr().String(), map[string][]byte{
		"ca.crt":     caCert,
		"client.crt": clientCert,
		"client.key": clientKey,
	}
}

func (s *recordingProviderServer) Validate(_ context.Context, _ *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	return &pb.ValidateResponse{Valid: true}, nil
}

func newMutualTLSArtifacts(t *testing.T, host string) (serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM []byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-ca",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})

	serverCertPEM, serverKeyPEM = newSignedCert(t, caDER, caKeyPEM, host, true)
	clientCertPEM, clientKeyPEM = newSignedCert(t, caDER, caKeyPEM, "provider-client", false)

	return serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM
}

func newSignedCert(t *testing.T, caDER, caKeyPEM []byte, commonName string, isServer bool) ([]byte, []byte) {
	t.Helper()

	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if isServer {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		template.IPAddresses = []net.IP{net.ParseIP(commonName)}
	}

	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	caKeyBlock, _ := pem.Decode(caKeyPEM)
	require.NotNil(t, caKeyBlock)

	caKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	require.NoError(t, err)

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &certKey.PublicKey, caKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certKey)})

	return certPEM, keyPEM
}

func readyRuntimeClass(name string) *esv1alpha1.ClusterProviderClass {
	return &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: esv1alpha1.ClusterProviderClassStatus{
			Conditions: []metav1.Condition{{
				Type:   "Ready",
				Status: metav1.ConditionTrue,
				Reason: "Healthy",
			}},
		},
	}
}

func reconcileRequest(obj client.Object) ctrl.Request {
	return ctrl.Request{NamespacedName: client.ObjectKeyFromObject(obj)}
}

func TestFindProviderStoresForRuntimeClass(t *testing.T) {
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-runtime"},
	}

	reconciler := newProviderStoreReconciler(
		t,
		runtimeClass,
		&esv2alpha1.ProviderStore{
			ObjectMeta: metav1.ObjectMeta{Name: "implicit-kind", Namespace: "team-a"},
			Spec: esv2alpha1.ProviderStoreSpec{
				RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "shared-runtime"},
			},
		},
		&esv2alpha1.ProviderStore{
			ObjectMeta: metav1.ObjectMeta{Name: "explicit-kind", Namespace: "team-b"},
			Spec: esv2alpha1.ProviderStoreSpec{
				RuntimeRef: esv2alpha1.StoreRuntimeRef{
					Name: "shared-runtime",
					Kind: "ClusterProviderClass",
				},
			},
		},
		&esv2alpha1.ProviderStore{
			ObjectMeta: metav1.ObjectMeta{Name: "other-runtime", Namespace: "team-c"},
			Spec: esv2alpha1.ProviderStoreSpec{
				RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "other-runtime"},
			},
		},
	)

	requests := findProviderStoresForRuntimeClass(context.Background(), reconciler.Client, runtimeClass)

	assert.ElementsMatch(t, []ctrl.Request{
		{NamespacedName: client.ObjectKey{Name: "implicit-kind", Namespace: "team-a"}},
		{NamespacedName: client.ObjectKey{Name: "explicit-kind", Namespace: "team-b"}},
	}, requests)
}

func TestFindClusterProviderStoresForRuntimeClass(t *testing.T) {
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-runtime"},
	}

	reconciler := newClusterProviderStoreReconciler(
		t,
		runtimeClass,
		&esv2alpha1.ClusterProviderStore{
			ObjectMeta: metav1.ObjectMeta{Name: "implicit-kind"},
			Spec: esv2alpha1.ClusterProviderStoreSpec{
				RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "shared-runtime"},
			},
		},
		&esv2alpha1.ClusterProviderStore{
			ObjectMeta: metav1.ObjectMeta{Name: "explicit-kind"},
			Spec: esv2alpha1.ClusterProviderStoreSpec{
				RuntimeRef: esv2alpha1.StoreRuntimeRef{
					Name: "shared-runtime",
					Kind: "ClusterProviderClass",
				},
			},
		},
		&esv2alpha1.ClusterProviderStore{
			ObjectMeta: metav1.ObjectMeta{Name: "other-runtime"},
			Spec: esv2alpha1.ClusterProviderStoreSpec{
				RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "other-runtime"},
			},
		},
	)

	requests := findClusterProviderStoresForRuntimeClass(context.Background(), reconciler.Client, runtimeClass)

	assert.ElementsMatch(t, []ctrl.Request{
		{NamespacedName: client.ObjectKey{Name: "implicit-kind"}},
		{NamespacedName: client.ObjectKey{Name: "explicit-kind"}},
	}, requests)
}
