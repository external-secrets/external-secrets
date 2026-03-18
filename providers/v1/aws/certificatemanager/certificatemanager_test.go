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

package certificatemanager

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/smithy-go"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// ---------------------------------------------------------------------------
// Test certificate helpers
// ---------------------------------------------------------------------------

// testCerts holds PEM-encoded test certificate material.
type testCerts struct {
	// LeafPEM is the leaf certificate PEM block.
	LeafPEM []byte
	// IntermediatePEM is the intermediate CA certificate PEM block.
	IntermediatePEM []byte
	// RootPEM is the root CA certificate PEM block (self-signed).
	RootPEM []byte
	// PrivateKeyPEM is the leaf private key.
	PrivateKeyPEM []byte
	// TLSCrt is the standard kubernetes.io/tls tls.crt value:
	// leaf + intermediate concatenated (no root).
	TLSCrt []byte
}

// generateTestCerts creates a minimal three-tier PKI (root → intermediate → leaf) for testing.
func generateTestCerts(t *testing.T) testCerts {
	t.Helper()

	// Root CA (self-signed)
	rootKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	rootTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Root CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create root cert: %v", err)
	}
	rootCert, _ := x509.ParseCertificate(rootDER)

	// Intermediate CA (signed by root)
	interKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	interTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Test Intermediate CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		SubjectKeyId:          []byte{2},
		AuthorityKeyId:        rootCert.SubjectKeyId,
	}
	interDER, err := x509.CreateCertificate(rand.Reader, interTmpl, rootCert, &interKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create intermediate cert: %v", err)
	}
	interCert, _ := x509.ParseCertificate(interDER)

	// Leaf (signed by intermediate)
	leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	leafTmpl := &x509.Certificate{
		SerialNumber:   big.NewInt(3),
		Subject:        pkix.Name{CommonName: "test.example.com"},
		NotBefore:      time.Now().Add(-time.Hour),
		NotAfter:       time.Now().Add(24 * time.Hour),
		SubjectKeyId:   []byte{3},
		AuthorityKeyId: interCert.SubjectKeyId,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, interCert, &leafKey.PublicKey, interKey)
	if err != nil {
		t.Fatalf("create leaf cert: %v", err)
	}

	leafPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	interPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: interDER})
	rootPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER})

	leafKeyDER, _ := x509.MarshalECPrivateKey(leafKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})

	// tls.crt = leaf + intermediate (no root, matching cert-manager output)
	tlsCrt := append(leafPEM, interPEM...)

	return testCerts{
		LeafPEM:         leafPEM,
		IntermediatePEM: interPEM,
		RootPEM:         rootPEM,
		PrivateKeyPEM:   privKeyPEM,
		TLSCrt:          tlsCrt,
	}
}

// ---------------------------------------------------------------------------
// Fake ACM client and test helpers
// ---------------------------------------------------------------------------

// fakeACMClient is a test double for ACMInterface.
type fakeACMClient struct {
	importCertificateFn func(ctx context.Context, params *acm.ImportCertificateInput, optFns ...func(*acm.Options)) (*acm.ImportCertificateOutput, error)
	deleteCertificateFn func(ctx context.Context, params *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error)
	listCertificatesFn  func(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)
	addTagsFn           func(ctx context.Context, params *acm.AddTagsToCertificateInput, optFns ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error)
	listTagsFn          func(ctx context.Context, params *acm.ListTagsForCertificateInput, optFns ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error)
	removeTagsFn        func(ctx context.Context, params *acm.RemoveTagsFromCertificateInput, optFns ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error)
}

func (f *fakeACMClient) ImportCertificate(ctx context.Context, params *acm.ImportCertificateInput, optFns ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
	return f.importCertificateFn(ctx, params, optFns...)
}
func (f *fakeACMClient) DeleteCertificate(ctx context.Context, params *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
	return f.deleteCertificateFn(ctx, params, optFns...)
}
func (f *fakeACMClient) ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
	return f.listCertificatesFn(ctx, params, optFns...)
}
func (f *fakeACMClient) AddTagsToCertificate(ctx context.Context, params *acm.AddTagsToCertificateInput, optFns ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
	return f.addTagsFn(ctx, params, optFns...)
}
func (f *fakeACMClient) ListTagsForCertificate(ctx context.Context, params *acm.ListTagsForCertificateInput, optFns ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
	return f.listTagsFn(ctx, params, optFns...)
}
func (f *fakeACMClient) RemoveTagsFromCertificate(ctx context.Context, params *acm.RemoveTagsFromCertificateInput, optFns ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
	return f.removeTagsFn(ctx, params, optFns...)
}

