package certificatemanager

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// --- inline fake ACM client implementation ---
type fakeACM struct {
	descErr   error
	exportErr error
}

func (f *fakeACM) DescribeCertificate(ctx context.Context, in *acm.DescribeCertificateInput, _ ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error) {
	return &acm.DescribeCertificateOutput{}, f.descErr
}

func (f *fakeACM) ExportCertificate(ctx context.Context, in *acm.ExportCertificateInput, _ ...func(*acm.Options)) (*acm.ExportCertificateOutput, error) {
	if f.exportErr != nil {
		return nil, f.exportErr
	}
	return &acm.ExportCertificateOutput{
		Certificate:      aws.String("-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"),
		CertificateChain: aws.String("-----BEGIN CERTIFICATE-----\nCHAIN\n-----END CERTIFICATE-----"),
		PrivateKey:       aws.String("-----BEGIN PRIVATE KEY-----\nKEY\n-----END PRIVATE KEY-----"),
	}, nil
}

func (f *fakeACM) ListCertificates(ctx context.Context, in *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
	return &acm.ListCertificatesOutput{}, nil
}

func (f *fakeACM) GetCertificate(ctx context.Context, in *acm.GetCertificateInput, _ ...func(*acm.Options)) (*acm.GetCertificateOutput, error) {
	return &acm.GetCertificateOutput{}, nil
}

func (f *fakeACM) AddTagsToCertificate(ctx context.Context, in *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
	return &acm.AddTagsToCertificateOutput{}, nil
}

func (f *fakeACM) RemoveTagsFromCertificate(ctx context.Context, in *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
	return &acm.RemoveTagsFromCertificateOutput{}, nil
}

// --- helper to create provider instance with fake client ---
func newTestProvider() *CertificateManager {
	return &CertificateManager{
		client: &fakeACM{},
	}
}

// --- actual test ---
func TestGetSecret_Success(t *testing.T) {
	ctx := context.Background()
	cm := newTestProvider()

	out, err := cm.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
		Key: "arn:aws:acm:eu-central-1:123456789012:certificate/abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(out)
	for _, want := range []string{"CERT", "CHAIN", "KEY"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q: %s", want, got)
		}
	}
}
