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

// Package certificatemanager implements an AWS Certificate Manager provider for External Secrets Operator.
// It supports importing TLS certificates stored in Kubernetes secrets into ACM via PushSecret.
package certificatemanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/smithy-go"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// PushSecretMetadataSpec contains metadata for pushing TLS certificates to AWS Certificate Manager.
type PushSecretMetadataSpec struct {
	// Tags are the AWS resource tags to apply to the certificate.
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
	// CertificateKey is the key in the Kubernetes secret containing the PEM-encoded certificate.
	// Defaults to "tls.crt".
	// +optional
	CertificateKey string `json:"certificateKey,omitempty"`
	// PrivateKeyKey is the key in the Kubernetes secret containing the PEM-encoded private key.
	// Defaults to "tls.key".
	// +optional
	PrivateKeyKey string `json:"privateKeyKey,omitempty"`
	// CertificateChainKey is the key in the Kubernetes secret containing the PEM-encoded certificate chain.
	// Defaults to "ca.crt". If the key is absent from the secret, no chain is sent to ACM.
	// +optional
	CertificateChainKey string `json:"certificateChainKey,omitempty"`
}

const (
	managedBy             = "managed-by"
	externalSecrets       = "external-secrets"
	remoteKeyTag          = "external-secrets-remote-key"
	defaultCertificateKey = "tls.crt"
	defaultPrivateKeyKey  = "tls.key"
	defaultCertChainKey   = "ca.crt"
	errNotImplemented     = "operation not supported by AWS Certificate Manager provider"
	errNotManagedByESO    = "certificate not managed by external-secrets"
)

// errCertificateNotFound is returned by listTags when the certificate no longer exists in ACM.
var errCertificateNotFound = errors.New("certificate not found")

// ACMInterface is the subset of the AWS ACM API used by this provider.
type ACMInterface interface {
	ImportCertificate(ctx context.Context, params *acm.ImportCertificateInput, optFns ...func(*acm.Options)) (*acm.ImportCertificateOutput, error)
	DeleteCertificate(ctx context.Context, params *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error)
	ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)
	AddTagsToCertificate(ctx context.Context, params *acm.AddTagsToCertificateInput, optFns ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error)
	ListTagsForCertificate(ctx context.Context, params *acm.ListTagsForCertificateInput, optFns ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error)
	RemoveTagsFromCertificate(ctx context.Context, params *acm.RemoveTagsFromCertificateInput, optFns ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error)
}

// CertificateManager is a provider for AWS Certificate Manager.
type CertificateManager struct {
	client       ACMInterface
	referentAuth bool
	cfg          *aws.Config
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &CertificateManager{}

var log = ctrl.Log.WithName("provider").WithName("aws").WithName("certificatemanager")

// New creates a new CertificateManager client.
func New(_ context.Context, cfg *aws.Config, referentAuth bool) (*CertificateManager, error) {
	return &CertificateManager{
		client:       acm.NewFromConfig(*cfg),
		referentAuth: referentAuth,
		cfg:          cfg,
	}, nil
}

// Capabilities returns WriteOnly since ACM is a certificate lifecycle service, not a general secret store.
func (cm *CertificateManager) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreWriteOnly
}

