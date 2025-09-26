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

package oracle

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	"github.com/oracle/oci-go-sdk/v65/secrets"
	"github.com/oracle/oci-go-sdk/v65/vault"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errOracleClient               = "cannot setup new oracle client: %w"
	errORACLECredSecretName       = "invalid oracle SecretStore resource: missing oracle APIKey"
	errUninitalizedOracleProvider = "provider oracle is not initialized"
	errFetchSAKSecret             = "could not fetch SecretAccessKey secret: %w"
	errMissingPK                  = "missing PrivateKey"
	errMissingUser                = "missing User ID"
	errMissingTenancy             = "missing Tenancy ID"
	errMissingRegion              = "missing Region"
	errMissingFingerprint         = "missing Fingerprint"
	errMissingVault               = "missing Vault"
	errJSONSecretUnmarshal        = "unable to unmarshal secret from JSON: %w"
	errMissingKey                 = "missing Key in secret: %s"
	errUnexpectedContent          = "unexpected secret bundle content"
	errSettingOCIEnvVariables     = "unable to set OCI SDK environment variable %s: %w"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &VaultManagementService{}
var _ esv1.Provider = &VaultManagementService{}

type VaultManagementService struct {
	Client                VMInterface
	KmsVaultClient        KmsVCInterface
	VaultClient           VaultInterface
	vault                 string
	compartment           string
	encryptionKey         string
	workloadIdentityMutex sync.Mutex
}

type VMInterface interface {
	GetSecretBundleByName(ctx context.Context, request secrets.GetSecretBundleByNameRequest) (secrets.GetSecretBundleByNameResponse, error)
}

type KmsVCInterface interface {
	GetVault(ctx context.Context, request keymanagement.GetVaultRequest) (response keymanagement.GetVaultResponse, err error)
}

type VaultInterface interface {
	ListSecrets(ctx context.Context, request vault.ListSecretsRequest) (response vault.ListSecretsResponse, err error)
	CreateSecret(ctx context.Context, request vault.CreateSecretRequest) (response vault.CreateSecretResponse, err error)
	UpdateSecret(ctx context.Context, request vault.UpdateSecretRequest) (response vault.UpdateSecretResponse, err error)
	ScheduleSecretDeletion(ctx context.Context, request vault.ScheduleSecretDeletionRequest) (response vault.ScheduleSecretDeletionResponse, err error)
}

const (
	SecretNotFound = iota
	SecretExists
	SecretAPIError
)

func (vms *VaultManagementService) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if vms.encryptionKey == "" {
		return errors.New("SecretStore must reference encryption key")
	}
	value := secret.Data[data.GetSecretKey()]
	if data.GetSecretKey() == "" {
		secretData := map[string]string{}
		for k, v := range secret.Data {
			secretData[k] = string(v)
		}
		jsonSecret, err := json.Marshal(secretData)
		if err != nil {
			return fmt.Errorf("unable to create json %v from value: %v", value, secretData)
		}
		value = jsonSecret
	}

	secretName := data.GetRemoteKey()
	encodedValue := base64.StdEncoding.EncodeToString(value)
	sec, action, err := vms.getSecretBundleWithCode(ctx, secretName)
	switch action {
	case SecretNotFound:
		_, err = vms.VaultClient.CreateSecret(ctx, vault.CreateSecretRequest{
			CreateSecretDetails: vault.CreateSecretDetails{
				CompartmentId: &vms.compartment,
				KeyId:         &vms.encryptionKey,
				SecretContent: vault.Base64SecretContentDetails{
					Content: &encodedValue,
				},
				SecretName: &secretName,
				VaultId:    &vms.vault,
			},
		})
		return sanitizeOCISDKErr(err)
	case SecretExists:
		payload, err := decodeBundle(sec)
		if err != nil {
			return err
		}
		if bytes.Equal(payload, value) {
			return nil
		}
		_, err = vms.VaultClient.UpdateSecret(ctx, vault.UpdateSecretRequest{
			SecretId: sec.SecretId,
			UpdateSecretDetails: vault.UpdateSecretDetails{
				SecretContent: vault.Base64SecretContentDetails{
					Content: &encodedValue,
				},
			},
		})
		return sanitizeOCISDKErr(err)
	default:
		return sanitizeOCISDKErr(err)
	}
}