// pushSecretData implements esv1.PushSecretData for testing.
type pushSecretData struct {
	remoteKey string
	secretKey string
	property  string
	metadata  *apiextensionsv1.JSON
}

func (p *pushSecretData) GetRemoteKey() string               { return p.remoteKey }
func (p *pushSecretData) GetSecretKey() string               { return p.secretKey }
func (p *pushSecretData) GetProperty() string                { return p.property }
func (p *pushSecretData) GetMetadata() *apiextensionsv1.JSON { return p.metadata }

// remoteRef implements esv1.PushSecretRemoteRef for testing.
type remoteRef struct {
	remoteKey string
	property  string
}

func (r remoteRef) GetRemoteKey() string { return r.remoteKey }
func (r remoteRef) GetProperty() string  { return r.property }

// tlsSecret builds a standard kubernetes.io/tls secret where tls.crt holds the
// provided PEM data (typically leaf + intermediate chain) and tls.key the private key.
func tlsSecret(tlsCrt, tlsKey []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-tls", Namespace: "default"},
		Data: map[string][]byte{
			"tls.crt": tlsCrt,
			"tls.key": tlsKey,
		},
	}
}

// managedTags returns the tag set that marks a certificate as managed by ESO with a given remoteKey.
func managedTags(remoteKey string) []types.Tag {
	return []types.Tag{
		{Key: aws.String(managedBy), Value: aws.String(externalSecrets)},
		{Key: aws.String(remoteKeyTag), Value: aws.String(remoteKey)},
	}
}

func newProvider(fake *fakeACMClient) *CertificateManager {
	return &CertificateManager{
		client:       fake,
		referentAuth: false,
		cfg:          &aws.Config{},
	}
}

// ---------------------------------------------------------------------------
// splitCertificatePEM tests
// ---------------------------------------------------------------------------

func TestSplitCertificatePEM_LeafOnly(t *testing.T) {
	certs := generateTestCerts(t)

	leaf, chain, err := splitCertificatePEM(certs.LeafPEM)
	if err != nil {
		t.Fatalf("splitCertificatePEM: %v", err)
	}
	if string(leaf) != string(certs.LeafPEM) {
		t.Error("leaf does not match input")
	}
	if len(chain) != 0 {
		t.Errorf("expected empty chain for leaf-only input, got %d bytes", len(chain))
	}
}

func TestSplitCertificatePEM_LeafAndIntermediate(t *testing.T) {
	certs := generateTestCerts(t)
	// tls.crt = leaf + intermediate (cert-manager format)
	leaf, chain, err := splitCertificatePEM(certs.TLSCrt)
	if err != nil {
		t.Fatalf("splitCertificatePEM: %v", err)
	}
	if string(leaf) != string(certs.LeafPEM) {
		t.Error("leaf does not match first PEM block")
	}
	if string(chain) != string(certs.IntermediatePEM) {
		t.Error("chain does not match intermediate PEM block")
	}
}

func TestSplitCertificatePEM_RootExcluded(t *testing.T) {
	certs := generateTestCerts(t)
	// tls.crt = leaf + intermediate + root (root must be stripped)
	withRoot := append(certs.TLSCrt, certs.RootPEM...)

	leaf, chain, err := splitCertificatePEM(withRoot)
	if err != nil {
		t.Fatalf("splitCertificatePEM: %v", err)
	}
	if string(leaf) != string(certs.LeafPEM) {
		t.Error("leaf does not match first PEM block")
	}
	// Root must not appear in chain.
	if string(chain) != string(certs.IntermediatePEM) {
		t.Errorf("chain should contain only intermediate, not root")
	}
}