// PushSecret imports a TLS certificate from the Kubernetes secret into AWS Certificate Manager.
// The Kubernetes secret must contain the certificate and private key (and optionally a certificate chain).
// The remoteKey is used to tag the certificate for future lookups.
func (cm *CertificateManager) PushSecret(ctx context.Context, secret *corev1.Secret, psd esv1.PushSecretData) error {
	meta, err := constructMetadata(psd.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse push secret metadata: %w", err)
	}

	certKey := meta.Spec.CertificateKey
	if certKey == "" {
		certKey = defaultCertificateKey
	}
	privKeyKey := meta.Spec.PrivateKeyKey
	if privKeyKey == "" {
		privKeyKey = defaultPrivateKeyKey
	}
	chainKey := meta.Spec.CertificateChainKey
	if chainKey == "" {
		chainKey = defaultCertChainKey
	}

	certPEM, ok := secret.Data[certKey]
	if !ok || len(certPEM) == 0 {
		return fmt.Errorf("key %q not found or empty in secret %s/%s", certKey, secret.Namespace, secret.Name)
	}
	privKeyPEM, ok := secret.Data[privKeyKey]
	if !ok || len(privKeyPEM) == 0 {
		return fmt.Errorf("key %q not found or empty in secret %s/%s", privKeyKey, secret.Namespace, secret.Name)
	}
	chainPEM := secret.Data[chainKey] // optional

	remoteKey := psd.GetRemoteKey()

	// Find an existing certificate tagged with this remote key.
	existingARN, err := cm.findCertificateARN(ctx, remoteKey)
	if err != nil {
		return fmt.Errorf("failed to search for existing certificate: %w", err)
	}

	input := &acm.ImportCertificateInput{
		Certificate: certPEM,
		PrivateKey:  privKeyPEM,
	}
	if len(chainPEM) > 0 {
		input.CertificateChain = chainPEM
	}
	if existingARN != "" {
		// Re-import (update) the existing certificate in-place.
		input.CertificateArn = aws.String(existingARN)
		log.Info("re-importing existing ACM certificate", "arn", existingARN, "remoteKey", remoteKey)
	} else {
		log.Info("importing new ACM certificate", "remoteKey", remoteKey)
	}

	out, err := cm.client.ImportCertificate(ctx, input)
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMImportCertificate, err)
	if err != nil {
		return fmt.Errorf("failed to import certificate: %w", err)
	}

	certARN := aws.ToString(out.CertificateArn)

	// On first import, set the ESO management tags so we can track this certificate.
	if existingARN == "" {
		requiredTags := []types.Tag{
			{Key: aws.String(managedBy), Value: aws.String(externalSecrets)},
			{Key: aws.String(remoteKeyTag), Value: aws.String(remoteKey)},
		}
		_, err = cm.client.AddTagsToCertificate(ctx, &acm.AddTagsToCertificateInput{
			CertificateArn: aws.String(certARN),
			Tags:           requiredTags,
		})
		metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMAddTagsToCertificate, err)
		if err != nil {
			return fmt.Errorf("failed to tag imported certificate: %w", err)
		}
	}

	// Reconcile user-defined tags.
	if err := cm.syncTags(ctx, certARN, meta.Spec.Tags); err != nil {
		return fmt.Errorf("failed to sync certificate tags: %w", err)
	}

	return nil
}

// SecretExists returns true if an ACM certificate tagged with the given remoteKey exists.
func (cm *CertificateManager) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	arn, err := cm.findCertificateARN(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return false, err
	}
	return arn != "", nil
}

// DeleteSecret deletes the ACM certificate tagged with the given remoteKey, if it is managed by ESO.
func (cm *CertificateManager) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	arn, err := cm.findCertificateARN(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return fmt.Errorf("failed to search for certificate to delete: %w", err)
	}
	if arn == "" {
		// Already gone.
		return nil
	}

	// Verify it is managed by ESO before deleting.
	tags, err := cm.listTags(ctx, arn)
	if errors.Is(err, errCertificateNotFound) {
		// Deleted by another process between findCertificateARN and here — treat as no-op.
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to list tags for certificate %s: %w", arn, err)
	}
	if !isManagedByESO(tags) {
		return errors.New(errNotManagedByESO)
	}

	_, err = cm.client.DeleteCertificate(ctx, &acm.DeleteCertificateInput{
		CertificateArn: aws.String(arn),
	})
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMDeleteCertificate, err)
	if err != nil {
		return fmt.Errorf("failed to delete certificate %s: %w", arn, err)
	}
	log.Info("deleted ACM certificate", "arn", arn, "remoteKey", remoteRef.GetRemoteKey())
	return nil
}

// GetSecret is not supported by the Certificate Manager provider.
func (cm *CertificateManager) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// GetSecretMap is not supported by the Certificate Manager provider.
func (cm *CertificateManager) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// GetAllSecrets is not supported by the Certificate Manager provider.
func (cm *CertificateManager) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// Close is a no-op for the Certificate Manager provider.
func (cm *CertificateManager) Close(_ context.Context) error {
	return nil
}

// Validate validates that the provider credentials are reachable.
func (cm *CertificateManager) Validate() (esv1.ValidationResult, error) {
	if cm.referentAuth {
		return esv1.ValidationResultUnknown, nil
	}
	_, err := cm.cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		return esv1.ValidationResultError, err
	}
	return esv1.ValidationResultReady, nil
}