func (vms *VaultManagementService) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	secretName := remoteRef.GetRemoteKey()
	resp, action, err := vms.getSecretBundleWithCode(ctx, secretName)
	switch action {
	case SecretNotFound:
		return nil
	case SecretExists:
		if resp.TimeOfDeletion != nil {
			return nil
		}
		_, err = vms.VaultClient.ScheduleSecretDeletion(ctx, vault.ScheduleSecretDeletionRequest{
			SecretId: resp.SecretId,
		})
		return sanitizeOCISDKErr(err)
	default:
		return sanitizeOCISDKErr(err)
	}
}

func (vms *VaultManagementService) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

func (vms *VaultManagementService) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	var page *string
	var summaries []vault.SecretSummary

	for {
		resp, err := vms.VaultClient.ListSecrets(ctx, vault.ListSecretsRequest{
			CompartmentId: &vms.compartment,
			Page:          page,
			VaultId:       &vms.vault,
		})
		if err != nil {
			return nil, sanitizeOCISDKErr(err)
		}
		summaries = append(summaries, resp.Items...)
		if page = resp.OpcNextPage; resp.OpcNextPage == nil {
			break
		}
	}

	return vms.filteredSummaryResult(ctx, summaries, ref)
}

func (vms *VaultManagementService) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(vms.Client) {
		return nil, errors.New(errUninitalizedOracleProvider)
	}

	sec, err := vms.Client.GetSecretBundleByName(ctx, secrets.GetSecretBundleByNameRequest{
		VaultId:    &vms.vault,
		SecretName: &ref.Key,
		Stage:      secrets.GetSecretBundleByNameStageEnum(ref.Version),
	})
	if err != nil {
		return nil, sanitizeOCISDKErr(err)
	}

	payload, err := decodeBundle(sec)
	if err != nil {
		return nil, err
	}
	if ref.Property == "" {
		return payload, nil
	}

	val := gjson.Get(string(payload), ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf(errMissingKey, ref.Key)
	}

	return []byte(val.String()), nil
}

func decodeBundle(sec secrets.GetSecretBundleByNameResponse) ([]byte, error) {
	bt, ok := sec.SecretBundleContent.(secrets.Base64SecretBundleContentDetails)
	if !ok {
		return nil, errors.New(errUnexpectedContent)
	}
	payload, err := base64.StdEncoding.DecodeString(*bt.Content)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (vms *VaultManagementService) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := vms.GetSecret(ctx, ref)
	if err != nil {
		return nil, sanitizeOCISDKErr(err)
	}
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}
	return secretData, nil
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (vms *VaultManagementService) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient constructs a new secrets client based on the provided store.
func (vms *VaultManagementService) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	oracleSpec := storeSpec.Provider.Oracle

	if oracleSpec.Vault == "" {
		return nil, errors.New(errMissingVault)
	}

	if oracleSpec.Region == "" {
		return nil, errors.New(errMissingRegion)
	}

	configurationProvider, err := vms.constructProvider(ctx, store, oracleSpec, kube, namespace)
	if err != nil {
		return nil, err
	}

	secretManagementService, err := secrets.NewSecretsClientWithConfigurationProvider(configurationProvider)
	if err != nil {
		return nil, fmt.Errorf(errOracleClient, err)
	}
	secretManagementService.SetRegion(oracleSpec.Region)

	kmsVaultClient, err := keymanagement.NewKmsVaultClientWithConfigurationProvider(configurationProvider)
	if err != nil {
		return nil, fmt.Errorf(errOracleClient, err)
	}
	kmsVaultClient.SetRegion(oracleSpec.Region)

	vaultClient, err := vault.NewVaultsClientWithConfigurationProvider(configurationProvider)
	if err != nil {
		return nil, fmt.Errorf(errOracleClient, err)
	}
	vaultClient.SetRegion(oracleSpec.Region)

	if storeSpec.RetrySettings != nil {
		if err := vms.configureRetryPolicy(storeSpec, secretManagementService, kmsVaultClient, vaultClient); err != nil {
			return nil, fmt.Errorf(errOracleClient, err)
		}
	}

	return &VaultManagementService{
		Client:         secretManagementService,
		KmsVaultClient: kmsVaultClient,
		VaultClient:    vaultClient,
		vault:          oracleSpec.Vault,
		compartment:    oracleSpec.Compartment,
		encryptionKey:  oracleSpec.EncryptionKey,
	}, nil
}

