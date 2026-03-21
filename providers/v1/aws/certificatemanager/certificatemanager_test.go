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
	"strings"
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

type testCerts struct {
	LeafPEM         []byte
	IntermediatePEM []byte
	RootPEM         []byte
	PrivateKeyPEM   []byte
	TLSCrt          []byte
}

func generateTestCerts(t *testing.T) testCerts {
	t.Helper()

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

	tlsCrt := append(leafPEM, interPEM...)

	return testCerts{
		LeafPEM:         leafPEM,
		IntermediatePEM: interPEM,
		RootPEM:         rootPEM,
		PrivateKeyPEM:   privKeyPEM,
		TLSCrt:          tlsCrt,
	}
}

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

type remoteRef struct {
	remoteKey string
	property  string
}

func (r remoteRef) GetRemoteKey() string { return r.remoteKey }
func (r remoteRef) GetProperty() string  { return r.property }

func tlsSecret(tlsCrt, tlsKey []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-tls", Namespace: "default"},
		Data: map[string][]byte{
			"tls.crt": tlsCrt,
			"tls.key": tlsKey,
		},
	}
}

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

func clearARNCache() {
	arnCache.Range(func(key, _ any) bool {
		arnCache.Delete(key)
		return true
	})
}

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

func TestSplitCertificatePEM_NoCertificates(t *testing.T) {
	_, _, err := splitCertificatePEM([]byte("not a certificate"))
	if err == nil {
		t.Fatal("expected error for input with no CERTIFICATE blocks")
	}
}

func TestSplitCertificatePEM_InvalidLeafCertificate(t *testing.T) {
	badPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("not valid DER")})
	_, _, err := splitCertificatePEM(badPEM)
	if err == nil {
		t.Fatal("expected error for invalid leaf certificate DER")
	}
	if got := err.Error(); !strings.Contains(got, "failed to parse leaf certificate") {
		t.Errorf("expected error to mention leaf certificate, got: %s", got)
	}
}

func TestPushSecret_NewCertificate(t *testing.T) {
	clearARNCache()
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/new"

	var gotCert, gotChain []byte
	var gotTags []types.Tag
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
			gotTags = p.Tags
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
	if len(gotTags) != 3 {
		t.Fatalf("expected 3 management tags on new import, got %d", len(gotTags))
	}
	tagMap := make(map[string]string, len(gotTags))
	for _, tag := range gotTags {
		tagMap[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	if tagMap[managedBy] != externalSecrets {
		t.Errorf("expected tag %q=%q, got %q", managedBy, externalSecrets, tagMap[managedBy])
	}
	if tagMap[remoteKeyTag] != "my-cert" {
		t.Errorf("expected tag %q=%q, got %q", remoteKeyTag, "my-cert", tagMap[remoteKeyTag])
	}
	expectedHash := computeContentHash(certs.TLSCrt, certs.PrivateKeyPEM)
	if tagMap[contentHashTag] != expectedHash {
		t.Errorf("expected tag %q=%q, got %q", contentHashTag, expectedHash, tagMap[contentHashTag])
	}
}

func TestPushSecret_LeafOnly_NoChainSent(t *testing.T) {
	clearARNCache()
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
	secret := tlsSecret(certs.LeafPEM, certs.PrivateKeyPEM)

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("PushSecret: %v", err)
	}
	if len(gotChain) != 0 {
		t.Error("CertificateChain should be nil when tls.crt contains only the leaf")
	}
}

func TestPushSecret_ReimportExisting(t *testing.T) {
	clearARNCache()
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
			tags := append(managedTags("my-cert"),
				types.Tag{Key: aws.String(contentHashTag), Value: aws.String("stale-hash")})
			return &acm.ListTagsForCertificateOutput{Tags: tags}, nil
		},
		importCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			importedARN = aws.ToString(p.CertificateArn)
			if len(p.Tags) > 0 {
				t.Error("Tags must not be set on re-import")
			}
			return &acm.ImportCertificateOutput{CertificateArn: p.CertificateArn}, nil
		},
		addTagsFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
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

