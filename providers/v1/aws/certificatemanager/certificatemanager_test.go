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
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	rgtTypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/aws/smithy-go"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/youmark/pkcs8"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	fakeacm "github.com/external-secrets/external-secrets/providers/v1/aws/certificatemanager/fake"
)

const (
	testRemoteKey  = "my-cert"
	pemPrivateKey  = "PRIVATE KEY"
	errPushSecret  = "PushSecret: %v"
	errGetSecret   = "GetSecret: %v"
	errExpectedTag = "expected tag %q=%q, got %q"
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

	leafKeyDER, _ := x509.MarshalPKCS8PrivateKey(leafKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{Type: pemPrivateKey, Bytes: leafKeyDER})

	tlsCrt := make([]byte, 0, len(leafPEM)+len(interPEM))
	tlsCrt = append(tlsCrt, leafPEM...)
	tlsCrt = append(tlsCrt, interPEM...)

	return testCerts{
		LeafPEM:         leafPEM,
		IntermediatePEM: interPEM,
		RootPEM:         rootPEM,
		PrivateKeyPEM:   privKeyPEM,
		TLSCrt:          tlsCrt,
	}
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

func newProvider(fc *fakeacm.Client) *CertificateManager {
	return &CertificateManager{
		client:       fc,
		rgtClient:    stubRgt(""),
		referentAuth: false,
		cfg:          &aws.Config{},
		arnCache:     expirable.NewLRU[string, string](arnCacheSize, nil, arnCacheTTL),
		exportCache:  expirable.NewLRU[string, exportCacheEntry](exportCacheSize, nil, exportCacheTTL),
	}
}

func stubRgt(certARN string) *fakeacm.RgtClient {
	return &fakeacm.RgtClient{
		GetResourcesFn: func(_ context.Context, _ *resourcegroupstaggingapi.GetResourcesInput, _ ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
			out := &resourcegroupstaggingapi.GetResourcesOutput{}
			if certARN != "" {
				out.ResourceTagMappingList = []rgtTypes.ResourceTagMapping{
					{ResourceARN: aws.String(certARN)},
				}
			}
			return out, nil
		},
	}
}

func exportableDescribe(serial string) fakeacm.DescribeCertificateFn {
	return func(_ context.Context, _ *acm.DescribeCertificateInput, _ ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error) {
		return &acm.DescribeCertificateOutput{
			Certificate: &types.CertificateDetail{
				Serial: aws.String(serial),
				Options: &types.CertificateOptions{
					Export: types.CertificateExportEnabled,
				},
			},
		}, nil
	}
}

func TestSplitCertificatePEM_LeafOnly(t *testing.T) {
	certs := generateTestCerts(t)

	leaf, chain, err := splitCertificatePEM(certs.LeafPEM)
	if err != nil {
		t.Fatalf("splitCertificatePEM: %v", err)
	}
	if !bytes.Equal(leaf, certs.LeafPEM) {
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
	if !bytes.Equal(leaf, certs.LeafPEM) {
		t.Error("leaf does not match first PEM block")
	}
	if !bytes.Equal(chain, certs.IntermediatePEM) {
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
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/new"

	var gotCert, gotChain []byte
	var gotTags []types.Tag
	fake := &fakeacm.Client{
		ImportCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			if p.CertificateArn != nil {
				t.Error("expected no CertificateArn on first import")
			}
			gotCert = p.Certificate
			gotChain = p.CertificateChain
			gotTags = p.Tags
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		AddTagsToCertificateFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags(testRemoteKey)}, nil
		},
		RemoveTagsFromCertificateFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: testRemoteKey}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf(errPushSecret, err)
	}
	if !bytes.Equal(gotCert, certs.LeafPEM) {
		t.Error("Certificate field should be the leaf certificate only")
	}
	if !bytes.Equal(gotChain, certs.IntermediatePEM) {
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
		t.Errorf(errExpectedTag, managedBy, externalSecrets, tagMap[managedBy])
	}
	if tagMap[remoteKeyTag] != testRemoteKey {
		t.Errorf(errExpectedTag, remoteKeyTag, testRemoteKey, tagMap[remoteKeyTag])
	}
	expectedHash := computeContentHash(certs.TLSCrt, certs.PrivateKeyPEM)
	if tagMap[contentHashTag] != expectedHash {
		t.Errorf(errExpectedTag, contentHashTag, expectedHash, tagMap[contentHashTag])
	}
}

