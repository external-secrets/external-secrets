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

// Package certificatemanager implements the AWS Certificate Manager provider for external-secrets.
package certificatemanager

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"

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

// PushSecretMetadataSpec contains metadata for pushing certificates to ACM.
type PushSecretMetadataSpec struct {
	Tags map[string]string `json:"tags,omitempty"`
}

const (
	managedBy           = "managed-by"
	externalSecrets     = "external-secrets"
	remoteKeyTag        = "external-secrets-remote-key"
	contentHashTag      = "external-secrets-content-hash"
	tlsCertKey          = "tls.crt"
	tlsPrivateKeyKey    = "tls.key"
	errNotImplemented   = "operation not supported by AWS Certificate Manager provider"
	errNotManagedByESO  = "certificate not managed by external-secrets"
	errSecretKeyNotEmpty = "secretKey must be empty for the AWS Certificate Manager provider: " +
		"the whole kubernetes.io/tls secret is required (tls.crt and tls.key)"
)

var (
	errCertificateNotFound = errors.New("certificate not found")

	// arnCache mitigates eventual consistency of ListCertificates by caching
	// remoteKey → ARN after a successful import.
	arnCache sync.Map
)

// ACMInterface is a subset of the ACM API.
type ACMInterface interface {
	ImportCertificate(ctx context.Context, params *acm.ImportCertificateInput, optFns ...func(*acm.Options)) (*acm.ImportCertificateOutput, error)
	DeleteCertificate(ctx context.Context, params *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error)
	ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error)
	AddTagsToCertificate(ctx context.Context, params *acm.AddTagsToCertificateInput, optFns ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error)
	ListTagsForCertificate(ctx context.Context, params *acm.ListTagsForCertificateInput, optFns ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error)
	RemoveTagsFromCertificate(ctx context.Context, params *acm.RemoveTagsFromCertificateInput, optFns ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error)
}

// CertificateManager is a provider for AWS ACM.
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

// PushSecret imports a kubernetes.io/tls secret into AWS Certificate Manager.
func (cm *CertificateManager) PushSecret(ctx context.Context, secret *corev1.Secret, psd esv1.PushSecretData) error {
	if psd.GetSecretKey() != "" {
		return errors.New(errSecretKeyNotEmpty)
	}

	meta, err := constructMetadata(psd.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse push secret metadata: %w", err)
	}

	certPEM, ok := secret.Data[tlsCertKey]
	if !ok || len(certPEM) == 0 {
		return fmt.Errorf("key %q not found or empty in secret %s/%s", tlsCertKey, secret.Namespace, secret.Name)
	}
	privKeyPEM, ok := secret.Data[tlsPrivateKeyKey]
	if !ok || len(privKeyPEM) == 0 {
		return fmt.Errorf("key %q not found or empty in secret %s/%s", tlsPrivateKeyKey, secret.Namespace, secret.Name)
	}

	leafPEM, chainPEM, err := splitCertificatePEM(certPEM)
	if err != nil {
		return fmt.Errorf("failed to parse %q: %w", tlsCertKey, err)
	}

	remoteKey := psd.GetRemoteKey()
	contentHash := computeContentHash(certPEM, privKeyPEM)

	existingARN, err := cm.findCertificateARN(ctx, remoteKey)
	if err != nil {
		return fmt.Errorf("failed to search for existing certificate: %w", err)
	}

	if existingARN != "" {
		tags, err := cm.listTags(ctx, existingARN)
		if err != nil {
			return fmt.Errorf("failed to list tags for %s: %w", existingARN, err)
		}

		if getTagValue(tags, contentHashTag) == contentHash {
			log.Info("certificate unchanged, skipping re-import", "arn", existingARN, "remoteKey", remoteKey)
			if err := cm.syncTags(ctx, existingARN, meta.Spec.Tags); err != nil {
				return fmt.Errorf("failed to sync certificate tags: %w", err)
			}
			return nil
		}

		input := &acm.ImportCertificateInput{
			Certificate:    leafPEM,
			PrivateKey:     privKeyPEM,
			CertificateArn: aws.String(existingARN),
		}
		if len(chainPEM) > 0 {
			input.CertificateChain = chainPEM
		}
		log.Info("re-importing existing ACM certificate", "arn", existingARN, "remoteKey", remoteKey)

		_, err = cm.client.ImportCertificate(ctx, input)
		metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMImportCertificate, err)
		if err != nil {
			return fmt.Errorf("failed to import certificate: %w", err)
		}

		if err := cm.syncTags(ctx, existingARN, meta.Spec.Tags); err != nil {
			return fmt.Errorf("failed to sync certificate tags: %w", err)
		}
		return cm.updateContentHash(ctx, existingARN, contentHash)
	}

	// Include management tags atomically with the first import.
	input := &acm.ImportCertificateInput{
		Certificate: leafPEM,
		PrivateKey:  privKeyPEM,
		Tags: []types.Tag{
			{Key: aws.String(managedBy), Value: aws.String(externalSecrets)},
			{Key: aws.String(remoteKeyTag), Value: aws.String(remoteKey)},
			{Key: aws.String(contentHashTag), Value: aws.String(contentHash)},
		},
	}
	if len(chainPEM) > 0 {
		input.CertificateChain = chainPEM
	}
	log.Info("importing new ACM certificate", "remoteKey", remoteKey)

	out, err := cm.client.ImportCertificate(ctx, input)
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMImportCertificate, err)
	if err != nil {
		return fmt.Errorf("failed to import certificate: %w", err)
	}

	certARN := aws.ToString(out.CertificateArn)
	arnCache.Store(remoteKey, certARN)

	if err := cm.syncTags(ctx, certARN, meta.Spec.Tags); err != nil {
		return fmt.Errorf("failed to sync certificate tags: %w", err)
	}

	return nil
}

