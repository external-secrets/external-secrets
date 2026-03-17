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
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/smithy-go"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

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

// tlsSecret returns a minimal fake TLS secret.
func tlsSecret(cert, key, chain []byte) *corev1.Secret {
	data := map[string][]byte{
		"tls.crt": cert,
		"tls.key": key,
	}
	if chain != nil {
		data["ca.crt"] = chain
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-tls", Namespace: "default"},
		Data:       data,
	}
}

// managedTags returns the tag set that marks a certificate as managed by ESO with a given remoteKey.
func managedTags(remoteKey string) []types.Tag {
	return []types.Tag{
		{Key: aws.String(managedBy), Value: aws.String(externalSecrets)},
		{Key: aws.String(remoteKeyTag), Value: aws.String(remoteKey)},
	}
}

// buildMetadataJSON serialises a PushSecretMetadataSpec into the wire format expected by ParseMetadataParameters.
func buildMetadataJSON(t *testing.T, spec PushSecretMetadataSpec) *apiextensionsv1.JSON {
	t.Helper()
	wrapper := map[string]interface{}{
		"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
		"kind":       "PushSecretMetadata",
		"spec":       spec,
	}
	b, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("buildMetadataJSON: %v", err)
	}
	return &apiextensionsv1.JSON{Raw: b}
}

func newProvider(fake *fakeACMClient) *CertificateManager {
	return &CertificateManager{
		client:       fake,
		referentAuth: false,
		cfg:          &aws.Config{},
	}
}

// ---------------------------------------------------------------------------
// PushSecret tests
// ---------------------------------------------------------------------------

func TestPushSecret_NewCertificate(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/new"
	fake := &fakeACMClient{
		// No certificates exist yet.
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
		importCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			if p.CertificateArn != nil {
				t.Error("expected no CertificateArn on first import")
			}
			return &acm.ImportCertificateOutput{CertificateArn: aws.String(arn)}, nil
		},
		addTagsFn: func(_ context.Context, p *acm.AddTagsToCertificateInput, _ ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error) {
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
	secret := tlsSecret([]byte("CERT"), []byte("KEY"), nil)

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("PushSecret: %v", err)
	}
}

func TestPushSecret_ReimportExisting(t *testing.T) {
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
			// Should not be called on re-import.
			t.Error("AddTagsToCertificate should not be called when re-importing")
			return &acm.AddTagsToCertificateOutput{}, nil
		},
		removeTagsFn: func(_ context.Context, _ *acm.RemoveTagsFromCertificateInput, _ ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error) {
			return &acm.RemoveTagsFromCertificateOutput{}, nil
		},
	}

	psd := &pushSecretData{remoteKey: "my-cert"}
	secret := tlsSecret([]byte("CERT"), []byte("KEY"), []byte("CHAIN"))

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
			// tls.key is present but tls.crt is missing.
			"tls.key": []byte("KEY"),
		},
	}

	err := newProvider(fake).PushSecret(context.Background(), secret, psd)
	if err == nil {
		t.Fatal("expected error when tls.crt is missing")
	}
}

func TestPushSecret_CustomKeyNames(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/custom"
	var gotCert, gotKey []byte

	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
		importCertificateFn: func(_ context.Context, p *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
			gotCert = p.Certificate
			gotKey = p.PrivateKey
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

	psd := &pushSecretData{
		remoteKey: "my-cert",
		metadata: buildMetadataJSON(t, PushSecretMetadataSpec{
			CertificateKey: "my.cert",
			PrivateKeyKey:  "my.key",
		}),
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "custom-secret", Namespace: "default"},
		Data: map[string][]byte{
			"my.cert": []byte("CUSTOM-CERT"),
			"my.key":  []byte("CUSTOM-KEY"),
		},
	}

	if err := newProvider(fake).PushSecret(context.Background(), secret, psd); err != nil {
		t.Fatalf("PushSecret: %v", err)
	}
	if string(gotCert) != "CUSTOM-CERT" {
		t.Errorf("expected certificate CUSTOM-CERT, got %s", gotCert)
	}
	if string(gotKey) != "CUSTOM-KEY" {
		t.Errorf("expected key CUSTOM-KEY, got %s", gotKey)
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
	err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "not-ours"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSecret_NotFound(t *testing.T) {
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{}, nil
		},
	}

	// Should be a no-op.
	if err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "gone"}); err != nil {
		t.Fatalf("DeleteSecret on missing cert: %v", err)
	}
}

func TestDeleteSecret_ExplicitlyTaggedAsNotManaged(t *testing.T) {
	const arn = "arn:aws:acm:us-east-1:123456789012:certificate/external"

	// The certificate has the remoteKey tag but NOT the managed-by tag.
	fake := &fakeACMClient{
		listCertificatesFn: func(_ context.Context, _ *acm.ListCertificatesInput, _ ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
			return &acm.ListCertificatesOutput{
				CertificateSummaryList: []types.CertificateSummary{{CertificateArn: aws.String(arn)}},
			}, nil
		},
		listTagsFn: func(_ context.Context, p *acm.ListTagsForCertificateInput, _ ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error) {
			// Only the remoteKeyTag, no managed-by tag — this simulates a cert that was
			// not imported by ESO.
			return &acm.ListTagsForCertificateOutput{Tags: []types.Tag{
				{Key: aws.String(remoteKeyTag), Value: aws.String("ext-cert")},
			}}, nil
		},
	}

	err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "ext-cert"})
	// findCertificateARN won't match because matchesTags requires BOTH tags.
	// So the call should be a no-op (cert not found via matchesTags).
	if err != nil {
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

	err := newProvider(fake).DeleteSecret(context.Background(), remoteRef{remoteKey: "my-cert"})
	if err != nil {
		t.Fatalf("expected no-op when cert disappears between find and verify, got: %v", err)
	}
}

// smithyFakeNotFound is a minimal smithy.APIError stub for ResourceNotFoundException.
type smithyFakeNotFound struct{}

func (e *smithyFakeNotFound) Error() string         { return "ResourceNotFoundException" }
func (e *smithyFakeNotFound) ErrorCode() string     { return "ResourceNotFoundException" }
func (e *smithyFakeNotFound) ErrorMessage() string  { return "certificate not found" }
func (e *smithyFakeNotFound) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

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