func TestPushSecret_LeafOnly_NoChainSent(t *testing.T) {
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/leaf-only"

	var gotChain []byte
	fake := &fakeacm.Client{
		ImportCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			gotChain = p.CertificateChain
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		AddTagsToCertificateFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags(testRemoteKey)}, nil
		},
		RemoveTagsFromCertificateFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: testRemoteKey}
	secret := tlsSecret(certs.LeafPEM, certs.PrivateKeyPEM)

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf(errPushSecret, err)
	}
	if len(gotChain) != 0 {
		t.Error("CertificateChain should be nil when tls.crt contains only the leaf")
	}
}

func TestPushSecret_ReimportExisting(t *testing.T) {
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/existing"
	var importedARN string

	fake := &fakeacm.Client{
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			tags := append(managedTags(testRemoteKey),
				types.Tag{Key: aws.String(contentHashTag), Value: aws.String("stale-hash")})
			return &acm.ListTagsForCertificateOutput{Tags: tags}, nil
		},
		ImportCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			importedARN = aws.ToString(p.CertificateArn)
			if len(p.Tags) > 0 {
				t.Error("Tags must not be set on re-import")
			}
			return &acm.ImportCertificateOutput{CertificateArn: p.CertificateArn}, nil
		},
		AddTagsToCertificateFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		RemoveTagsFromCertificateFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: testRemoteKey}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	provider := newProvider(fake)
	provider.rgtClient = stubRgt(arn)
	if err := provider.PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf(errPushSecret, err)
	}
	if importedARN != arn {
		t.Errorf("expected CertificateArn %q, got %q", arn, importedARN)
	}
}

func TestPushSecret_SkipsReimportWhenUnchanged(t *testing.T) {
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/unchanged"

	currentHash := computeContentHash(certs.TLSCrt, certs.PrivateKeyPEM)
	importCalled := false

	fake := &fakeacm.Client{
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			tags := append(managedTags(testRemoteKey),
				types.Tag{Key: aws.String(contentHashTag), Value: aws.String(currentHash)})
			return &acm.ListTagsForCertificateOutput{Tags: tags}, nil
		},
		ImportCertificateFn: func(_ context.Context, _ *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			importCalled = true
			t.Error("ImportCertificate should not be called when content is unchanged")
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		AddTagsToCertificateFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		RemoveTagsFromCertificateFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: testRemoteKey}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	provider := newProvider(fake)
	provider.rgtClient = stubRgt(arn)
	if err := provider.PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf(errPushSecret, err)
	}
	if importCalled {
		t.Error("ImportCertificate was called despite unchanged content")
	}
}

func TestPushSecret_MissingCertKey(t *testing.T) {
	fake := &fakeacm.Client{}

	psd := &pushSecretData{remoteKey: testRemoteKey}
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
		remoteKey: testRemoteKey,
		secretKey: "tls.crt",
	}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	err := newProvider(&fakeacm.Client{}).PushSecret(context.Background(), secret, psd)
	if err == nil {
		t.Fatal("expected error when secretKey is non-empty")
	}
	if err.Error() != errSecretKeyNotEmpty {
		t.Errorf("expected error %q, got %q", errSecretKeyNotEmpty, err.Error())
	}
}