func (vms *VaultManagementService) constructOptions(storeSpec *esv1.SecretStoreSpec) ([]common.RetryPolicyOption, error) {
	opts := []common.RetryPolicyOption{common.WithShouldRetryOperation(common.DefaultShouldRetryOperation)}

	if mr := storeSpec.RetrySettings.MaxRetries; mr != nil {
		attempts := safeConvert(*mr)
		opts = append(opts, common.WithMaximumNumberAttempts(attempts))
	}

	if ri := storeSpec.RetrySettings.RetryInterval; ri != nil {
		i, err := time.ParseDuration(*storeSpec.RetrySettings.RetryInterval)
		if err != nil {
			return nil, fmt.Errorf(errOracleClient, err)
		}
		opts = append(opts, common.WithFixedBackoff(i))
	}
	return opts, nil
}

func safeConvert(i int32) uint {
	if i < 0 {
		return 0
	}

	return uint(i)
}

func (vms *VaultManagementService) getSecretBundleWithCode(ctx context.Context, secretName string) (secrets.GetSecretBundleByNameResponse, int, error) {
	// Try to look up the secret, which will determine if we should create or update the secret.
	resp, err := vms.Client.GetSecretBundleByName(ctx, secrets.GetSecretBundleByNameRequest{
		SecretName: &secretName,
		VaultId:    &vms.vault,
	})
	// Get a PushSecret action depending on the ListSecrets response.
	action := getSecretBundleCode(err)
	return resp, action, err
}

func getSecretBundleCode(err error) int {
	if err != nil {
		// If we got a 404 service error, try to create the secret.

		if serviceErr, ok := err.(common.ServiceError); ok && serviceErr.GetHTTPStatusCode() == 404 {
			return SecretNotFound
		}
		return SecretAPIError
	}
	// Otherwise, update the existing secret.
	return SecretExists
}

func (vms *VaultManagementService) filteredSummaryResult(ctx context.Context, secretSummaries []vault.SecretSummary, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	secretMap := map[string][]byte{}
	for _, summary := range secretSummaries {
		matches, err := matchesRef(summary, ref)
		if err != nil {
			return nil, err
		}
		if !matches || summary.TimeOfDeletion != nil {
			continue
		}
		secret, err := vms.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
			Key: *summary.SecretName,
		})
		if err != nil {
			return nil, err
		}
		secretMap[*summary.SecretName] = secret
	}
	return secretMap, nil
}

func matchesRef(secretSummary vault.SecretSummary, ref esv1.ExternalSecretFind) (bool, error) {
	if ref.Name != nil {
		matchString, err := regexp.MatchString(ref.Name.RegExp, *secretSummary.SecretName)
		if err != nil {
			return false, err
		}
		return matchString, nil
	}
	for k, v := range ref.Tags {
		if val, ok := secretSummary.FreeformTags[k]; ok {
			if val == v {
				return true, nil
			}
		}
	}
	return false, nil
}

func getSecretData(ctx context.Context, kube kclient.Client, namespace, storeKind string, secretRef esmeta.SecretKeySelector) (string, error) {
	if secretRef.Name == "" {
		return "", errors.New(errORACLECredSecretName)
	}
	secret, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		storeKind,
		namespace,
		&secretRef,
	)
	if err != nil {
		return "", fmt.Errorf(errFetchSAKSecret, err)
	}
	return secret, nil
}