func TestPushSecret_SkipsReimportWhenUnchanged(t *testing.T) {
	clearARNCache()
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/unchanged"

	currentHash := computeContentHash(certs.TLSCrt, certs.PrivateKeyPEM)
	importCalled := false

	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{
					{CertificateArn: aws.String(arn)},
				},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			tags := append(managedTags("my-cert"),
				types.Tag{Key: aws.String(contentHashTag), Value: aws.String(currentHash)})
			return &acm.ListTagsForCertificateOutput{Tags: tags}, nil
		},
		importCertificateFn: func(_ context.Context, _ *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			importCalled = true
			t.Error("ImportCertificate should not be called when content is unchanged")
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		addTagsFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
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
	if importCalled {
		t.Error("ImportCertificate was called despite unchanged content")
	}
}

func TestPushSecret_MissingCertKey(t *testing.T) {
	clearARNCache()
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: "my-cert"}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "bad-secret", Namespace: "default"},
		Data: map[string][]byte{
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
		secretKey: "tls.crt",
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

func TestPushSecret_CachePreventsDoubleImport(t *testing.T) {
	clearARNCache()
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/cached"

	importCount := 0
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
		importCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			importCount++
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		addTagsFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		listTagsFn: func(_ context.Context, p *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			if aws.ToString(p.CertificateArn) == arn {
				return &acm.ListTagsForCertificateOutput{Tags: managedTags("my-cert")}, nil
			}
			return &acm.ListTagsForCertificateOutput{}, nil
		},
		removeTagsFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: "my-cert"}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	provider := newProvider(fake)
	if err := provider.PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("first PushSecret: %v", err)
	}
	if importCount != 1 {
		t.Fatalf("expected 1 import call after first push, got %d", importCount)
	}

	// Cache should resolve the ARN despite ListCertificates returning empty.
	provider2 := newProvider(fake)
	if err := provider2.PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("second PushSecret: %v", err)
	}
	if importCount != 2 {
		t.Fatalf("expected 2 import calls total, got %d", importCount)
	}

	importCount = 0
	var secondImportHadARN bool
	fake.importCertificateFn = func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
		importCount++
		secondImportHadARN = p.CertificateArn != nil
		return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
	}
	provider3 := newProvider(fake)
	if err := provider3.PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("third PushSecret: %v", err)
	}
	if !secondImportHadARN {
		t.Error("expected re-import (CertificateArn set) on cached lookup, got new import")
	}
}

func TestPushSecret_CacheClearedOnDelete(t *testing.T) {
	clearARNCache()
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/to-delete-cache"

	importCount := 0
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{{CertificateArn: aws.String(arn)}},
			}, nil
		},
		importCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			importCount++
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
		deleteCertificateFn: func(_ context.Context, _ *acm.DeleteCertificateInput, _ ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
			return &acm.DeleteCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: "my-cert"}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("PushSecret: %v", err)
	}
	if _, ok := arnCache.Load("my-cert"); !ok {
		t.Fatal("expected ARN to be cached after PushSecret")
	}

	if err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "my-cert"}); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	if _, ok := arnCache.Load("my-cert"); ok {
		t.Error("expected ARN cache to be cleared after DeleteSecret")
	}
}

func TestSecretExists_Found(t *testing.T) {
	clearARNCache()
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
	clearARNCache()
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

func TestDeleteSecret_Managed(t *testing.T) {
	clearARNCache()
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
	clearARNCache()
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/not-ours"

	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{{CertificateArn: aws.String(arn)}},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags("other-cert")}, nil
		},
	}

	if err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "not-ours"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSecret_NotFound(t *testing.T) {
	clearARNCache()
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
	clearARNCache()
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/external"

	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{{CertificateArn: aws.String(arn)}},
			}, nil
		},
		listTagsFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: []types.Tag{
				{Key: aws.String(remoteKeyTag), Value: aws.String("ext-cert")},
			}}, nil
		},
	}

	if err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "ext-cert"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSecret_DeletedBetweenFindAndListTags(t *testing.T) {
	clearARNCache()
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/race"

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
				return &acm.ListTagsForCertificateOutput{Tags: managedTags("my-cert")}, nil
			}
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

type smithyFakeNotFound struct{}

func (e *smithyFakeNotFound) Error() string                  { return "ResourceNotFoundException" }
func (e *smithyFakeNotFound) ErrorCode() string              { return "ResourceNotFoundException" }
func (e *smithyFakeNotFound) ErrorMessage() string           { return "certificate not found" }
func (e *smithyFakeNotFound) ErrorFault() smithy.ErrorFault  { return smithy.FaultClient }

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