func TestPushSecret_CachePreventsDoubleImport(t *testing.T) {
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/cached"

	var currentHash string
	fake := &fakeacm.Client{
		ImportCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			for _, tag := range p.Tags {
				if aws.ToString(tag.Key) == contentHashTag {
					currentHash = aws.ToString(tag.Value)
				}
			}
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		AddTagsToCertificateFn: func(_ context.Context, p *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			for _, tag := range p.Tags {
				if aws.ToString(tag.Key) == contentHashTag {
					currentHash = aws.ToString(tag.Value)
				}
			}
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		ListTagsForCertificateFn: func(_ context.Context, p *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			if aws.ToString(p.CertificateArn) == arn {
				tags := append(managedTags(testRemoteKey),
					types.Tag{Key: aws.String(contentHashTag), Value: aws.String(currentHash)})
				return &acm.ListTagsForCertificateOutput{Tags: tags}, nil
			}
			return &acm.ListTagsForCertificateOutput{}, nil
		},
		RemoveTagsFromCertificateFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	tagLookupCount := 0
	fakeRgt := &fakeacm.RgtClient{
		GetResourcesFn: func(_ context.Context, _ *resourcegroupstaggingapi.GetResourcesInput, _ ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
			tagLookupCount++
			return &resourcegroupstaggingapi.GetResourcesOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: testRemoteKey}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	provider := newProvider(fake)
	provider.rgtClient = fakeRgt
	if err := provider.PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("first PushSecret: %v", err)
	}
	if tagLookupCount != 1 {
		t.Fatalf("expected GetResources to be called once on first push, got %d", tagLookupCount)
	}

	// Second push on the same provider: cached ARN must skip the lookup via RGT API.
	if err := provider.PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("second PushSecret: %v", err)
	}
	if tagLookupCount != 1 {
		t.Errorf("expected cached ARN to skip GetResources, got %d lookups", tagLookupCount)
	}
}

func TestPushSecret_CacheClearedOnDelete(t *testing.T) {
	certs := generateTestCerts(t)
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/to-delete-cache"

	importCount := 0
	fake := &fakeacm.Client{
		ImportCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			importCount++
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		AddTagsToCertificateFn: func(_ context.Context, _ *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags(testRemoteKey)}, nil
		},
		RemoveTagsFromCertificateFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
		DeleteCertificateFn: func(_ context.Context, _ *acm.DeleteCertificateInput, _ ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
			return &acm.DeleteCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: testRemoteKey}
	secret := tlsSecret(certs.TLSCrt, certs.PrivateKeyPEM)

	provider := newProvider(fake)
	provider.rgtClient = stubRgt(arn)
	if err := provider.PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf(errPushSecret, err)
	}
	if _, ok := provider.arnCache.Get(testRemoteKey); !ok {
		t.Fatal("expected ARN to be cached after PushSecret")
	}

	if err := provider.DeleteSecret(context.Background(), remoteRef{remoteKey: testRemoteKey}); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	if _, ok := provider.arnCache.Get(testRemoteKey); ok {
		t.Error("expected ARN cache to be cleared after DeleteSecret")
	}
}

func TestSecretExists_Found(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/abc"
	fake := &fakeacm.Client{
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags(testRemoteKey)}, nil
		},
	}

	provider := newProvider(fake)
	provider.rgtClient = stubRgt(arn)
	exists, err := provider.SecretExists(context.Background(), remoteRef{remoteKey: testRemoteKey})
	if err != nil {
		t.Fatalf("SecretExists: %v", err)
	}
	if !exists {
		t.Error("expected certificate to exist")
	}
}

func TestSecretExists_StaleCacheEvictedOnNotFound(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/stale"

	fake := &fakeacm.Client{
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return nil, &smithyFakeNotFound{}
		},
	}
	fakeRgt := &fakeacm.RgtClient{
		GetResourcesFn: func(_ context.Context, _ *resourcegroupstaggingapi.GetResourcesInput, _ ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
			t.Error("GetResources must not be called when the ARN is served from cache")
			return &resourcegroupstaggingapi.GetResourcesOutput{}, nil
		},
	}

	provider := newProvider(fake)
	provider.rgtClient = fakeRgt
	provider.arnCache.Add(testRemoteKey, arn)

	exists, err := provider.SecretExists(context.Background(), remoteRef{remoteKey: testRemoteKey})
	if err != nil {
		t.Fatalf("SecretExists: %v", err)
	}
	if exists {
		t.Error("expected SecretExists to report false when the cert is gone")
	}
	if _, ok := provider.arnCache.Get(testRemoteKey); ok {
		t.Error("expected stale cache entry to be evicted")
	}
}