func getUserAuthConfigurationProvider(ctx context.Context, kube kclient.Client, store *esv1.OracleProvider, namespace, storeKind, region string) (common.ConfigurationProvider, error) {
	privateKey, err := getSecretData(ctx, kube, namespace, storeKind, store.Auth.SecretRef.PrivateKey)
	if err != nil {
		return nil, err
	}
	if privateKey == "" {
		return nil, errors.New(errMissingPK)
	}

	fingerprint, err := getSecretData(ctx, kube, namespace, storeKind, store.Auth.SecretRef.Fingerprint)
	if err != nil {
		return nil, err
	}
	if fingerprint == "" {
		return nil, errors.New(errMissingFingerprint)
	}

	if store.Auth.User == "" {
		return nil, errors.New(errMissingUser)
	}

	if store.Auth.Tenancy == "" {
		return nil, errors.New(errMissingTenancy)
	}

	return common.NewRawConfigurationProvider(store.Auth.Tenancy, store.Auth.User, region, fingerprint, privateKey, nil), nil
}

func (vms *VaultManagementService) Close(_ context.Context) error {
	return nil
}

func (vms *VaultManagementService) Validate() (esv1.ValidationResult, error) {
	_, err := vms.KmsVaultClient.GetVault(
		context.Background(), keymanagement.GetVaultRequest{
			VaultId: &vms.vault,
		},
	)
	if err != nil {
		failure, ok := common.IsServiceError(err)
		if ok {
			code := failure.GetCode()
			switch code {
			case "NotAuthenticated":
				return esv1.ValidationResultError, sanitizeOCISDKErr(err)
			case "NotAuthorizedOrNotFound":
				// User authentication was successful, but user might not have a permission like:
				//
				// Allow group external_secrets to read vaults in tenancy
				//
				// Which is fine, because to read secrets we only need:
				//
				// Allow group external_secrets to read secret-family in tenancy
				//
				// But we can't test for this permission without knowing the name of a secret
				return esv1.ValidationResultUnknown, sanitizeOCISDKErr(err)
			default:
				return esv1.ValidationResultError, sanitizeOCISDKErr(err)
			}
		} else {
			return esv1.ValidationResultError, err
		}
	}

	return esv1.ValidationResultReady, nil
}

func (vms *VaultManagementService) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	oracleSpec := storeSpec.Provider.Oracle

	vault := oracleSpec.Vault
	if vault == "" {
		return nil, errors.New("vault cannot be empty")
	}

	region := oracleSpec.Region
	if region == "" {
		return nil, errors.New("region cannot be empty")
	}

	auth := oracleSpec.Auth
	if auth == nil {
		return nil, nil
	}

	user := oracleSpec.Auth.User
	if user == "" {
		return nil, errors.New("user cannot be empty")
	}

	tenant := oracleSpec.Auth.Tenancy
	if tenant == "" {
		return nil, errors.New("tenant cannot be empty")
	}
	privateKey := oracleSpec.Auth.SecretRef.PrivateKey

	if privateKey.Name == "" {
		return nil, errors.New("privateKey.name cannot be empty")
	}

	if privateKey.Key == "" {
		return nil, errors.New("privateKey.key cannot be empty")
	}

	err := utils.ValidateSecretSelector(store, privateKey)
	if err != nil {
		return nil, err
	}

	fingerprint := oracleSpec.Auth.SecretRef.Fingerprint

	if fingerprint.Name == "" {
		return nil, errors.New("fingerprint.name cannot be empty")
	}

	if fingerprint.Key == "" {
		return nil, errors.New("fingerprint.key cannot be empty")
	}

	err = utils.ValidateSecretSelector(store, fingerprint)
	if err != nil {
		return nil, err
	}

	if oracleSpec.ServiceAccountRef != nil {
		if err := utils.ValidateReferentServiceAccountSelector(store, *oracleSpec.ServiceAccountRef); err != nil {
			return nil, fmt.Errorf("invalid ServiceAccountRef: %w", err)
		}
	}

	return nil, nil
}

func (vms *VaultManagementService) getWorkloadIdentityProvider(store esv1.GenericStore, serviceAcccountRef *esmeta.ServiceAccountSelector, region, namespace string) (configurationProvider common.ConfigurationProvider, err error) {
	defer func() {
		if uerr := os.Unsetenv(auth.ResourcePrincipalVersionEnvVar); uerr != nil {
			err = errors.Join(err, fmt.Errorf(errSettingOCIEnvVariables, auth.ResourcePrincipalRegionEnvVar, uerr))
		}
		if uerr := os.Unsetenv(auth.ResourcePrincipalRegionEnvVar); uerr != nil {
			err = errors.Join(err, fmt.Errorf("unabled to unset OCI SDK environment variable %s: %w", auth.ResourcePrincipalVersionEnvVar, uerr))
		}
		vms.workloadIdentityMutex.Unlock()
	}()
	vms.workloadIdentityMutex.Lock()
	// OCI SDK requires specific environment variables for workload identity.
	if err := os.Setenv(auth.ResourcePrincipalVersionEnvVar, auth.ResourcePrincipalVersion2_2); err != nil {
		return nil, fmt.Errorf(errSettingOCIEnvVariables, auth.ResourcePrincipalVersionEnvVar, err)
	}
	if err := os.Setenv(auth.ResourcePrincipalRegionEnvVar, region); err != nil {
		return nil, fmt.Errorf(errSettingOCIEnvVariables, auth.ResourcePrincipalRegionEnvVar, err)
	}
	// If no service account is specified, use the pod service account to create the Workload Identity provider.
	if serviceAcccountRef == nil {
		return auth.OkeWorkloadIdentityConfigurationProvider()
	}
	// Ensure the service account ref is being used appropriately, so arbitrary tokens are not minted by the provider.
	if err = utils.ValidateServiceAccountSelector(store, *serviceAcccountRef); err != nil {
		return nil, fmt.Errorf("invalid ServiceAccountRef: %w", err)
	}
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	tokenProvider := NewTokenProvider(clientset, serviceAcccountRef, namespace)
	return auth.OkeWorkloadIdentityConfigurationProviderWithServiceAccountTokenProvider(tokenProvider)
}

func (vms *VaultManagementService) constructProvider(ctx context.Context, store esv1.GenericStore, oracleSpec *esv1.OracleProvider, kube kclient.Client, namespace string) (common.ConfigurationProvider, error) {
	var (
		configurationProvider common.ConfigurationProvider
		err                   error
	)

	if oracleSpec.PrincipalType == esv1.WorkloadPrincipal {
		configurationProvider, err = vms.getWorkloadIdentityProvider(store, oracleSpec.ServiceAccountRef, oracleSpec.Region, namespace)
	} else if oracleSpec.PrincipalType == esv1.InstancePrincipal || oracleSpec.Auth == nil {
		configurationProvider, err = auth.InstancePrincipalConfigurationProvider()
	} else {
		configurationProvider, err = getUserAuthConfigurationProvider(ctx, kube, oracleSpec, namespace, store.GetObjectKind().GroupVersionKind().Kind, oracleSpec.Region)
	}
	if err != nil {
		return nil, fmt.Errorf(errOracleClient, err)
	}

	return configurationProvider, nil
}

func (vms *VaultManagementService) configureRetryPolicy(
	storeSpec *esv1.SecretStoreSpec,
	secretManagementService secrets.SecretsClient,
	kmsVaultClient keymanagement.KmsVaultClient,
	vaultClient vault.VaultsClient,
) error {
	opts, err := vms.constructOptions(storeSpec)
	if err != nil {
		return err
	}

	customRetryPolicy := common.NewRetryPolicyWithOptions(opts...)

	secretManagementService.SetCustomClientConfiguration(common.CustomClientConfiguration{
		RetryPolicy: &customRetryPolicy,
	})

	kmsVaultClient.SetCustomClientConfiguration(common.CustomClientConfiguration{
		RetryPolicy: &customRetryPolicy,
	})

	vaultClient.SetCustomClientConfiguration(common.CustomClientConfiguration{
		RetryPolicy: &customRetryPolicy,
	})

	return err
}

func sanitizeOCISDKErr(err error) error {
	if err == nil {
		return nil
	}
	// If we have a ServiceError from the OCI SDK, strip only the message from the verbose error

	if serviceError, ok := err.(common.ServiceErrorRichInfo); ok {
		return fmt.Errorf("%s service failed to %s, HTTP status code %d: %s", serviceError.GetTargetService(), serviceError.GetOperationName(), serviceError.GetHTTPStatusCode(), serviceError.GetMessage())
	}
	return err
}

func init() {
	esv1.Register(&VaultManagementService{}, &esv1.SecretStoreProvider{
		Oracle: &esv1.OracleProvider{},
	}, esv1.MaintenanceStatusMaintained)
}