func TestSplitCertificatePEM_NoCertificates(t *testing.T) {
	_, _, err := splitCertificatePEM([]byte("not a certificate"))
	if err == nil {
		t.Fatal("expected error for input with no CERTIFICATE blocks")
	}
}

// ---------------------------------------------------------------------------
// PushSecret tests
// ---------------------------------------------------------------------------

func TestPushSecret_NewCertificate(t *testing.T) {
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/new"

	var gotCert, gotChain []byte
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
		importCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			if p.CertificateArn != nil {
				t.Error("expected no CertificateArn on first import")
			}
			gotCert = p.Certificate
			gotChain = p.CertificateChain
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		addTagsFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags("my-cert")}, nil
		},
		removeTagsFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: "my-cert"}
	// tls.crt contains leaf + intermediate (standard cert-manager output)
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("PushSecret: %v", err)
	}
	if string(gotCert) != string(certs.LeafPEM) {
		t.Error("Certificate field should be the leaf certificate only")
	}
	if string(gotChain) != string(certs.IntermediatePEM) {
		t.Error("CertificateChain field should be the intermediate certificate")
	}
}

func TestPushSecret_LeafOnly_NoChainSent(t *testing.T) {
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/leaf-only"

	var gotChain []byte
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
		importCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			gotChain = p.CertificateChain
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		addTagsFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags("my-cert")}, nil
		},
		removeTagsFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: "my-cert"}
	// tls.crt contains only the leaf certificate — no chain should be sent to ACM.
	secret := tlsSecret(certs.LeafPEM, certs.PrivateKeyPEM)

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("PushSecret: %v", err)
	}
	if len(gotChain) != 0 {
		t.Error("CertificateChain should be nil when tls.crt contains only the leaf")
	}
}

func TestPushSecret_ReimportExisting(t *testing.T) {
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/existing"
	var importedARN string

	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{
					{CertificateArn: aws.String(arn)},
				},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags("my-cert")}, nil
		},
		importCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			importedARN = aws.ToString(p.CertificateArn)
			return &acm.ImportCertificateOutput{CertificateArn: p.CertificateArn}, nil
		},
		addTagsFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			t.Error("AddTagsToCertificate should not be called when re-importing")
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		removeTagsFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: "my-cert"}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("PushSecret: %v", err)
	}
	if importedARN != arn {
		t.Errorf("expected CertificateArn %q, got %q", arn, importedARN)
	}
}

func TestPushSecret_MissingCertKey(t *testing.T) {
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: "my-cert"}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "bad-secret", Namespace: "default"},
		Data: map[string][]byte{
			// tls.crt is missing, only tls.key present.
			"tls.key": []byte("KEY"),
		},
	}

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err == nil {
		t.Fatal("expected error when tls.crt is missing")
	}
}

func TestPushSecret_SecretKeyNotEmpty_Rejected(t *testing.T) {
	certs := generateTestCerts(t)

	psd := &pushSecretData{
		remoteKey: "my-cert",
		secretKey: "tls.crt", // must be empty for ACM
	}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	err := newProvider(&fakeACMClient{}).PushSecret(context.Background(), secret, psd)
	if err == nil {
		t.Fatal("expected error when secretKey is non-empty")
	}
	if err.Error() != errSecretKeyNotEmpty {
		t.Errorf("expected error %q, got %q", errSecretKeyNotEmpty, err.Error())
	}
}

// ---------------------------------------------------------------------------
// SecretExists tests
// ---------------------------------------------------------------------------

func TestSecretExists_Found(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{{CertificateArn: aws.String(arn)}},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags("my-cert")}, nil
		},
	}

	exists, err := newProvider(fake).SecretExists(context.Background(), remoteRef{remoteKey: "my-cert"})
	if err != nil {
		t.Fatalf("SecretExists: %v", err)
	}
	if !exists {
		t.Error("expected certificate to exist")
	}
}

func TestSecretExists_NotFound(t *testing.T) {
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
	}

	exists, err := newProvider(fake).SecretExists(context.Background(), remoteRef{remoteKey: "missing"})
	if err != nil {
		t.Fatalf("SecretExists: %v", err)
	}
	if exists {
		t.Error("expected certificate to not exist")
	}
}