func TestSearchCertificatesByTag_Duplicates(t *testing.T) {
	const (
		arnA = "arn:aws:acm:us-east-1:123456789012:certificate/aaa"
		arnB = "arn:aws:acm:us-east-1:123456789012:certificate/bbb"
		arnC = "arn:aws:acm:us-east-1:123456789012:certificate/ccc"
	)

	call := 0
	fakeRgt := &fakeacm.RgtClient{
		GetResourcesFn: func(_ context.Context, in *resourcegroupstaggingapi.GetResourcesInput, _ ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
			call++
			switch call {
			case 1:
				if in.PaginationToken != nil {
					t.Errorf("expected no pagination token on first call, got %q", *in.PaginationToken)
				}
				next := "page2"
				return &resourcegroupstaggingapi.GetResourcesOutput{
					ResourceTagMappingList: []rgtTypes.ResourceTagMapping{
						{ResourceARN: aws.String(arnC)},
						{ResourceARN: aws.String(arnA)},
					},
					PaginationToken: &next,
				}, nil
			case 2:
				if in.PaginationToken == nil || *in.PaginationToken != "page2" {
					t.Errorf("expected pagination token %q on second call, got %v", "page2", in.PaginationToken)
				}
				return &resourcegroupstaggingapi.GetResourcesOutput{
					ResourceTagMappingList: []rgtTypes.ResourceTagMapping{
						{ResourceARN: aws.String(arnB)},
					},
				}, nil
			}
			t.Fatalf("unexpected extra GetResources call: %d", call)
			return nil, nil
		},
	}

	provider := newProvider(&fakeacm.Client{})
	provider.rgtClient = fakeRgt

	got, err := provider.searchCertificatesByTag(context.Background(), testRemoteKey)
	if err != nil {
		t.Fatalf("searchCertificatesByTag: %v", err)
	}
	if got != arnA {
		t.Errorf("expected lexically smallest ARN %q, got %q", arnA, got)
	}
	if call != 2 {
		t.Errorf("expected 2 GetResources calls (full pagination), got %d", call)
	}
}

func TestSecretExists_NotFound(t *testing.T) {
	exists, err := newProvider(&fakeacm.Client{}).SecretExists(context.Background(), remoteRef{remoteKey: "missing"})
	if err != nil {
		t.Fatalf("SecretExists: %v", err)
	}
	if exists {
		t.Error("expected certificate to not exist")
	}
}

func TestDeleteSecret_Managed(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/to-delete"
	deleted := false

	fake := &fakeacm.Client{
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: managedTags(testRemoteKey)}, nil
		},
		DeleteCertificateFn: func(_ context.Context, p *acm.DeleteCertificateInput, _ ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
			if aws.ToString(p.CertificateArn) != arn {
				t.Errorf("expected ARN %q, got %q", arn, aws.ToString(p.CertificateArn))
			}
			deleted = true
			return &acm.DeleteCertificateOutput{}, nil
		},
	}

	provider := newProvider(fake)
	provider.rgtClient = stubRgt(arn)
	if err := provider.DeleteSecret(context.Background(), remoteRef{remoteKey: testRemoteKey}); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	if !deleted {
		t.Error("expected DeleteCertificate to be called")
	}
}

func TestDeleteSecret_NotFound(t *testing.T) {
	if err := newProvider(&fakeacm.Client{}).DeleteSecret(context.Background(), remoteRef{remoteKey: "gone"}); err != nil {
		t.Fatalf("DeleteSecret on missing cert: %v", err)
	}
}

func TestDeleteSecret_NotManagedByESO(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/external"

	fake := &fakeacm.Client{
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return &acm.ListTagsForCertificateOutput{Tags: []types.Tag{
				{Key: aws.String(remoteKeyTag), Value: aws.String("ext-cert")},
			}}, nil
		},
	}

	provider := newProvider(fake)
	provider.rgtClient = stubRgt(arn)
	err := provider.DeleteSecret(context.Background(), remoteRef{remoteKey: "ext-cert"})
	if err == nil || err.Error() != errNotManagedByESO {
		t.Fatalf("expected %q, got: %v", errNotManagedByESO, err)
	}
}

func TestDeleteSecret_DeletedBetweenFindAndListTags(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/race"

	fake := &fakeacm.Client{
		ListTagsForCertificateFn: func(_ context.Context, _ *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			return nil, &smithyFakeNotFound{}
		},
		DeleteCertificateFn: func(_ context.Context, _ *acm.DeleteCertificateInput, _ ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error) {
			t.Error("DeleteCertificate should not be called when cert is already gone")
			return &acm.DeleteCertificateOutput{}, nil
		},
	}

	provider := newProvider(fake)
	provider.rgtClient = stubRgt(arn)
	if err := provider.DeleteSecret(context.Background(), remoteRef{remoteKey: testRemoteKey}); err != nil {
		t.Fatalf("expected no-op when cert disappears between find and verify, got: %v", err)
	}
}

type smithyFakeNotFound struct{}

func (e *smithyFakeNotFound) Error() string                 { return errResourceNotFound }
func (e *smithyFakeNotFound) ErrorCode() string             { return errResourceNotFound }
func (e *smithyFakeNotFound) ErrorMessage() string          { return "certificate not found" }
func (e *smithyFakeNotFound) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

