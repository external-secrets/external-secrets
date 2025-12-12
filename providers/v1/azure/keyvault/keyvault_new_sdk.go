/*
Copyright © 2025 ESO Maintainer Team

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

package keyvault

import (
	"context"
	"crypto/sha3"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// New SDK implementations for setter methods.
func (a *Azure) setKeyVaultSecretWithNewSDK(ctx context.Context, secretName string, value []byte, _ *time.Time, tags map[string]string) error {
	// Check if secret exists and if we can create/update it
	existingSecret, err := a.secretsClient.GetSecret(ctx, secretName, "", nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)

	if err != nil {
		var respErr *azcore.ResponseError
		if !errors.As(err, &respErr) || respErr.StatusCode != 404 {
			return fmt.Errorf("cannot get secret %v: %w", secretName, parseNewSDKError(err))
		}
	} else {
		// Check if managed by external-secrets using new SDK tags
		if existingSecret.Tags != nil {
			if managedByTag, exists := existingSecret.Tags[managedBy]; !exists || managedByTag == nil || *managedByTag != managerLabel {
				return fmt.Errorf("secret %v not managed by external-secrets", secretName)
			}
		}

		// Check if secret content is the same
		val := string(value)
		if existingSecret.Value != nil && val == *existingSecret.Value {
			// Note: We're not checking expiration here since the new SDK doesn't support setting it
			// This means the new SDK implementation will always update the secret if the content is the same
			// but different expiration is requested
			return nil
		}
	}

	// Prepare tags for new SDK
	secretTags := map[string]*string{
		managedBy: to.Ptr(managerLabel),
	}
	for k, v := range tags {
		secretTags[k] = &v
	}

	// Set the secret
	val := string(value)
	params := azsecrets.SetSecretParameters{
		Value: &val,
		Tags:  secretTags,
	}

	// Note: The new SDK doesn't support setting expiration in SetSecretParameters
	// This is a limitation compared to the legacy SDK - expiration would need to be handled differently

	_, err = a.secretsClient.SetSecret(ctx, secretName, params, nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVSetSecret, err)
	if err != nil {
		return fmt.Errorf("could not set secret %v: %w", secretName, parseNewSDKError(err))
	}
	return nil
}

func (a *Azure) setKeyVaultCertificateWithNewSDK(ctx context.Context, secretName string, value []byte, tags map[string]string) error {
	val := b64.StdEncoding.EncodeToString(value)
	localCert, err := getCertificateFromValue(value)
	if err != nil {
		return fmt.Errorf("value from secret is not a valid certificate: %w", err)
	}

	// Check if certificate exists
	cert, err := a.certsClient.GetCertificate(ctx, secretName, "", nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetCertificate, err)

	if err != nil {
		var respErr *azcore.ResponseError
		if !errors.As(err, &respErr) || respErr.StatusCode != 404 {
			return fmt.Errorf("cannot get certificate %v: %w", secretName, parseNewSDKError(err))
		}
	} else {
		// Check if managed by external-secrets
		if cert.Tags != nil {
			if managedByTag, exists := cert.Tags[managedBy]; !exists || managedByTag == nil || *managedByTag != managerLabel {
				return fmt.Errorf("certificate %v not managed by external-secrets", secretName)
			}
		}

		// Check if certificate content is the same
		b512 := sha3.Sum512(localCert.Raw)
		if cert.CER != nil && b512 == sha3.Sum512(cert.CER) {
			return nil
		}
	}

	// Prepare tags for new SDK
	certTags := map[string]*string{
		managedBy: to.Ptr(managerLabel),
	}
	for k, v := range tags {
		certTags[k] = &v
	}

	params := azcertificates.ImportCertificateParameters{
		Base64EncodedCertificate: &val,
		Tags:                     certTags,
	}

	_, err = a.certsClient.ImportCertificate(ctx, secretName, params, nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVImportCertificate, err)
	if err != nil {
		return fmt.Errorf("could not import certificate %v: %w", secretName, parseNewSDKError(err))
	}
	return nil
}

func (a *Azure) setKeyVaultKeyWithNewSDK(ctx context.Context, secretName string, value []byte, tags map[string]string) error {
	key, err := getKeyFromValue(value)
	if err != nil {
		return fmt.Errorf("could not load private key %v: %w", secretName, err)
	}
	jwKey, err := jwk.FromRaw(key)
	if err != nil {
		return fmt.Errorf("failed to generate a JWK from secret %v content: %w", secretName, err)
	}
	buf, err := json.Marshal(jwKey)
	if err != nil {
		return fmt.Errorf("error parsing key: %w", err)
	}

	var azkey azkeys.JSONWebKey
	err = json.Unmarshal(buf, &azkey)
	if err != nil {
		return fmt.Errorf("error unmarshalling key: %w", err)
	}

	// Check if key exists
	keyFromVault, err := a.keysClient.GetKey(ctx, secretName, "", nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetKey, err)

	if err != nil {
		var respErr *azcore.ResponseError
		if !errors.As(err, &respErr) || respErr.StatusCode != 404 {
			return fmt.Errorf("cannot get key %v: %w", secretName, parseNewSDKError(err))
		}
	} else if keyFromVault.Tags != nil {
		// Check if managed by external-secrets
		if managedByTag, exists := keyFromVault.Tags[managedBy]; !exists || managedByTag == nil || *managedByTag != managerLabel {
			return fmt.Errorf("key %v not managed by external-secrets", secretName)
		}
	}

	// For key comparison, we'll do a simple check - if we get here and the key exists, we'll update it
	// A more sophisticated comparison could be added later if needed

	// Prepare tags for new SDK
	keyTags := map[string]*string{
		managedBy: to.Ptr(managerLabel),
	}
	for k, v := range tags {
		keyTags[k] = &v
	}

	params := azkeys.ImportKeyParameters{
		Key: &azkey,
		KeyAttributes: &azkeys.KeyAttributes{
			Enabled: to.Ptr(true),
		},
		Tags: keyTags,
	}

	_, err = a.keysClient.ImportKey(ctx, secretName, params, nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVImportKey, err)
	if err != nil {
		return fmt.Errorf("could not import key %v: %w", secretName, parseNewSDKError(err))
	}
	return nil
}

// isValidSecret checks if a secret is valid and enabled.
func (a *Azure) isValidSecret(secret *azsecrets.SecretProperties) bool {
	return secret.ID != nil &&
		secret.Attributes != nil &&
		*secret.Attributes.Enabled
}

// secretMatchesTags checks if secret matches required tags.
func (a *Azure) secretMatchesTags(secret *azsecrets.SecretProperties, requiredTags map[string]string) bool {
	if len(requiredTags) == 0 {
		return true
	}

	for k, v := range requiredTags {
		if val, ok := secret.Tags[k]; !ok || val == nil || *val != v {
			return false
		}
	}
	return true
}

// secretMatchesNamePattern checks if secret name matches the regex pattern.
// Logs error and returns false if the regex is invalid to ensure failed matches are excluded.
func (a *Azure) secretMatchesNamePattern(secretName string, nameRef *esv1.FindName) bool {
	if nameRef == nil || nameRef.RegExp == "" {
		return true
	}

	isMatch, err := regexp.MatchString(nameRef.RegExp, secretName)
	if err != nil {
		// Log invalid regex pattern and return false to exclude this secret
		// This ensures that malformed regex patterns don't silently pass
		fmt.Printf("invalid regex pattern %q: %v\n", nameRef.RegExp, err)
		return false
	}
	return isMatch
}

// processSecretsPage processes a single page of secrets from the list operation.
func (a *Azure) processSecretsPage(ctx context.Context, secrets []*azsecrets.SecretProperties, ref esv1.ExternalSecretFind, secretsMap map[string][]byte) error {
	for _, secret := range secrets {
		if !a.isValidSecret(secret) {
			continue
		}

		if !a.secretMatchesTags(secret, ref.Tags) {
			continue
		}

		secretName := secret.ID.Name()
		if !a.secretMatchesNamePattern(secretName, ref.Name) {
			continue
		}

		// Get the secret value
		secretResp, err := a.secretsClient.GetSecret(ctx, secretName, "", nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
		if err != nil {
			return parseNewSDKError(err)
		}

		if secretResp.Value != nil {
			secretsMap[secretName] = []byte(*secretResp.Value)
		}
	}
	return nil
}

func (a *Azure) getAllSecretsWithNewSDK(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	secretsMap := make(map[string][]byte)
	pager := a.secretsClient.NewListSecretPropertiesPager(nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecrets, err)
		if err != nil {
			return nil, parseNewSDKError(err)
		}

		if err := a.processSecretsPage(ctx, page.Value, ref, secretsMap); err != nil {
			return nil, err
		}
	}
	return secretsMap, nil
}

func (a *Azure) getSecretTagsWithNewSDK(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string]*string, error) {
	_, secretName := getObjType(ref)
	secretResp, err := a.secretsClient.GetSecret(ctx, secretName, ref.Version, nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
	if err != nil {
		return nil, parseNewSDKError(err)
	}

	secretTagsData := make(map[string]*string)

	for tagname, tagval := range secretResp.Tags {
		name := secretName + "_" + tagname
		kv := make(map[string]string)
		err = json.Unmarshal([]byte(*tagval), &kv)
		// if the tagvalue is not in JSON format then we added to secretTagsData we added as it is
		if err != nil {
			secretTagsData[name] = tagval
		} else {
			for k, v := range kv {
				value := v
				secretTagsData[name+"_"+k] = &value
			}
		}
	}
	return secretTagsData, nil
}

// Helper functions for new Azure SDK

// getCloudConfiguration returns the appropriate cloud configuration for the environment type.
func getCloudConfiguration(provider *esv1.AzureKVProvider) (cloud.Configuration, error) {
	if provider.CustomCloudConfig != nil {
		if !ptr.ToBool(provider.UseAzureSDK) {
			return cloud.Configuration{}, errors.New("CustomCloudConfig requires UseAzureSDK to be set to true")
		}

		var baseConfig cloud.Configuration
		switch provider.EnvironmentType {
		case esv1.AzureEnvironmentGermanCloud:
			return cloud.Configuration{}, errors.New("Azure Germany (Microsoft Cloud Deutschland) was discontinued on October 29, 2021. Please use AzureStackCloud with custom configuration or migrate to public cloud regions")
		case esv1.AzureEnvironmentPublicCloud:
			baseConfig = cloud.AzurePublic
		case esv1.AzureEnvironmentUSGovernmentCloud:
			baseConfig = cloud.AzureGovernment
		case esv1.AzureEnvironmentChinaCloud:
			baseConfig = cloud.AzureChina
		case esv1.AzureEnvironmentAzureStackCloud:
			baseConfig = cloud.Configuration{
				Services: map[cloud.ServiceName]cloud.ServiceConfiguration{},
			}
		default:
			baseConfig = cloud.AzurePublic
		}

		return buildCustomCloudConfiguration(provider.CustomCloudConfig, baseConfig)
	}

	// no custom config - use standard cloud configurations
	switch provider.EnvironmentType {
	case esv1.AzureEnvironmentPublicCloud:
		return cloud.AzurePublic, nil
	case esv1.AzureEnvironmentUSGovernmentCloud:
		return cloud.AzureGovernment, nil
	case esv1.AzureEnvironmentChinaCloud:
		return cloud.AzureChina, nil
	case esv1.AzureEnvironmentGermanCloud:
		return cloud.Configuration{}, errors.New("Azure Germany (Microsoft Cloud Deutschland) was discontinued on October 29, 2021. Please use AzureStackCloud with custom configuration or migrate to public cloud regions")
	case esv1.AzureEnvironmentAzureStackCloud:
		return cloud.Configuration{}, errors.New("CustomCloudConfig is required when EnvironmentType is AzureStackCloud")
	default:
		return cloud.AzurePublic, nil
	}
}

// buildCustomCloudConfiguration creates a custom cloud.Configuration by merging custom config with base config.
func buildCustomCloudConfiguration(config *esv1.AzureCustomCloudConfig, baseConfig cloud.Configuration) (cloud.Configuration, error) {
	cloudConfig := cloud.Configuration{
		ActiveDirectoryAuthorityHost: baseConfig.ActiveDirectoryAuthorityHost,
		Services:                     map[cloud.ServiceName]cloud.ServiceConfiguration{},
	}

	for k, v := range baseConfig.Services {
		cloudConfig.Services[k] = v
	}

	// Set Active Directory endpoint with custom value (required)
	cloudConfig.ActiveDirectoryAuthorityHost = config.ActiveDirectoryEndpoint

	// Set Resource Manager endpoint if provided
	if config.ResourceManagerEndpoint != nil {
		cloudConfig.Services[cloud.ResourceManager] = cloud.ServiceConfiguration{
			Audience: *config.ResourceManagerEndpoint,
			Endpoint: *config.ResourceManagerEndpoint,
		}
	}

	// Note: Key Vault endpoint and DNS suffix are handled directly by the Key Vault client
	// through the vault URL, not through the cloud configuration

	return cloudConfig, nil
}

// buildManagedIdentityCredential creates a ManagedIdentityCredential.
func buildManagedIdentityCredential(az *Azure, cloudConfig cloud.Configuration) (azcore.TokenCredential, error) {
	opts := &azidentity.ManagedIdentityCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: cloudConfig,
		},
	}

	// Configure user-assigned identity if specified
	if az.provider.IdentityID != nil {
		opts.ID = azidentity.ClientID(*az.provider.IdentityID)
	}

	return azidentity.NewManagedIdentityCredential(opts)
}

// buildServicePrincipalCredential creates service principal credentials.
func buildServicePrincipalCredential(ctx context.Context, az *Azure, cloudConfig cloud.Configuration) (azcore.TokenCredential, error) {
	if az.provider.TenantID == nil {
		return nil, errors.New(errMissingTenant)
	}
	if az.provider.AuthSecretRef == nil {
		return nil, errors.New(errMissingSecretRef)
	}
	if az.provider.AuthSecretRef.ClientID == nil {
		return nil, errors.New(errMissingClientIDSecret)
	}

	// Get clientID
	clientID, err := resolvers.SecretKeyRef(
		ctx,
		az.crClient,
		az.store.GetKind(),
		az.namespace,
		az.provider.AuthSecretRef.ClientID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get clientID: %w", err)
	}

	clientOpts := azcore.ClientOptions{
		Cloud: cloudConfig,
	}

	// Check if using client secret or client certificate
	if az.provider.AuthSecretRef.ClientSecret != nil && az.provider.AuthSecretRef.ClientCertificate != nil {
		return nil, errors.New(errInvalidClientCredentials)
	}

	if az.provider.AuthSecretRef.ClientSecret != nil {
		// Client secret authentication
		clientSecret, err := resolvers.SecretKeyRef(
			ctx,
			az.crClient,
			az.store.GetKind(),
			az.namespace,
			az.provider.AuthSecretRef.ClientSecret,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get clientSecret: %w", err)
		}

		opts := &azidentity.ClientSecretCredentialOptions{
			ClientOptions: clientOpts,
		}

		return azidentity.NewClientSecretCredential(*az.provider.TenantID, clientID, clientSecret, opts)
	} else if az.provider.AuthSecretRef.ClientCertificate != nil {
		// Client certificate authentication
		certData, err := resolvers.SecretKeyRef(
			ctx,
			az.crClient,
			az.store.GetKind(),
			az.namespace,
			az.provider.AuthSecretRef.ClientCertificate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get clientCertificate: %w", err)
		}

		// Parse certificate and key
		certs, key, err := azidentity.ParseCertificates([]byte(certData), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate: %w", err)
		}

		opts := &azidentity.ClientCertificateCredentialOptions{
			ClientOptions: clientOpts,
		}

		return azidentity.NewClientCertificateCredential(*az.provider.TenantID, clientID, certs, key, opts)
	}

	return nil, errors.New(errMissingClientIDSecret)
}

// buildWorkloadIdentityCredential creates workload identity credentials.
func buildWorkloadIdentityCredential(ctx context.Context, az *Azure, cloudConfig cloud.Configuration) (azcore.TokenCredential, error) {
	clientOpts := azcore.ClientOptions{
		Cloud: cloudConfig,
	}

	// If no serviceAccountRef is provided, use environment variables (webhook mode)
	if az.provider.ServiceAccountRef == nil {
		opts := &azidentity.WorkloadIdentityCredentialOptions{
			ClientOptions: clientOpts,
		}
		return azidentity.NewWorkloadIdentityCredential(opts)
	}

	// ServiceAccountRef mode - get values from service account and secrets
	ns := az.namespace
	if az.store.GetKind() == esv1.ClusterSecretStoreKind && az.provider.ServiceAccountRef.Namespace != nil {
		ns = *az.provider.ServiceAccountRef.Namespace
	}

	var sa corev1.ServiceAccount
	err := az.crClient.Get(ctx, types.NamespacedName{
		Name:      az.provider.ServiceAccountRef.Name,
		Namespace: ns,
	}, &sa)
	if err != nil {
		return nil, fmt.Errorf("failed to get service account: %w", err)
	}

	// Get clientID from service account annotations
	var clientID string
	if val, found := sa.ObjectMeta.Annotations[AnnotationClientID]; found {
		clientID = val
	} else {
		return nil, fmt.Errorf(errMissingClient, AnnotationClientID)
	}

	// Get tenantID
	var tenantID string
	if az.provider.TenantID != nil {
		tenantID = *az.provider.TenantID
	} else if val, found := sa.ObjectMeta.Annotations[AnnotationTenantID]; found {
		tenantID = val
	} else {
		return nil, errors.New(errMissingTenant)
	}

	// Use ClientAssertionCredential to avoid filesystem access in read-only environments
	// This provides a callback function that fetches tokens dynamically
	getAssertion := func(ctx context.Context) (string, error) {
		audiences := []string{AzureDefaultAudience}
		if len(az.provider.ServiceAccountRef.Audiences) > 0 {
			audiences = append(audiences, az.provider.ServiceAccountRef.Audiences...)
		}

		token, err := FetchSAToken(ctx, ns, az.provider.ServiceAccountRef.Name, audiences, az.kubeClient)
		if err != nil {
			return "", fmt.Errorf("failed to fetch service account token: %w", err)
		}
		return token, nil
	}

	opts := &azidentity.ClientAssertionCredentialOptions{
		ClientOptions: clientOpts,
	}

	return azidentity.NewClientAssertionCredential(tenantID, clientID, getAssertion, opts)
}

// canDeleteWithNewSDK checks if a resource can be deleted based on tags and error status.
func canDeleteWithNewSDK(tags map[string]*string, err error) (bool, error) {
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			if respErr.StatusCode == 404 {
				// Resource doesn't exist, nothing to delete
				return false, nil
			}
			// Other API error
			return false, fmt.Errorf("unexpected api error: %w", err)
		}
		// Non-Azure error
		return false, fmt.Errorf("could not parse error: %w", err)
	}

	// Check if managed by external-secrets
	if tags == nil {
		return false, nil
	}

	managedByTag, exists := tags[managedBy]
	if !exists || managedByTag == nil || *managedByTag != managerLabel {
		// Not managed by external-secrets, don't delete
		return false, nil
	}

	return true, nil
}

// Delete methods using new Azure SDK.
func (a *Azure) deleteKeyVaultSecretWithNewSDK(ctx context.Context, secretName string) error {
	secret, err := a.secretsClient.GetSecret(ctx, secretName, "", nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)

	ok, err := canDeleteWithNewSDK(secret.Tags, err)
	if err != nil {
		return fmt.Errorf("error getting secret %v: %w", secretName, err)
	}

	if ok {
		_, err = a.secretsClient.DeleteSecret(ctx, secretName, nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVDeleteSecret, err)
		if err != nil {
			return fmt.Errorf("error deleting secret %v: %w", secretName, err)
		}
	}
	return nil
}

func (a *Azure) deleteKeyVaultCertificateWithNewSDK(ctx context.Context, certName string) error {
	cert, err := a.certsClient.GetCertificate(ctx, certName, "", nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetCertificate, err)

	ok, err := canDeleteWithNewSDK(cert.Tags, err)
	if err != nil {
		return fmt.Errorf("error getting certificate %v: %w", certName, err)
	}

	if ok {
		_, err = a.certsClient.DeleteCertificate(ctx, certName, nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVDeleteCertificate, err)
		if err != nil {
			return fmt.Errorf("error deleting certificate %v: %w", certName, err)
		}
	}
	return nil
}

func (a *Azure) deleteKeyVaultKeyWithNewSDK(ctx context.Context, keyName string) error {
	key, err := a.keysClient.GetKey(ctx, keyName, "", nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetKey, err)

	ok, err := canDeleteWithNewSDK(key.Tags, err)
	if err != nil {
		return fmt.Errorf("error getting key %v: %w", keyName, err)
	}

	if ok {
		_, err = a.keysClient.DeleteKey(ctx, keyName, nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVDeleteKey, err)
		if err != nil {
			return fmt.Errorf("error deleting key %v: %w", keyName, err)
		}
	}
	return nil
}

// GetSecret implementation using new Azure SDK.
func (a *Azure) getSecretWithNewSDK(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	objectType, secretName := getObjType(ref)

	switch objectType {
	case defaultObjType:
		// Get secret using new SDK
		resp, err := a.secretsClient.GetSecret(ctx, secretName, ref.Version, nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
		if err != nil {
			return nil, parseNewSDKError(err)
		}
		if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
			return getSecretTag(resp.Tags, ref.Property)
		}
		return getProperty(*resp.Value, ref.Property, ref.Key)

	case objectTypeCert:
		// Get certificate using new SDK
		resp, err := a.certsClient.GetCertificate(ctx, secretName, ref.Version, nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetCertificate, err)
		if err != nil {
			return nil, parseNewSDKError(err)
		}
		if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
			return getSecretTag(resp.Tags, ref.Property)
		}
		return resp.CER, nil

	case objectTypeKey:
		// Get key using new SDK
		resp, err := a.keysClient.GetKey(ctx, secretName, ref.Version, nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetKey, err)
		if err != nil {
			return nil, parseNewSDKError(err)
		}
		if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
			return getSecretTag(resp.Tags, ref.Property)
		}
		keyBytes, err := json.Marshal(resp.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal key: %w", err)
		}
		return getProperty(string(keyBytes), ref.Property, ref.Key)
	}

	return nil, fmt.Errorf(errUnknownObjectType, secretName)
}

// secretExistsWithNewSDK checks if a secret/certificate/key exists in Azure Key Vault using the new SDK.
// Returns (true, nil) if the object exists, (false, nil) if it doesn't exist, or (false, err) on error.
func (a *Azure) secretExistsWithNewSDK(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	objectType, secretName := getObjType(esv1.ExternalSecretDataRemoteRef{Key: remoteRef.GetRemoteKey()})

	var err error
	switch objectType {
	case defaultObjType:
		_, err = a.secretsClient.GetSecret(ctx, secretName, "", nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
	case objectTypeCert:
		_, err = a.certsClient.GetCertificate(ctx, secretName, "", nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetCertificate, err)
	case objectTypeKey:
		_, err = a.keysClient.GetKey(ctx, secretName, "", nil)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetKey, err)
	default:
		errMsg := fmt.Sprintf("secret type '%v' is not supported", objectType)
		return false, errors.New(errMsg)
	}

	err = parseNewSDKError(err)
	if err != nil {
		var noSecretErr esv1.NoSecretError
		if errors.As(err, &noSecretErr) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// parseNewSDKError converts new Azure SDK errors to the same format as legacy errors.
func parseNewSDKError(err error) error {
	if err == nil {
		return nil
	}

	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		if respErr.StatusCode == 404 {
			return esv1.NoSecretError{}
		}
		// Return error in the same format as the legacy parseError function
		return err
	}

	return err
}