// findCertificateARN searches ACM for a certificate tagged with the given remoteKey.
// Returns the ARN of the matching certificate, or an empty string if not found.
func (cm *CertificateManager) findCertificateARN(ctx context.Context, remoteKey string) (string, error) {
	var nextToken *string
	for {
		out, err := cm.client.ListCertificates(ctx, &acm.ListCertificatesInput{
			NextToken: nextToken,
		})
		metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMListCertificates, err)
		if err != nil {
			return "", err
		}

		for _, cert := range out.CertificateSummaryList {
			if cert.CertificateArn == nil {
				continue
			}
			tags, err := cm.listTags(ctx, aws.ToString(cert.CertificateArn))
			if errors.Is(err, errCertificateNotFound) {
				// Deleted concurrently — skip and keep scanning.
				continue
			}
			if err != nil {
				return "", fmt.Errorf("failed to list tags for %s: %w", aws.ToString(cert.CertificateArn), err)
			}
			if matchesTags(tags, remoteKey) {
				return aws.ToString(cert.CertificateArn), nil
			}
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return "", nil
}

// listTags returns the ACM tags for the given certificate ARN.
// It returns errCertificateNotFound if the certificate no longer exists.
func (cm *CertificateManager) listTags(ctx context.Context, arn string) ([]types.Tag, error) {
	out, err := cm.client.ListTagsForCertificate(ctx, &acm.ListTagsForCertificateInput{
		CertificateArn: aws.String(arn),
	})
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMListTagsForCertificate, err)
	if err != nil {
		var aerr smithy.APIError
		if errors.As(err, &aerr) && aerr.ErrorCode() == "ResourceNotFoundException" {
			return nil, errCertificateNotFound
		}
		return nil, err
	}
	return out.Tags, nil
}

// syncTags reconciles user-defined tags on the certificate, preserving ESO management tags.
func (cm *CertificateManager) syncTags(ctx context.Context, arn string, desiredTags map[string]string) error {
	current, err := cm.listTags(ctx, arn)
	if err != nil {
		return err
	}

	currentMap := make(map[string]string, len(current))
	for _, t := range current {
		currentMap[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}

	// Remove tags that are no longer desired, skipping ESO management tags.
	var toRemove []types.Tag
	for k, v := range currentMap {
		if k == managedBy || k == remoteKeyTag {
			continue
		}
		if _, ok := desiredTags[k]; !ok {
			toRemove = append(toRemove, types.Tag{Key: aws.String(k), Value: aws.String(v)})
		}
	}
	if len(toRemove) > 0 {
		_, err = cm.client.RemoveTagsFromCertificate(ctx, &acm.RemoveTagsFromCertificateInput{
			CertificateArn: aws.String(arn),
			Tags:           toRemove,
		})
		metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMRemoveTagsFromCertificate, err)
		if err != nil {
			return err
		}
	}

	// Add or update desired tags.
	var toAdd []types.Tag
	for k, v := range desiredTags {
		if k == managedBy || k == remoteKeyTag {
			continue
		}
		if currentMap[k] != v {
			toAdd = append(toAdd, types.Tag{Key: aws.String(k), Value: aws.String(v)})
		}
	}
	if len(toAdd) > 0 {
		_, err = cm.client.AddTagsToCertificate(ctx, &acm.AddTagsToCertificateInput{
			CertificateArn: aws.String(arn),
			Tags:           toAdd,
		})
		metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMAddTagsToCertificate, err)
		if err != nil {
			return err
		}
	}
	return nil
}

// isManagedByESO checks whether the certificate was originally imported by ESO.
func isManagedByESO(tags []types.Tag) bool {
	for _, t := range tags {
		if aws.ToString(t.Key) == managedBy && aws.ToString(t.Value) == externalSecrets {
			return true
		}
	}
	return false
}

// matchesTags returns true when the certificate carries both the ESO management tag and the matching remoteKey tag.
func matchesTags(tags []types.Tag, remoteKey string) bool {
	var hasManagedBy, hasRemoteKey bool
	for _, t := range tags {
		switch aws.ToString(t.Key) {
		case managedBy:
			if aws.ToString(t.Value) == externalSecrets {
				hasManagedBy = true
			}
		case remoteKeyTag:
			if aws.ToString(t.Value) == remoteKey {
				hasRemoteKey = true
			}
		}
	}
	return hasManagedBy && hasRemoteKey
}

// constructMetadata parses the PushSecretMetadata from raw JSON, returning safe defaults when absent.
func constructMetadata(data *apiextensionsv1.JSON) (*metadata.PushSecretMetadata[PushSecretMetadataSpec], error) {
	meta, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](data)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		meta = &metadata.PushSecretMetadata[PushSecretMetadataSpec]{}
	}
	return meta, nil
}