func encryptPKCS8ForTest(t *testing.T, privateDER, passphrase []byte) []byte {
	t.Helper()

	privKey, err := x509.ParsePKCS8PrivateKey(privateDER)
	if err != nil {
		t.Fatalf("parse pkcs8: %v", err)
	}

	encryptedDER, err := pkcs8.MarshalPrivateKey(privKey, passphrase, nil)
	if err != nil {
		t.Fatalf("encrypt pkcs8: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: encryptedDER})
}

func TestGetSecret_ReturnsConcatenatedBundle(t *testing.T) {
	certs := generateTestCerts(t)

	pkcs8DER, _ := x509.MarshalPKCS8PrivateKey(func() *ecdsa.PrivateKey {
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		return key
	}())

	fake := &fakeacm.Client{
		DescribeCertificateFn: exportableDescribe("AA:BB:CC"),
		ExportCertificateFn: func(_ context.Context, p *acm.ExportCertificateInput, _ ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
			encPEM := encryptPKCS8ForTest(t, pkcs8DER, p.Passphrase)
			return &acm.ExportCertificateOutput{
				Certificate:      aws.String(string(certs.LeafPEM)),
				CertificateChain: aws.String(string(certs.IntermediatePEM)),
				PrivateKey:       aws.String(string(encPEM)),
			}, nil
		},
	}

	result, err := newProvider(fake).GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "arn:aws:acm:us-east-1:123456789012:certificate/test",
	})
	if err != nil {
		t.Fatalf(errGetSecret, err)
	}
	if !strings.Contains(string(result), "-----BEGIN CERTIFICATE-----") {
		t.Error("expected certificate in output")
	}
	if !strings.Contains(string(result), "-----BEGIN PRIVATE KEY-----") {
		t.Error("expected decrypted private key in output")
	}
	if strings.Contains(string(result), "ENCRYPTED") {
		t.Error("private key should be decrypted")
	}
}

func TestGetSecret_CertOnly(t *testing.T) {
	certs := generateTestCerts(t)

	fake := &fakeacm.Client{
		DescribeCertificateFn: exportableDescribe("DD:EE:FF"),
		ExportCertificateFn: func(_ context.Context, _ *acm.ExportCertificateInput, _ ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
			return &acm.ExportCertificateOutput{
				Certificate: aws.String(string(certs.LeafPEM)),
			}, nil
		},
	}

	result, err := newProvider(fake).GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "arn:aws:acm:us-east-1:123456789012:certificate/test",
	})
	if err != nil {
		t.Fatalf(errGetSecret, err)
	}
	if !bytes.Equal(result, certs.LeafPEM) {
		t.Error("expected only the leaf certificate")
	}
}

func TestGetSecret_EmptyARN(t *testing.T) {
	_, err := newProvider(&fakeacm.Client{}).GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{})
	if err == nil {
		t.Fatal("expected error for empty ARN")
	}
}

func TestGetSecret_NotExportable(t *testing.T) {
	exportCalled := false
	fake := &fakeacm.Client{
		DescribeCertificateFn: func(_ context.Context, _ *acm.DescribeCertificateInput, _ ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error) {
			return &acm.DescribeCertificateOutput{
				Certificate: &types.CertificateDetail{
					Serial: aws.String("11:22:33"),
					Options: &types.CertificateOptions{
						Export: types.CertificateExportDisabled,
					},
				},
			}, nil
		},
		ExportCertificateFn: func(_ context.Context, _ *acm.ExportCertificateInput, _ ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
			exportCalled = true
			return nil, &smithyFakeValidation{}
		},
	}

	_, err := newProvider(fake).GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "arn:aws:acm:us-east-1:123456789012:certificate/public",
	})
	if err == nil {
		t.Fatal("expected error for non-exportable certificate")
	}
	if !strings.Contains(err.Error(), "not exportable") {
		t.Errorf("expected not-exportable error message, got: %s", err.Error())
	}
	if exportCalled {
		t.Error("ExportCertificate should not be called for non-exportable certificates")
	}
}