// SecretExists checks if a certificate with the given remoteKey exists in ACM.
func (cm *CertificateManager) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	arn, err := cm.findCertificateARN(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return false, err
	}
	return arn != "", nil
}

// DeleteSecret deletes the ACM certificate identified by remoteKey.
func (cm *CertificateManager) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	arn, err := cm.findCertificateARN(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return fmt.Errorf("failed to search for certificate to delete: %w", err)
	}
	if arn == "" {
		return nil
	}

	tags, err := cm.listTags(ctx, arn)
	if errors.Is(err, errCertificateNotFound) {
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
	arnCache.Delete(remoteRef.GetRemoteKey())
	log.Info("deleted ACM certificate", "arn", arn, "remoteKey", remoteRef.GetRemoteKey())
	return nil
}

func (cm *CertificateManager) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, errors.New(errNotImplemented)
}

func (cm *CertificateManager) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

func (cm *CertificateManager) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// Close closes the provider.
func (cm *CertificateManager) Close(_ context.Context) error {
	return nil
}

// Validate checks if the provider is configured correctly.
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

func (cm *CertificateManager) findCertificateARN(ctx context.Context, remoteKey string) (string, error) {
	if cached, ok := arnCache.Load(remoteKey); ok {
		arn := cached.(string)
		tags, err := cm.listTags(ctx, arn)
		if err == nil && matchesTags(tags, remoteKey) {
			return arn, nil
		}
		if !errors.Is(err, errCertificateNotFound) && err != nil {
			return "", fmt.Errorf("failed to verify cached certificate %s: %w", arn, err)
		}
		arnCache.Delete(remoteKey)
	}

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
				continue
			}
			if err != nil {
				return "", fmt.Errorf("failed to list tags for %s: %w", aws.ToString(cert.CertificateArn), err)
			}
			if matchesTags(tags, remoteKey) {
				arnCache.Store(remoteKey, aws.ToString(cert.CertificateArn))
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

func (cm *CertificateManager) syncTags(ctx context.Context, arn string, desiredTags map[string]string) error {
	current, err := cm.listTags(ctx, arn)
	if err != nil {
		return err
	}

	currentMap := make(map[string]string, len(current))
	for _, t := range current {
		currentMap[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}

	var toRemove []types.Tag
	for k, v := range currentMap {
		if isReservedTag(k) {
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

	var toAdd []types.Tag
	for k, v := range desiredTags {
		if isReservedTag(k) {
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

func isManagedByESO(tags []types.Tag) bool {
	for _, t := range tags {
		if aws.ToString(t.Key) == managedBy && aws.ToString(t.Value) == externalSecrets {
			return true
		}
	}
	return false
}

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

func isReservedTag(key string) bool {
	return key == managedBy || key == remoteKeyTag || key == contentHashTag
}

func computeContentHash(certPEM, keyPEM []byte) string {
	h := sha256.New()
	h.Write(certPEM)
	h.Write(keyPEM)
	return hex.EncodeToString(h.Sum(nil))
}

func getTagValue(tags []types.Tag, key string) string {
	for _, t := range tags {
		if aws.ToString(t.Key) == key {
			return aws.ToString(t.Value)
		}
	}
	return ""
}

func (cm *CertificateManager) updateContentHash(ctx context.Context, arn, hash string) error {
	_, err := cm.client.AddTagsToCertificate(ctx, &acm.AddTagsToCertificateInput{
		CertificateArn: aws.String(arn),
		Tags:           []types.Tag{{Key: aws.String(contentHashTag), Value: aws.String(hash)}},
	})
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMAddTagsToCertificate, err)
	return err
}

// splitCertificatePEM splits a PEM bundle into the leaf certificate and the remaining chain.
func splitCertificatePEM(certPEM []byte) (leaf []byte, chain []byte, err error) {
	var blocks []*pem.Block
	rest := certPEM
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			blocks = append(blocks, block)
		}
	}
	if len(blocks) == 0 {
		return nil, nil, errors.New("no CERTIFICATE PEM blocks found")
	}

	if _, err := x509.ParseCertificate(blocks[0].Bytes); err != nil {
		return nil, nil, fmt.Errorf("failed to parse leaf certificate: %w", err)
	}

	leaf = pem.EncodeToMemory(blocks[0])

	for _, b := range blocks[1:] {
		chain = append(chain, pem.EncodeToMemory(b)...)
	}
	return leaf, chain, nil
}

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