// ---------------------------------------------------------------------------
// DeleteSecret tests
// ---------------------------------------------------------------------------

func TestDeleteSecret_Managed(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/to-delete"
	deleted := false

	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{{CertificateArn: aws.String(arn)}},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags("my-cert")}, nil
		},
		deleteCertificateFn: func(_ context.Context, p *acm.DeleteCertificateInput, _ ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
			if aws.ToString(p.CertificateArn) != arn {
				t.Errorf("expected ARN %q, got %q", arn, aws.ToString(p.CertificateArn))
			}
			deleted = true
			return &acm.DeleteCertificateOutput{}, nil
		},
	}

	if err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "my-cert"}); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	if !deleted {
		t.Error("expected DeleteCertificate to be called")
	}
}

func TestDeleteSecret_NotManagedByESO(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/not-ours"

	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{{CertificateArn: aws.String(arn)}},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			// matchesTags will return false for "not-ours" remoteKey since tag has "other-cert".
			return &acm.ListTagsForCertificateOutput{Tags: managedTags("other-cert")}, nil
		},
	}

	// "not-ours" doesn't match "other-cert" tag, so findCertificateARN returns "".
	if err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "not-ours"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSecret_NotFound(t *testing.T) {
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
	}

	if err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "gone"}); err != nil {
		t.Fatalf("DeleteSecret on missing cert: %v", err)
	}
}

func TestDeleteSecret_ExplicitlyTaggedAsNotManaged(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/external"

	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{{CertificateArn: aws.String(arn)}},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			// Only the remoteKeyTag, no managed-by tag — cert not imported by ESO.
			return &acm.ListTagsForCertificateOutput{Tags: []types.Tag{
				{Key: aws.String(remoteKeyTag), Value: aws.String("ext-cert")},
			}}, nil
		},
	}

	// findCertificateARN won't match because matchesTags requires BOTH tags, so no-op.
	if err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "ext-cert"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSecret_DeletedBetweenFindAndListTags(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/race"

	// findCertificateARN succeeds (cert appears in ListCertificates), but by the time
	// listTags is called for the management-check, the cert has already been deleted.
	callCount := 0
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{{CertificateArn: aws.String(arn)}},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			callCount++
			if callCount == 1 {
				// First call: inside findCertificateARN — cert still exists with matching tags.
				return &acm.ListTagsForCertificateOutput{Tags: managedTags("my-cert")}, nil
			}
			// Second call: inside DeleteSecret's management check — cert is now gone.
			return nil, &smithyFakeNotFound{}
		},
		deleteCertificateFn: func(_ context.Context, _ *acm.DeleteCertificateInput, _ ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
			t.Error("DeleteCertificate should not be called when cert is already gone")
			return &acm.DeleteCertificateOutput{}, nil
		},
	}

	if err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "my-cert"}); err != nil {
		t.Fatalf("expected no-op when cert disappears between find and verify, got: %v", err)
	}
}

// smithyFakeNotFound is a minimal smithy.APIError stub for ResourceNotFoundException.
type smithyFakeNotFound struct{}

func (e *smithyFakeNotFound) Error() string                  { return "ResourceNotFoundException" }
func (e *smithyFakeNotFound) ErrorCode() string              { return "ResourceNotFoundException" }
func (e *smithyFakeNotFound) ErrorMessage() string           { return "certificate not found" }
func (e *smithyFakeNotFound) ErrorFault() smithy.ErrorFault  { return smithy.FaultClient }

// ---------------------------------------------------------------------------
// Unsupported read operations
// ---------------------------------------------------------------------------

func TestGetSecret_Unsupported(t *testing.T) {
	cm := newProvider(&fakeACMClient{})
	_, err := cm.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != errNotImplemented {
		t.Errorf("expected error %q, got %q", errNotImplemented, err.Error())
	}
}

func TestCapabilities(t *testing.T) {
	cm := newProvider(&fakeACMClient{})
	if cm.Capabilities() != esv1.SecretStoreWriteOnly {
		t.Errorf("expected WriteOnly capability")
	}
}