func TestGetSecret_NotExportable_NilOptions(t *testing.T) {
	fake := &fakeacm.Client{
		DescribeCertificateFn: func(_ context.Context, _ *acm.DescribeCertificateInput, _ ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error) {
			return &acm.DescribeCertificateOutput{
				Certificate: &types.CertificateDetail{
					Serial: aws.String("44:55:66"),
				},
			}, nil
		},
	}

	_, err := newProvider(fake).GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "arn:aws:acm:us-east-1:123456789012:certificate/no-options",
	})
	if err == nil {
		t.Fatal("expected error when Options is nil")
	}
	if !strings.Contains(err.Error(), "not exportable") {
		t.Errorf("expected not-exportable error, got: %s", err.Error())
	}
}

func TestGetSecret_CacheHit(t *testing.T) {
	const certARN = "arn:aws:acm:us-east-1:123456789012:certificate/cached"
	cachedPEM := []byte("cached-pem-bundle")

	exportCalled := false
	fake := &fakeacm.Client{
		DescribeCertificateFn: exportableDescribe("AA:BB"),
		ExportCertificateFn: func(_ context.Context, _ *acm.ExportCertificateInput, _ ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
			exportCalled = true
			return nil, fmt.Errorf("should not be called")
		},
	}

	provider := newProvider(fake)
	provider.exportCache.Add(certARN, exportCacheEntry{serial: "AA:BB", pem: cachedPEM})

	result, err := provider.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: certARN,
	})
	if err != nil {
		t.Fatalf(errGetSecret, err)
	}
	if !bytes.Equal(result, cachedPEM) {
		t.Errorf("expected cached PEM, got %q", string(result))
	}
	if exportCalled {
		t.Error("ExportCertificate should not be called on cache hit")
	}
}

func TestGetSecret_CacheMissOnSerialChange(t *testing.T) {
	certs := generateTestCerts(t)
	const certARN = "arn:aws:acm:us-east-1:123456789012:certificate/renewed"

	fake := &fakeacm.Client{
		DescribeCertificateFn: exportableDescribe("NEW:SERIAL"),
		ExportCertificateFn: func(_ context.Context, _ *acm.ExportCertificateInput, _ ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
			return &acm.ExportCertificateOutput{
				Certificate: aws.String(string(certs.LeafPEM)),
			}, nil
		},
	}

	provider := newProvider(fake)
	provider.exportCache.Add(certARN, exportCacheEntry{serial: "OLD:SERIAL", pem: []byte("old-pem")})

	result, err := provider.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: certARN,
	})
	if err != nil {
		t.Fatalf(errGetSecret, err)
	}
	if !bytes.Equal(result, certs.LeafPEM) {
		t.Error("expected fresh export after serial change")
	}

	cached, ok := provider.exportCache.Get(certARN)
	if !ok {
		t.Fatal("expected cache to be updated after export")
	}
	if cached.serial != "NEW:SERIAL" {
		t.Error("cached serial should be updated")
	}
}

func TestDecryptPKCS8PEM_Roundtrip(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pkcs8DER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}

	passphrase := []byte("roundtrip-test-passphrase")
	encPEM := encryptPKCS8ForTest(t, pkcs8DER, passphrase)

	decPEM, err := decryptPKCS8PEM(encPEM, passphrase)
	if err != nil {
		t.Fatalf("decryptPKCS8PEM: %v", err)
	}

	block, _ := pem.Decode(decPEM)
	if block == nil || block.Type != pemPrivateKey {
		t.Fatalf("expected PRIVATE KEY PEM block, got %v", block)
	}
	if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
		t.Fatalf("decrypted key is not valid PKCS#8: %v", err)
	}
}

func TestDecryptPKCS8PEM_AlreadyUnencrypted(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pkcs8DER, _ := x509.MarshalPKCS8PrivateKey(key)
	unencPEM := pem.EncodeToMemory(&pem.Block{Type: pemPrivateKey, Bytes: pkcs8DER})

	result, err := decryptPKCS8PEM(unencPEM, []byte("unused"))
	if err != nil {
		t.Fatalf("decryptPKCS8PEM: %v", err)
	}
	if !bytes.Equal(result, unencPEM) {
		t.Error("unencrypted PEM should be returned as-is")
	}
}

type smithyFakeValidation struct{}

func (e *smithyFakeValidation) Error() string                 { return "ValidationException" }
func (e *smithyFakeValidation) ErrorCode() string             { return "ValidationException" }
func (e *smithyFakeValidation) ErrorMessage() string          { return "certificate is not exportable" }
func (e *smithyFakeValidation) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }
