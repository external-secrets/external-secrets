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
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"sort"
	"strings"
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
	ctrl "sigs.k8s.io/controller-runtime"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	awsutil "github.com/external-secrets/external-secrets/providers/v1/aws/util"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// PushSecretMetadataSpec contains metadata for pushing certificates to ACM.
type PushSecretMetadataSpec struct {
	Tags map[string]string `json:"tags,omitempty"`
}

const (
	managedBy            = "managed-by"
	externalSecrets      = "external-secrets"
	remoteKeyTag         = "external-secrets-remote-key"
	contentHashTag       = "external-secrets-content-hash"
	acmResourceType      = "acm:certificate"
	tlsCertKey           = "tls.crt"
	tlsPrivateKeyKey     = "tls.key"
	errNotImplemented    = "operation not supported by AWS Certificate Manager provider"
	errResourceNotFound  = "ResourceNotFoundException"
	errNotManagedByESO   = "certificate not managed by external-secrets"
	errSecretKeyNotEmpty = "secretKey must be empty for the AWS Certificate Manager provider: " +
		"the whole kubernetes.io/tls secret is required (tls.crt and tls.key)"
	errSyncTags      = "failed to sync certificate tags: %w"
	passphraseLength = 32
	// TTLs can be long because cached entries are always validated against
	// AWS API, which prevents serving stale data.
	arnCacheTTL     = 24 * time.Hour
	arnCacheSize    = 512
	exportCacheTTL  = 24 * time.Hour
	exportCacheSize = 128
)

type exportCacheEntry struct {
	serial string
	pem    []byte
}

type certBundle struct {
	leaf    []byte
	chain   []byte
	privKey []byte
}

var (
	errCertificateNotFound = errors.New("certificate not found")

	passphraseChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	// generatePassphrase returns a random alphanumeric passphrase safe for
	// the ACM ExportCertificate API (no #, $, or % characters).
	generatePassphrase = func() ([]byte, error) {
		b := make([]byte, passphraseLength)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		for i, v := range b {
			b[i] = passphraseChars[int(v)%len(passphraseChars)]
		}
		return b, nil
	}
)

// ACMInterface is a subset of the ACM API used by this provider.
// see: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/acm
type ACMInterface interface {
	ImportCertificate(ctx context.Context, input *acm.ImportCertificateInput, optFns ...func(*acm.Options)) (*acm.ImportCertificateOutput, error)
	DeleteCertificate(ctx context.Context, input *acm.DeleteCertificateInput, optFns ...func(*acm.Options)) (*acm.DeleteCertificateOutput, error)
	DescribeCertificate(ctx context.Context, input *acm.DescribeCertificateInput, optFns ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error)
	ExportCertificate(ctx context.Context, input *acm.ExportCertificateInput, optFns ...func(*acm.Options)) (*acm.ExportCertificateOutput, error)
	AddTagsToCertificate(ctx context.Context, input *acm.AddTagsToCertificateInput, optFns ...func(*acm.Options)) (*acm.AddTagsToCertificateOutput, error)
	ListTagsForCertificate(ctx context.Context, input *acm.ListTagsForCertificateInput, optFns ...func(*acm.Options)) (*acm.ListTagsForCertificateOutput, error)
	RemoveTagsFromCertificate(ctx context.Context, input *acm.RemoveTagsFromCertificateInput, optFns ...func(*acm.Options)) (*acm.RemoveTagsFromCertificateOutput, error)
}

// ResourceGetter is the subset of the Resource Groups Tagging API used to
// locate certificates by tag with server-side filtering.
// see: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi
type ResourceGetter interface {
	GetResources(ctx context.Context, input *resourcegroupstaggingapi.GetResourcesInput, optFns ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error)
}

// CertificateManager is a provider for AWS ACM.
type CertificateManager struct {
	cfg          *aws.Config
	client       ACMInterface
	rgtClient    ResourceGetter
	referentAuth bool
	prefix       string
	// arnCache mitigates eventual consistency of ListCertificates by caching
	// remoteKey → ARN after a successful import.
	arnCache *expirable.LRU[string, string]
	// exportCache caches ExportCertificate results keyed by ARN to avoid
	// repeated paid API calls.
	exportCache *expirable.LRU[string, exportCacheEntry]
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &CertificateManager{}

var log = ctrl.Log.WithName("provider").WithName("aws").WithName("certificatemanager")

// New constructs a CertificateManager Provider that is specific to a store.
func New(_ context.Context, cfg *aws.Config, prefix string, referentAuth bool) (*CertificateManager, error) {
	return &CertificateManager{
		cfg:          cfg,
		referentAuth: referentAuth,
		client: acm.NewFromConfig(*cfg, func(o *acm.Options) {
			o.EndpointResolverV2 = customEndpointResolver{}
		}),
		rgtClient: resourcegroupstaggingapi.NewFromConfig(*cfg, func(o *resourcegroupstaggingapi.Options) {
			o.EndpointResolverV2 = customTaggingEndpointResolver{}
		}),
		prefix:      prefix,
		arnCache:    expirable.NewLRU[string, string](arnCacheSize, nil, arnCacheTTL),
		exportCache: expirable.NewLRU[string, exportCacheEntry](exportCacheSize, nil, exportCacheTTL),
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

	leaf, chain, err := splitCertificatePEM(certPEM)
	if err != nil {
		return fmt.Errorf("failed to parse %q: %w", tlsCertKey, err)
	}

	bundle := certBundle{leaf: leaf, chain: chain, privKey: privKeyPEM}
	remoteKey := cm.prefix + psd.GetRemoteKey()
	contentHash := computeContentHash(certPEM, privKeyPEM)

	existingARN, err := cm.findCertificateARN(ctx, remoteKey)
	if err != nil {
		return fmt.Errorf("failed to search for existing certificate: %w", err)
	}

	if existingARN != "" {
		return cm.reimportCertificate(ctx, existingARN, bundle, contentHash, remoteKey, meta.Spec.Tags)
	}
	return cm.importNewCertificate(ctx, bundle, contentHash, remoteKey, meta.Spec.Tags)
}

func (cm *CertificateManager) reimportCertificate(ctx context.Context, arn string, b certBundle, contentHash, remoteKey string, tags map[string]string) error {
	currentTags, err := cm.listTags(ctx, arn)
	if errors.Is(err, errCertificateNotFound) {
		cm.arnCache.Remove(remoteKey)
		return cm.importNewCertificate(ctx, b, contentHash, remoteKey, tags)
	}
	if err != nil {
		return fmt.Errorf("failed to list tags for %s: %w", arn, err)
	}

	if getTagValue(currentTags, contentHashTag) == contentHash {
		log.Info("certificate unchanged, skipping re-import", "arn", arn, "remoteKey", remoteKey)
		if err := cm.syncTags(ctx, arn, tags); err != nil {
			return fmt.Errorf(errSyncTags, err)
		}
		return nil
	}

	input := &acm.ImportCertificateInput{
		Certificate:    b.leaf,
		PrivateKey:     b.privKey,
		CertificateArn: aws.String(arn),
	}
	if len(b.chain) > 0 {
		input.CertificateChain = b.chain
	}
	log.Info("re-importing existing ACM certificate", "arn", arn, "remoteKey", remoteKey)

	_, err = cm.client.ImportCertificate(ctx, input)
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMImportCertificate, err)
	if err != nil {
		return fmt.Errorf("failed to import certificate: %w", awsutil.SanitizeErr(err))
	}

	if err := cm.syncTags(ctx, arn, tags); err != nil {
		return fmt.Errorf(errSyncTags, err)
	}
	return cm.updateContentHash(ctx, arn, contentHash)
}

func (cm *CertificateManager) importNewCertificate(ctx context.Context, b certBundle, contentHash, remoteKey string, tags map[string]string) error {
	input := &acm.ImportCertificateInput{
		Certificate: b.leaf,
		PrivateKey:  b.privKey,
		Tags: []types.Tag{
			{Key: aws.String(managedBy), Value: aws.String(externalSecrets)},
			{Key: aws.String(remoteKeyTag), Value: aws.String(remoteKey)},
			{Key: aws.String(contentHashTag), Value: aws.String(contentHash)},
		},
	}
	if len(b.chain) > 0 {
		input.CertificateChain = b.chain
	}
	log.Info("importing new ACM certificate", "remoteKey", remoteKey)

	out, err := cm.client.ImportCertificate(ctx, input)
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMImportCertificate, err)
	if err != nil {
		return fmt.Errorf("failed to import certificate: %w", awsutil.SanitizeErr(err))
	}

	certARN := aws.ToString(out.CertificateArn)
	cm.arnCache.Add(remoteKey, certARN)

	if err := cm.syncTags(ctx, certARN, tags); err != nil {
		return fmt.Errorf(errSyncTags, err)
	}
	return nil
}

// SecretExists checks if a certificate with the given remoteKey exists in ACM.
func (cm *CertificateManager) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	remoteKey := cm.prefix + remoteRef.GetRemoteKey()
	arn, err := cm.findCertificateARN(ctx, remoteKey)
	if err != nil {
		return false, err
	}
	if arn == "" {
		return false, nil
	}
	if _, err := cm.listTags(ctx, arn); err != nil {
		if errors.Is(err, errCertificateNotFound) {
			cm.arnCache.Remove(remoteKey)
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteSecret deletes the ACM certificate identified by remoteKey.
func (cm *CertificateManager) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	remoteKey := cm.prefix + remoteRef.GetRemoteKey()
	arn, err := cm.findCertificateARN(ctx, remoteKey)
	if err != nil {
		return fmt.Errorf("failed to search for certificate to delete: %w", err)
	}
	if arn == "" {
		return nil
	}

	tags, err := cm.listTags(ctx, arn)
	if errors.Is(err, errCertificateNotFound) {
		cm.arnCache.Remove(remoteKey)
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
		return fmt.Errorf("failed to delete certificate %s: %w", arn, awsutil.SanitizeErr(err))
	}
	cm.arnCache.Remove(remoteKey)
	log.Info("deleted ACM certificate", "arn", arn, "remoteKey", remoteKey)
	return nil
}

// GetSecret returns a certificate from ACM as a concatenated PEM bundle.
// ref.Key must be the certificate ARN. The certificate must have export enabled.
func (cm *CertificateManager) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	certARN := ref.Key
	if certARN == "" {
		return nil, fmt.Errorf("certificate ARN must be specified in remoteRef.key")
	}

	descOut, err := cm.client.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
		CertificateArn: aws.String(certARN),
	})
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMDescribeCertificate, err)
	if err != nil {
		return nil, fmt.Errorf("failed to describe certificate %s: %w", certARN, awsutil.SanitizeErr(err))
	}

	detail := descOut.Certificate
	if detail.Options == nil || detail.Options.Export != types.CertificateExportEnabled {
		return nil, fmt.Errorf("certificate %s is not exportable (Options.Export is not ENABLED)", certARN)
	}

	// Use the serial number to detect certificate changes and avoid
	// redundant ExportCertificate calls (which are paid after 10k/month).
	serial := aws.ToString(detail.Serial)
	if entry, ok := cm.exportCache.Get(certARN); ok {
		if entry.serial == serial {
			return entry.pem, nil
		}
	}

	passphrase, err := generatePassphrase()
	if err != nil {
		return nil, fmt.Errorf("failed to generate passphrase: %w", err)
	}

	exportOut, err := cm.client.ExportCertificate(ctx, &acm.ExportCertificateInput{
		CertificateArn: aws.String(certARN),
		Passphrase:     passphrase,
	})
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMExportCertificate, err)
	if err != nil {
		return nil, fmt.Errorf("failed to export certificate %s: %w", certARN, awsutil.SanitizeErr(err))
	}

	var parts []string
	if exportOut.Certificate != nil {
		parts = append(parts, strings.TrimRight(*exportOut.Certificate, "\r\n"))
	}
	if exportOut.CertificateChain != nil {
		parts = append(parts, strings.TrimRight(*exportOut.CertificateChain, "\r\n"))
	}
	if exportOut.PrivateKey != nil {
		dk, dkErr := decryptPKCS8PEM([]byte(*exportOut.PrivateKey), passphrase)
		if dkErr != nil {
			return nil, fmt.Errorf("failed to decrypt exported private key: %w", dkErr)
		}
		parts = append(parts, strings.TrimRight(string(dk), "\r\n"))
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no data returned from ExportCertificate for %s", certARN)
	}
	result := strings.Join(parts, "\n") + "\n"

	cm.exportCache.Add(certARN, exportCacheEntry{serial: serial, pem: []byte(result)})
	return []byte(result), nil
}

// GetSecretMap is not supported by the ACM provider.
func (cm *CertificateManager) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// GetAllSecrets is not supported by the ACM provider.
func (cm *CertificateManager) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// Close is a no-op for the ACM provider.
func (cm *CertificateManager) Close(_ context.Context) error {
	return nil
}

// Validate checks if the provider is configured correctly.
func (cm *CertificateManager) Validate() (esv1.ValidationResult, error) {
	// skip validation stack because we do not have the client set with referent auth
	if cm.referentAuth {
		return esv1.ValidationResultUnknown, nil
	}
	_, err := cm.cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		return esv1.ValidationResultError, awsutil.SanitizeErr(err)
	}
	return esv1.ValidationResultReady, nil
}

func (cm *CertificateManager) findCertificateARN(ctx context.Context, remoteKey string) (string, error) {
	if cached, ok := cm.arnCache.Get(remoteKey); ok {
		return cached, nil
	}
	return cm.searchCertificatesByTag(ctx, remoteKey)
}

func (cm *CertificateManager) searchCertificatesByTag(ctx context.Context, remoteKey string) (string, error) {
	paginator := resourcegroupstaggingapi.NewGetResourcesPaginator(cm.rgtClient, &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: []string{acmResourceType},
		TagFilters: []rgtTypes.TagFilter{
			{Key: aws.String(managedBy), Values: []string{externalSecrets}},
			{Key: aws.String(remoteKeyTag), Values: []string{remoteKey}},
		},
	})

	var arns []string
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMGetResources, err)
		if err != nil {
			return "", awsutil.SanitizeErr(err)
		}
		for _, mapping := range out.ResourceTagMappingList {
			if certARN := aws.ToString(mapping.ResourceARN); certARN != "" {
				arns = append(arns, certARN)
			}
		}
	}

	if len(arns) == 0 {
		return "", nil
	}
	sort.Strings(arns)
	if len(arns) > 1 {
		log.Error(nil, "multiple certificates match the same remoteKey tag, using the lexically smallest",
			"remoteKey", remoteKey, "selected", arns[0], "matches", arns)
	}
	cm.arnCache.Add(remoteKey, arns[0])
	return arns[0], nil
}

func (cm *CertificateManager) listTags(ctx context.Context, arn string) ([]types.Tag, error) {
	out, err := cm.client.ListTagsForCertificate(ctx, &acm.ListTagsForCertificateInput{
		CertificateArn: aws.String(arn),
	})
	metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMListTagsForCertificate, err)
	if err != nil {
		var aerr smithy.APIError
		if errors.As(err, &aerr) && aerr.ErrorCode() == errResourceNotFound {
			return nil, errCertificateNotFound
		}
		return nil, awsutil.SanitizeErr(err)
	}
	return out.Tags, nil
}

func (cm *CertificateManager) syncTags(ctx context.Context, arn string, desiredTags map[string]string) error {
	current, err := cm.listTags(ctx, arn)
	if err != nil {
		return err
	}

	currentMap := tagsToMap(current)

	if toRemove := tagsToRemove(currentMap, desiredTags); len(toRemove) > 0 {
		_, err = cm.client.RemoveTagsFromCertificate(ctx, &acm.RemoveTagsFromCertificateInput{
			CertificateArn: aws.String(arn),
			Tags:           toRemove,
		})
		metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMRemoveTagsFromCertificate, err)
		if err != nil {
			return awsutil.SanitizeErr(err)
		}
	}

	if toAdd := tagsToAdd(currentMap, desiredTags); len(toAdd) > 0 {
		_, err = cm.client.AddTagsToCertificate(ctx, &acm.AddTagsToCertificateInput{
			CertificateArn: aws.String(arn),
			Tags:           toAdd,
		})
		metrics.ObserveAPICall(constants.ProviderAWSACM, constants.CallAWSACMAddTagsToCertificate, err)
		if err != nil {
			return awsutil.SanitizeErr(err)
		}
	}
	return nil
}

func tagsToMap(tags []types.Tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return m
}

func tagsToRemove(current, desired map[string]string) []types.Tag {
	var tags []types.Tag
	for k, v := range current {
		if isReservedTag(k) {
			continue
		}
		if _, ok := desired[k]; !ok {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
		}
	}
	return tags
}

func tagsToAdd(current, desired map[string]string) []types.Tag {
	var tags []types.Tag
	for k, v := range desired {
		if isReservedTag(k) {
			continue
		}
		if current[k] != v {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
		}
	}
	return tags
}

func isManagedByESO(tags []types.Tag) bool {
	for _, t := range tags {
		if aws.ToString(t.Key) == managedBy && aws.ToString(t.Value) == externalSecrets {
			return true
		}
	}
	return false
}

func isReservedTag(key string) bool {
	return key == managedBy || key == remoteKeyTag || key == contentHashTag
}

// computeContentHash returns a hex-encoded SHA-256 digest of the certificate
// and private key PEM. This is a content fingerprint used to detect changes
// and avoid redundant ImportCertificate API calls (1 rps rate limit). It is
// NOT used for any security or cryptographic purpose.
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
	if err != nil {
		return awsutil.SanitizeErr(err)
	}
	return nil
}

// splitCertificatePEM splits a PEM bundle into the leaf certificate and the remaining chain.
func splitCertificatePEM(certPEM []byte) (leaf, chain []byte, err error) {
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

// decryptPKCS8PEM decodes an "ENCRYPTED PRIVATE KEY" PEM block and returns the
// decrypted key as a "PRIVATE KEY" PEM block. If the PEM is already unencrypted
// it is returned as-is.
func decryptPKCS8PEM(encryptedPEM, passphrase []byte) ([]byte, error) {
	block, _ := pem.Decode(encryptedPEM)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	if block.Type == "PRIVATE KEY" {
		return encryptedPEM, nil
	}
	if block.Type != "ENCRYPTED PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected PEM type: %s", block.Type)
	}

	privateKey, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt PKCS#8 private key: %w", err)
	}

	derBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal decrypted private key: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	}), nil
}
