/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vault

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/go-logr/logr"
	"github.com/golang-jwt/jwt/v5"
	vault "github.com/hashicorp/vault/api"
	approle "github.com/hashicorp/vault/api/auth/approle"
	authaws "github.com/hashicorp/vault/api/auth/aws"
	authkubernetes "github.com/hashicorp/vault/api/auth/kubernetes"
	authldap "github.com/hashicorp/vault/api/auth/ldap"
	authuserpass "github.com/hashicorp/vault/api/auth/userpass"
	"github.com/spf13/pflag"
	"github.com/tidwall/gjson"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/cache"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/feature"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	vaultiamauth "github.com/external-secrets/external-secrets/pkg/provider/vault/iamauth"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

var (
	_           esv1beta1.Provider      = &Connector{}
	_           esv1beta1.SecretsClient = &client{}
	enableCache bool
	logger      = ctrl.Log.WithName("provider").WithName("vault")
	clientCache *cache.Cache[util.Client]
)

const (
	serviceAccTokenPath     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	defaultAWSRegion        = "us-east-1"
	defaultAWSAuthMountPath = "aws"

	errVaultStore                   = "received invalid Vault SecretStore resource: %w"
	errVaultCacheCreate             = "cannot create Vault client cache: %s"
	errVaultCacheRemove             = "error removing item from Vault client cache: %w"
	errVaultCacheEviction           = "unexpected eviction from Vault client cache"
	errVaultClient                  = "cannot setup new vault client: %w"
	errVaultCert                    = "cannot set Vault CA certificate: %w"
	errReadSecret                   = "cannot read secret data from Vault: %w"
	errAuthFormat                   = "cannot initialize Vault client: no valid auth method specified"
	errInvalidCredentials           = "invalid vault credentials: %w"
	errDataField                    = "failed to find data field"
	errJSONUnmarshall               = "failed to unmarshall JSON"
	errPathInvalid                  = "provided Path isn't a valid kv v2 path"
	errUnexpectedKey                = "unexpected key in data: %s"
	errVaultToken                   = "cannot parse Vault authentication token: %w"
	errVaultRequest                 = "error from Vault request: %w"
	errServiceAccount               = "cannot read Kubernetes service account token from file system: %w"
	errJwtNoTokenSource             = "neither `secretRef` nor `kubernetesServiceAccountToken` was supplied as token source for jwt authentication"
	errUnsupportedKvVersion         = "cannot perform find operations with kv version v1"
	errUnsupportedMetadataKvVersion = "cannot perform metadata fetch operations with kv version v1"
	errNotFound                     = "secret not found"
	errIrsaTokenEnvVarNotFoundOnPod = "expected env variable: %s not found on controller's pod"
	errIrsaTokenFileNotFoundOnPod   = "web ddentity token file not found at %s location: %w"
	errIrsaTokenFileNotReadable     = "could not read the web identity token from the file %s: %w"
	errIrsaTokenNotValidJWT         = "could not parse web identity token available at %s. not a valid jwt?: %w"
	errPodInfoNotFoundOnToken       = "could not find pod identity info on token %s: %w"

	errGetKubeSA             = "cannot get Kubernetes service account %q: %w"
	errGetKubeSASecrets      = "cannot find secrets bound to service account: %q"
	errGetKubeSANoToken      = "cannot find token in secrets bound to service account: %q"
	errGetKubeSATokenRequest = "cannot request Kubernetes service account token for service account %q: %w"

	errGetKubeSecret = "cannot get Kubernetes secret %q in namespace %q: %w"
	errSecretKeyFmt  = "cannot find secret data for key: %q"
	errConfigMapFmt  = "cannot find config map data for key: %q"

	errClientTLSAuth = "error from Client TLS Auth: %q"

	errVaultRevokeToken = "error while revoking token: %w"

	errUnknownCAProvider = "unknown caProvider type given"
	errCANamespace       = "cannot read secret for CAProvider due to missing namespace on kind ClusterSecretStore"

	errInvalidStore       = "invalid store"
	errInvalidStoreSpec   = "invalid store spec"
	errInvalidStoreProv   = "invalid store provider"
	errInvalidVaultProv   = "invalid vault provider"
	errInvalidAppRoleID   = "invalid Auth.AppRole: neither `roleId` nor `roleRef` was supplied"
	errInvalidAppRoleRef  = "invalid Auth.AppRole.RoleRef: %w"
	errInvalidAppRoleSec  = "invalid Auth.AppRole.SecretRef: %w"
	errInvalidClientCert  = "invalid Auth.Cert.ClientCert: %w"
	errInvalidCertSec     = "invalid Auth.Cert.SecretRef: %w"
	errInvalidJwtSec      = "invalid Auth.Jwt.SecretRef: %w"
	errInvalidJwtK8sSA    = "invalid Auth.Jwt.KubernetesServiceAccountToken.ServiceAccountRef: %w"
	errInvalidKubeSA      = "invalid Auth.Kubernetes.ServiceAccountRef: %w"
	errInvalidKubeSec     = "invalid Auth.Kubernetes.SecretRef: %w"
	errInvalidLdapSec     = "invalid Auth.Ldap.SecretRef: %w"
	errInvalidTokenRef    = "invalid Auth.TokenSecretRef: %w"
	errInvalidUserPassSec = "invalid Auth.UserPass.SecretRef: %w"

	errInvalidClientTLSCert   = "invalid ClientTLS.ClientCert: %w"
	errInvalidClientTLSSecret = "invalid ClientTLS.SecretRef: %w"
	errInvalidClientTLS       = "when provided, both ClientTLS.ClientCert and ClientTLS.SecretRef should be provided"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &client{}
var _ esv1beta1.Provider = &Connector{}

type client struct {
	kube      kclient.Client
	store     *esv1beta1.VaultProvider
	log       logr.Logger
	corev1    typedcorev1.CoreV1Interface
	client    util.Client
	auth      util.Auth
	logical   util.Logical
	token     util.Token
	namespace string
	storeKind string
}

func NewVaultClient(c *vault.Config) (util.Client, error) {
	cl, err := vault.NewClient(c)
	if err != nil {
		return nil, err
	}
	auth := cl.Auth()
	logical := cl.Logical()
	token := cl.Auth().Token()
	out := util.VClient{
		SetTokenFunc:     cl.SetToken,
		TokenFunc:        cl.Token,
		ClearTokenFunc:   cl.ClearToken,
		AuthField:        auth,
		AuthTokenField:   token,
		LogicalField:     logical,
		SetNamespaceFunc: cl.SetNamespace,
		AddHeaderFunc:    cl.AddHeader,
	}
	return &out, nil
}

func getVaultClient(c *Connector, store esv1beta1.GenericStore, cfg *vault.Config) (util.Client, error) {
	isStaticToken := store.GetSpec().Provider.Vault.Auth.TokenSecretRef != nil
	useCache := enableCache && !isStaticToken

	key := cache.Key{
		Name:      store.GetObjectMeta().Name,
		Namespace: store.GetObjectMeta().Namespace,
		Kind:      store.GetTypeMeta().Kind,
	}
	if useCache {
		client, ok := clientCache.Get(store.GetObjectMeta().ResourceVersion, key)
		if ok {
			return client, nil
		}
	}

	client, err := c.NewVaultClient(cfg)
	if err != nil {
		return nil, fmt.Errorf(errVaultClient, err)
	}

	if useCache && !clientCache.Contains(key) {
		clientCache.Add(store.GetObjectMeta().ResourceVersion, key, client)
	}
	return client, nil
}

type Connector struct {
	NewVaultClient func(c *vault.Config) (util.Client, error)
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (c *Connector) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}
func (c *Connector) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	// controller-runtime/client does not support TokenRequest or other subresource APIs
	// so we need to construct our own client and use it to fetch tokens
	// (for Kubernetes service account token auth)
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	return c.newClient(ctx, store, kube, clientset.CoreV1(), namespace)
}

func (c *Connector) newClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, corev1 typedcorev1.CoreV1Interface, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Vault == nil {
		return nil, errors.New(errVaultStore)
	}
	vaultSpec := storeSpec.Provider.Vault

	vStore, cfg, err := c.prepareConfig(ctx, kube, corev1, vaultSpec, storeSpec.RetrySettings, namespace, store.GetObjectKind().GroupVersionKind().Kind)
	if err != nil {
		return nil, err
	}

	client, err := getVaultClient(c, store, cfg)
	if err != nil {
		return nil, fmt.Errorf(errVaultClient, err)
	}

	return c.initClient(ctx, vStore, client, cfg, vaultSpec)
}

func (c *Connector) NewGeneratorClient(ctx context.Context, kube kclient.Client, corev1 typedcorev1.CoreV1Interface, vaultSpec *esv1beta1.VaultProvider, namespace string) (util.Client, error) {
	vStore, cfg, err := c.prepareConfig(ctx, kube, corev1, vaultSpec, nil, namespace, "Generator")
	if err != nil {
		return nil, err
	}

	client, err := c.NewVaultClient(cfg)
	if err != nil {
		return nil, err
	}

	_, err = c.initClient(ctx, vStore, client, cfg, vaultSpec)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Connector) prepareConfig(ctx context.Context, kube kclient.Client, corev1 typedcorev1.CoreV1Interface, vaultSpec *esv1beta1.VaultProvider, retrySettings *esv1beta1.SecretStoreRetrySettings, namespace, storeKind string) (*client, *vault.Config, error) {
	vStore := &client{
		kube:      kube,
		corev1:    corev1,
		store:     vaultSpec,
		log:       logger,
		namespace: namespace,
		storeKind: storeKind,
	}

	cfg, err := vStore.newConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Setup retry options if present
	if retrySettings != nil {
		if retrySettings.MaxRetries != nil {
			cfg.MaxRetries = int(*retrySettings.MaxRetries)
		} else {
			// By default we rely only on the reconciliation process for retrying
			cfg.MaxRetries = 0
		}

		if retrySettings.RetryInterval != nil {
			retryWait, err := time.ParseDuration(*retrySettings.RetryInterval)
			if err != nil {
				return nil, nil, err
			}
			cfg.MinRetryWait = retryWait
			cfg.MaxRetryWait = retryWait
		}
	}

	return vStore, cfg, nil
}

func (c *Connector) initClient(ctx context.Context, vStore *client, client util.Client, cfg *vault.Config, vaultSpec *esv1beta1.VaultProvider) (esv1beta1.SecretsClient, error) {
	if vaultSpec.Namespace != nil {
		client.SetNamespace(*vaultSpec.Namespace)
	}

	if vaultSpec.ReadYourWrites && vaultSpec.ForwardInconsistent {
		client.AddHeader("X-Vault-Inconsistent", "forward-active-node")
	}
	vStore.client = client
	vStore.auth = client.Auth()
	vStore.logical = client.Logical()
	vStore.token = client.AuthToken()

	// allow SecretStore controller validation to pass
	// when using referent namespace.
	if vStore.storeKind == esv1beta1.ClusterSecretStoreKind && vStore.namespace == "" && isReferentSpec(vaultSpec) {
		return vStore, nil
	}
	if err := vStore.setAuth(ctx, cfg); err != nil {
		return nil, err
	}

	return vStore, nil
}

func (c *Connector) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, fmt.Errorf(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, fmt.Errorf(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return nil, fmt.Errorf(errInvalidStoreProv)
	}
	p := spc.Provider.Vault
	if p == nil {
		return nil, fmt.Errorf(errInvalidVaultProv)
	}
	if p.Auth.AppRole != nil {
		// check SecretRef for valid configuration
		if err := utils.ValidateReferentSecretSelector(store, p.Auth.AppRole.SecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidAppRoleSec, err)
		}

		// prefer .auth.appRole.roleId, fallback to .auth.appRole.roleRef, give up after that.
		if p.Auth.AppRole.RoleID == "" { // prevents further RoleID tests if .auth.appRole.roleId is given
			if p.Auth.AppRole.RoleRef != nil { // check RoleRef for valid configuration
				if err := utils.ValidateReferentSecretSelector(store, *p.Auth.AppRole.RoleRef); err != nil {
					return nil, fmt.Errorf(errInvalidAppRoleRef, err)
				}
			} else { // we ran out of ways to get RoleID. return an appropriate error
				return nil, fmt.Errorf(errInvalidAppRoleID)
			}
		}
	}
	if p.Auth.Cert != nil {
		if err := utils.ValidateReferentSecretSelector(store, p.Auth.Cert.ClientCert); err != nil {
			return nil, fmt.Errorf(errInvalidClientCert, err)
		}
		if err := utils.ValidateReferentSecretSelector(store, p.Auth.Cert.SecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidCertSec, err)
		}
	}
	if p.Auth.Jwt != nil {
		if p.Auth.Jwt.SecretRef != nil {
			if err := utils.ValidateReferentSecretSelector(store, *p.Auth.Jwt.SecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidJwtSec, err)
			}
		} else if p.Auth.Jwt.KubernetesServiceAccountToken != nil {
			if err := utils.ValidateReferentServiceAccountSelector(store, p.Auth.Jwt.KubernetesServiceAccountToken.ServiceAccountRef); err != nil {
				return nil, fmt.Errorf(errInvalidJwtK8sSA, err)
			}
		} else {
			return nil, fmt.Errorf(errJwtNoTokenSource)
		}
	}
	if p.Auth.Kubernetes != nil {
		if p.Auth.Kubernetes.ServiceAccountRef != nil {
			if err := utils.ValidateReferentServiceAccountSelector(store, *p.Auth.Kubernetes.ServiceAccountRef); err != nil {
				return nil, fmt.Errorf(errInvalidKubeSA, err)
			}
		}
		if p.Auth.Kubernetes.SecretRef != nil {
			if err := utils.ValidateReferentSecretSelector(store, *p.Auth.Kubernetes.SecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidKubeSec, err)
			}
		}
	}
	if p.Auth.Ldap != nil {
		if err := utils.ValidateReferentSecretSelector(store, p.Auth.Ldap.SecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidLdapSec, err)
		}
	}
	if p.Auth.UserPass != nil {
		if err := utils.ValidateReferentSecretSelector(store, p.Auth.UserPass.SecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidUserPassSec, err)
		}
	}
	if p.Auth.TokenSecretRef != nil {
		if err := utils.ValidateReferentSecretSelector(store, *p.Auth.TokenSecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidTokenRef, err)
		}
	}
	if p.Auth.Iam != nil {
		if p.Auth.Iam.JWTAuth != nil {
			if p.Auth.Iam.JWTAuth.ServiceAccountRef != nil {
				if err := utils.ValidateReferentServiceAccountSelector(store, *p.Auth.Iam.JWTAuth.ServiceAccountRef); err != nil {
					return nil, fmt.Errorf(errInvalidTokenRef, err)
				}
			}
		}

		if p.Auth.Iam.SecretRef != nil {
			if err := utils.ValidateReferentSecretSelector(store, p.Auth.Iam.SecretRef.AccessKeyID); err != nil {
				return nil, fmt.Errorf(errInvalidTokenRef, err)
			}
			if err := utils.ValidateReferentSecretSelector(store, p.Auth.Iam.SecretRef.SecretAccessKey); err != nil {
				return nil, fmt.Errorf(errInvalidTokenRef, err)
			}
			if p.Auth.Iam.SecretRef.SessionToken != nil {
				if err := utils.ValidateReferentSecretSelector(store, *p.Auth.Iam.SecretRef.SessionToken); err != nil {
					return nil, fmt.Errorf(errInvalidTokenRef, err)
				}
			}
		}
	}
	if p.ClientTLS.CertSecretRef != nil && p.ClientTLS.KeySecretRef != nil {
		if err := utils.ValidateReferentSecretSelector(store, *p.ClientTLS.CertSecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidClientTLSCert, err)
		}
		if err := utils.ValidateReferentSecretSelector(store, *p.ClientTLS.KeySecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidClientTLSSecret, err)
		}
	} else if p.ClientTLS.CertSecretRef != nil || p.ClientTLS.KeySecretRef != nil {
		return nil, errors.New(errInvalidClientTLS)
	}
	return nil, nil
}

func (v *client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	path := v.buildPath(remoteRef.GetRemoteKey())
	metaPath, err := v.buildMetadataPath(remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}
	// Retrieve the secret map from vault and convert the secret value in string form.
	secretVal, err := v.readSecret(ctx, path, "")
	// If error is not of type secret not found, we should error
	if err != nil && errors.Is(err, esv1beta1.NoSecretError{}) {
		return nil
	}
	if err != nil {
		return err
	}
	metadata, err := v.readSecretMetadata(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}
	manager, ok := metadata["managed-by"]
	if !ok || manager != "external-secrets" {
		return nil
	}
	// If Push for a Property, we need to delete the property and update the secret
	if remoteRef.GetProperty() != "" {
		delete(secretVal, remoteRef.GetProperty())
		// If the only key left in the remote secret is the reference of the metadata.
		if v.store.Version == esv1beta1.VaultKVStoreV1 && len(secretVal) == 1 {
			delete(secretVal, "custom_metadata")
		}
		if len(secretVal) > 0 {
			secretToPush := secretVal
			if v.store.Version == esv1beta1.VaultKVStoreV2 {
				secretToPush = map[string]interface{}{
					"data": secretVal,
				}
			}
			_, err = v.logical.WriteWithContext(ctx, path, secretToPush)
			metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultDeleteSecret, err)
			return err
		}
	}
	_, err = v.logical.DeleteWithContext(ctx, path)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultDeleteSecret, err)
	if err != nil {
		return fmt.Errorf("could not delete secret %v: %w", remoteRef.GetRemoteKey(), err)
	}
	if v.store.Version == esv1beta1.VaultKVStoreV2 {
		_, err = v.logical.DeleteWithContext(ctx, metaPath)
		metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultDeleteSecret, err)
		if err != nil {
			return fmt.Errorf("could not delete secret metadata %v: %w", remoteRef.GetRemoteKey(), err)
		}
	}
	return nil
}

func (v *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	value := secret.Data[data.GetSecretKey()]
	label := map[string]interface{}{
		"custom_metadata": map[string]string{
			"managed-by": "external-secrets",
		},
	}
	secretVal := make(map[string]interface{})
	path := v.buildPath(data.GetRemoteKey())
	metaPath, err := v.buildMetadataPath(data.GetRemoteKey())
	if err != nil {
		return err
	}

	// Retrieve the secret map from vault and convert the secret value in string form.
	vaultSecret, err := v.readSecret(ctx, path, "")
	// If error is not of type secret not found, we should error
	if err != nil && !errors.Is(err, esv1beta1.NoSecretError{}) {
		return err
	}
	// If the secret exists (err == nil), we should check if it is managed by external-secrets
	if err == nil {
		metadata, err := v.readSecretMetadata(ctx, data.GetRemoteKey())
		if err != nil {
			return err
		}
		manager, ok := metadata["managed-by"]
		if !ok || manager != "external-secrets" {
			return fmt.Errorf("secret not managed by external-secrets")
		}
	}
	// Remove the metadata map to check the reconcile difference
	if v.store.Version == esv1beta1.VaultKVStoreV1 {
		delete(vaultSecret, "custom_metadata")
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err = enc.Encode(vaultSecret)
	if err != nil {
		return fmt.Errorf("error encoding vault secret: %w", err)
	}
	vaultSecretValue := bytes.TrimSpace(buf.Bytes())
	if err != nil {
		return fmt.Errorf("error marshaling vault secret: %w", err)
	}
	if bytes.Equal(vaultSecretValue, value) {
		return nil
	}
	// If a Push of a property only, we should merge and add/update the property
	if data.GetProperty() != "" {
		if _, ok := vaultSecret[data.GetProperty()]; ok {
			d := vaultSecret[data.GetProperty()].(string)
			if err != nil {
				return fmt.Errorf("error marshaling vault secret: %w", err)
			}
			// If the property has the same value, don't update the secret
			if bytes.Equal([]byte(d), value) {
				return nil
			}
		}
		for k, v := range vaultSecret {
			secretVal[k] = v
		}
		// Secret got from vault is already on map[string]string format
		secretVal[data.GetProperty()] = string(value)
	} else {
		err = json.Unmarshal(value, &secretVal)
		if err != nil {
			return fmt.Errorf("error unmarshalling vault secret: %w", err)
		}
	}
	secretToPush := secretVal
	// Adding custom_metadata to the secret for KV v1
	if v.store.Version == esv1beta1.VaultKVStoreV1 {
		secretToPush["custom_metadata"] = label["custom_metadata"]
	}
	if v.store.Version == esv1beta1.VaultKVStoreV2 {
		secretToPush = map[string]interface{}{
			"data": secretVal,
		}
	}
	if err != nil {
		return fmt.Errorf("failed to convert value to a valid JSON: %w", err)
	}
	// Secret metadata should be pushed separately only for KV2
	if v.store.Version == esv1beta1.VaultKVStoreV2 {
		_, err = v.logical.WriteWithContext(ctx, metaPath, label)
		metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
		if err != nil {
			return err
		}
	}
	// Otherwise, create or update the version.
	_, err = v.logical.WriteWithContext(ctx, path, secretToPush)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
	return err
}

// GetAllSecrets gets multiple secrets from the provider and loads into a kubernetes secret.
// First load all secrets from secretStore path configuration
// Then, gets secrets from a matching name or matching custom_metadata.
func (v *client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if v.store.Version == esv1beta1.VaultKVStoreV1 {
		return nil, errors.New(errUnsupportedKvVersion)
	}
	searchPath := ""
	if ref.Path != nil {
		searchPath = *ref.Path + "/"
	}
	potentialSecrets, err := v.listSecrets(ctx, searchPath)
	if err != nil {
		return nil, err
	}
	if ref.Name != nil {
		return v.findSecretsFromName(ctx, potentialSecrets, *ref.Name)
	}
	return v.findSecretsFromTags(ctx, potentialSecrets, ref.Tags)
}

func (v *client) findSecretsFromTags(ctx context.Context, candidates []string, tags map[string]string) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	for _, name := range candidates {
		match := true
		metadata, err := v.readSecretMetadata(ctx, name)
		if err != nil {
			return nil, err
		}
		for tk, tv := range tags {
			p, ok := metadata[tk]
			if !ok || p != tv {
				match = false
				break
			}
		}
		if match {
			secret, err := v.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: name})
			if errors.Is(err, esv1beta1.NoSecretError{}) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if secret != nil {
				secrets[name] = secret
			}
		}
	}
	return secrets, nil
}

func (v *client) findSecretsFromName(ctx context.Context, candidates []string, ref esv1beta1.FindName) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	matcher, err := find.New(ref)
	if err != nil {
		return nil, err
	}
	for _, name := range candidates {
		ok := matcher.MatchName(name)
		if ok {
			secret, err := v.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: name})
			if errors.Is(err, esv1beta1.NoSecretError{}) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if secret != nil {
				secrets[name] = secret
			}
		}
	}
	return secrets, nil
}

func (v *client) listSecrets(ctx context.Context, path string) ([]string, error) {
	secrets := make([]string, 0)
	url, err := v.buildMetadataPath(path)
	if err != nil {
		return nil, err
	}
	secret, err := v.logical.ListWithContext(ctx, url)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultListSecrets, err)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("provided path %v does not contain any secrets", url)
	}
	t, ok := secret.Data["keys"]
	if !ok {
		return nil, nil
	}
	paths := t.([]interface{})
	for _, p := range paths {
		strPath := p.(string)
		fullPath := path + strPath // because path always ends with a /
		if path == "" {
			fullPath = strPath
		}
		// Recurrently find secrets
		if !strings.HasSuffix(p.(string), "/") {
			secrets = append(secrets, fullPath)
		} else {
			partial, err := v.listSecrets(ctx, fullPath)
			if err != nil {
				return nil, err
			}
			secrets = append(secrets, partial...)
		}
	}
	return secrets, nil
}

func (v *client) readSecretMetadata(ctx context.Context, path string) (map[string]string, error) {
	metadata := make(map[string]string)
	url, err := v.buildMetadataPath(path)
	if err != nil {
		return nil, err
	}
	secret, err := v.logical.ReadWithDataWithContext(ctx, url, nil)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultReadSecretData, err)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
	}
	if secret == nil {
		return nil, errors.New(errNotFound)
	}
	t, ok := secret.Data["custom_metadata"]
	if !ok {
		return nil, nil
	}
	d, ok := t.(map[string]interface{})
	if !ok {
		return metadata, nil
	}
	for k, v := range d {
		metadata[k] = v.(string)
	}
	return metadata, nil
}

// GetSecret supports two types:
//  1. get the full secret as json-encoded value
//     by leaving the ref.Property empty.
//  2. get a key from the secret.
//     Nested values are supported by specifying a gjson expression
func (v *client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	var data map[string]interface{}
	var err error
	if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		if v.store.Version == esv1beta1.VaultKVStoreV1 {
			return nil, errors.New(errUnsupportedMetadataKvVersion)
		}

		metadata, err := v.readSecretMetadata(ctx, ref.Key)
		if err != nil {
			return nil, err
		}
		if len(metadata) == 0 {
			return nil, nil
		}
		data = make(map[string]interface{}, len(metadata))
		for k, v := range metadata {
			data[k] = v
		}
	} else {
		data, err = v.readSecret(ctx, ref.Key, ref.Version)
		if err != nil {
			return nil, err
		}
	}

	// Return nil if secret value is null
	if data == nil {
		return nil, esv1beta1.NoSecretError{}
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	// (1): return raw json if no property is defined
	if ref.Property == "" {
		return jsonStr, nil
	}

	// For backwards compatibility we want the
	// actual keys to take precedence over gjson syntax
	// (2): extract key from secret with property
	if _, ok := data[ref.Property]; ok {
		return utils.GetByteValueFromMap(data, ref.Property)
	}

	// (3): extract key from secret using gjson
	val := gjson.Get(string(jsonStr), ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf(errSecretKeyFmt, ref.Property)
	}
	return []byte(val.String()), nil
}

// GetSecretMap supports two modes of operation:
// 1. get the full secret from the vault data payload (by leaving .property empty).
// 2. extract key/value pairs from a (nested) object.
func (v *client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := v.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	var secretData map[string]interface{}
	err = json.Unmarshal(data, &secretData)
	if err != nil {
		return nil, err
	}
	byteMap := make(map[string][]byte, len(secretData))
	for k := range secretData {
		byteMap[k], err = utils.GetByteValueFromMap(secretData, k)
		if err != nil {
			return nil, err
		}
	}

	return byteMap, nil
}

func (v *client) Close(ctx context.Context) error {
	// Revoke the token if we have one set, it wasn't sourced from a TokenSecretRef,
	// and token caching isn't enabled
	if !enableCache && v.client.Token() != "" && v.store.Auth.TokenSecretRef == nil {
		err := revokeTokenIfValid(ctx, v.client)
		if err != nil {
			return err
		}
	}
	return nil
}

func isReferentSpec(prov *esv1beta1.VaultProvider) bool {
	if prov.Auth.TokenSecretRef != nil && prov.Auth.TokenSecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.AppRole != nil && prov.Auth.AppRole.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Kubernetes != nil && prov.Auth.Kubernetes.SecretRef != nil && prov.Auth.Kubernetes.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Kubernetes != nil && prov.Auth.Kubernetes.ServiceAccountRef != nil && prov.Auth.Kubernetes.ServiceAccountRef.Namespace == nil {
		return true
	}
	if prov.Auth.Ldap != nil && prov.Auth.Ldap.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.UserPass != nil && prov.Auth.UserPass.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Jwt != nil && prov.Auth.Jwt.SecretRef != nil && prov.Auth.Jwt.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Jwt != nil && prov.Auth.Jwt.KubernetesServiceAccountToken != nil && prov.Auth.Jwt.KubernetesServiceAccountToken.ServiceAccountRef.Namespace == nil {
		return true
	}
	if prov.Auth.Cert != nil && prov.Auth.Cert.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Iam != nil && prov.Auth.Iam.JWTAuth != nil && prov.Auth.Iam.JWTAuth.ServiceAccountRef != nil && prov.Auth.Iam.JWTAuth.ServiceAccountRef.Namespace == nil {
		return true
	}
	if prov.Auth.Iam != nil && prov.Auth.Iam.SecretRef != nil &&
		(prov.Auth.Iam.SecretRef.AccessKeyID.Namespace == nil ||
			prov.Auth.Iam.SecretRef.SecretAccessKey.Namespace == nil ||
			(prov.Auth.Iam.SecretRef.SessionToken != nil && prov.Auth.Iam.SecretRef.SessionToken.Namespace == nil)) {
		return true
	}
	return false
}

func (v *client) Validate() (esv1beta1.ValidationResult, error) {
	// when using referent namespace we can not validate the token
	// because the namespace is not known yet when Validate() is called
	// from the SecretStore controller.
	if v.storeKind == esv1beta1.ClusterSecretStoreKind && isReferentSpec(v.store) {
		return esv1beta1.ValidationResultUnknown, nil
	}
	_, err := checkToken(context.Background(), v.token)
	if err != nil {
		return esv1beta1.ValidationResultError, fmt.Errorf(errInvalidCredentials, err)
	}
	return esv1beta1.ValidationResultReady, nil
}

func (v *client) buildMetadataPath(path string) (string, error) {
	var url string
	if v.store.Version == esv1beta1.VaultKVStoreV1 {
		url = fmt.Sprintf("%s/%s", *v.store.Path, path)
	} else { // KV v2 is used
		if v.store.Path == nil && !strings.Contains(path, "data") {
			return "", fmt.Errorf(errPathInvalid)
		}
		if v.store.Path == nil {
			path = strings.Replace(path, "data", "metadata", 1)
			url = path
		} else {
			url = fmt.Sprintf("%s/metadata/%s", *v.store.Path, path)
		}
	}
	return url, nil
}

/*
	 buildPath is a helper method to build the vault equivalent path
		 from ExternalSecrets and SecretStore manifests. the path build logic
		 varies depending on the SecretStore KV version:
		 Example inputs/outputs:
		 # simple build:
		 kv version == "v2":
			provider_path: "secret/path"
			input: "foo"
			output: "secret/path/data/foo" # provider_path and data are prepended
		 kv version == "v1":
			provider_path: "secret/path"
			input: "foo"
			output: "secret/path/foo" # provider_path is prepended
		 # inheriting paths:
		 kv version == "v2":
			provider_path: "secret/path"
			input: "secret/path/foo"
			output: "secret/path/data/foo" #data is prepended
		 kv version == "v2":
			provider_path: "secret/path"
			input: "secret/path/data/foo"
			output: "secret/path/data/foo" #noop
		 kv version == "v1":
			provider_path: "secret/path"
			input: "secret/path/foo"
			output: "secret/path/foo" #noop
		 # provider path not defined:
		 kv version == "v2":
			provider_path: nil
			input: "secret/path/foo"
			output: "secret/data/path/foo" # data is prepended to secret/
		 kv version == "v2":
			provider_path: nil
			input: "secret/path/data/foo"
			output: "secret/path/data/foo" #noop
		 kv version == "v1":
			provider_path: nil
			input: "secret/path/foo"
			output: "secret/path/foo" #noop
*/
func (v *client) buildPath(path string) string {
	optionalMount := v.store.Path
	out := path
	// if optionalMount is Set, remove it from path if its there
	if optionalMount != nil {
		cut := *optionalMount + "/"
		if strings.HasPrefix(out, cut) {
			// This current logic induces a bug when the actual secret resides on same path names as the mount path.
			_, out, _ = strings.Cut(out, cut)
			// if data succeeds optionalMount on v2 store, we should remove it as well
			if strings.HasPrefix(out, "data/") && v.store.Version == esv1beta1.VaultKVStoreV2 {
				_, out, _ = strings.Cut(out, "data/")
			}
		}
		buildPath := strings.Split(out, "/")
		buildMount := strings.Split(*optionalMount, "/")
		if v.store.Version == esv1beta1.VaultKVStoreV2 {
			buildMount = append(buildMount, "data")
		}
		buildMount = append(buildMount, buildPath...)
		out = strings.Join(buildMount, "/")
		return out
	}
	if !strings.Contains(out, "/data/") && v.store.Version == esv1beta1.VaultKVStoreV2 {
		buildPath := strings.Split(out, "/")
		buildMount := []string{buildPath[0], "data"}
		buildMount = append(buildMount, buildPath[1:]...)
		out = strings.Join(buildMount, "/")
		return out
	}
	return out
}

func (v *client) readSecret(ctx context.Context, path, version string) (map[string]interface{}, error) {
	dataPath := v.buildPath(path)

	// path formated according to vault docs for v1 and v2 API
	// v1: https://www.vaultproject.io/api-docs/secret/kv/kv-v1#read-secret
	// v2: https://www.vaultproject.io/api/secret/kv/kv-v2#read-secret-version
	var params map[string][]string
	if version != "" {
		params = make(map[string][]string)
		params["version"] = []string{version}
	}
	vaultSecret, err := v.logical.ReadWithDataWithContext(ctx, dataPath, params)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultReadSecretData, err)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
	}
	if vaultSecret == nil {
		return nil, esv1beta1.NoSecretError{}
	}
	secretData := vaultSecret.Data
	if v.store.Version == esv1beta1.VaultKVStoreV2 {
		// Vault KV2 has data embedded within sub-field
		// reference - https://www.vaultproject.io/api/secret/kv/kv-v2#read-secret-version
		dataInt, ok := vaultSecret.Data["data"]
		if !ok {
			return nil, errors.New(errDataField)
		}
		if dataInt == nil {
			return nil, esv1beta1.NoSecretError{}
		}
		secretData, ok = dataInt.(map[string]interface{})
		if !ok {
			return nil, errors.New(errJSONUnmarshall)
		}
	}

	return secretData, nil
}

func (v *client) newConfig(ctx context.Context) (*vault.Config, error) {
	cfg := vault.DefaultConfig()
	cfg.Address = v.store.Server

	if len(v.store.CABundle) != 0 || v.store.CAProvider != nil {
		caCertPool := x509.NewCertPool()

		if len(v.store.CABundle) > 0 {
			ok := caCertPool.AppendCertsFromPEM(v.store.CABundle)
			if !ok {
				return nil, fmt.Errorf(errVaultCert, errors.New("failed to parse certificates from CertPool"))
			}
		}

		if v.store.CAProvider != nil && v.storeKind == esv1beta1.ClusterSecretStoreKind && v.store.CAProvider.Namespace == nil {
			return nil, errors.New(errCANamespace)
		}

		if v.store.CAProvider != nil {
			var cert []byte
			var err error

			switch v.store.CAProvider.Type {
			case esv1beta1.CAProviderTypeSecret:
				cert, err = getCertFromSecret(v)
			case esv1beta1.CAProviderTypeConfigMap:
				cert, err = getCertFromConfigMap(v)
			default:
				return nil, errors.New(errUnknownCAProvider)
			}

			if err != nil {
				return nil, err
			}

			ok := caCertPool.AppendCertsFromPEM(cert)
			if !ok {
				return nil, fmt.Errorf(errVaultCert, errors.New("failed to parse certificates from CertPool"))
			}
		}

		if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
			transport.TLSClientConfig.RootCAs = caCertPool
		}
	}

	err := v.configureClientTLS(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// If either read-after-write consistency feature is enabled, enable ReadYourWrites
	cfg.ReadYourWrites = v.store.ReadYourWrites || v.store.ForwardInconsistent

	return cfg, nil
}

func (v *client) configureClientTLS(ctx context.Context, cfg *vault.Config) error {
	clientTLS := v.store.ClientTLS
	if clientTLS.CertSecretRef != nil && clientTLS.KeySecretRef != nil {
		if clientTLS.KeySecretRef.Key == "" {
			clientTLS.KeySecretRef.Key = corev1.TLSPrivateKeyKey
		}
		clientKey, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, clientTLS.KeySecretRef)
		if err != nil {
			return err
		}

		if clientTLS.CertSecretRef.Key == "" {
			clientTLS.CertSecretRef.Key = corev1.TLSCertKey
		}
		clientCert, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, clientTLS.CertSecretRef)
		if err != nil {
			return err
		}

		cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
		if err != nil {
			return fmt.Errorf(errClientTLSAuth, err)
		}

		if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
			transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
		}
	}
	return nil
}

func getCertFromSecret(v *client) ([]byte, error) {
	secretRef := esmeta.SecretKeySelector{
		Name:      v.store.CAProvider.Name,
		Namespace: &v.namespace,
		Key:       v.store.CAProvider.Key,
	}

	if v.store.CAProvider.Namespace != nil {
		secretRef.Namespace = v.store.CAProvider.Namespace
	}

	ctx := context.Background()
	res, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, &secretRef)
	if err != nil {
		return nil, fmt.Errorf(errVaultCert, err)
	}

	return []byte(res), nil
}

func getCertFromConfigMap(v *client) ([]byte, error) {
	objKey := types.NamespacedName{
		Name:      v.store.CAProvider.Name,
		Namespace: v.namespace,
	}

	if v.store.CAProvider.Namespace != nil {
		objKey.Namespace = *v.store.CAProvider.Namespace
	}

	configMapRef := &corev1.ConfigMap{}
	ctx := context.Background()
	err := v.kube.Get(ctx, objKey, configMapRef)
	if err != nil {
		return nil, fmt.Errorf(errVaultCert, err)
	}

	val, ok := configMapRef.Data[v.store.CAProvider.Key]
	if !ok {
		return nil, fmt.Errorf(errConfigMapFmt, v.store.CAProvider.Key)
	}

	return []byte(val), nil
}

/*
setAuth gets a new token using the configured mechanism.
If there's already a valid token, does nothing.
*/
func (v *client) setAuth(ctx context.Context, cfg *vault.Config) error {
	tokenExists := false
	var err error
	if v.client.Token() != "" {
		tokenExists, err = checkToken(ctx, v.token)
	}
	if tokenExists {
		v.log.V(1).Info("Re-using existing token")
		return err
	}

	tokenExists, err = setSecretKeyToken(ctx, v)
	if tokenExists {
		v.log.V(1).Info("Set token from secret")
		return err
	}

	tokenExists, err = setAppRoleToken(ctx, v)
	if tokenExists {
		v.log.V(1).Info("Retrieved new token using AppRole auth")
		return err
	}

	tokenExists, err = setKubernetesAuthToken(ctx, v)
	if tokenExists {
		v.log.V(1).Info("Retrieved new token using Kubernetes auth")
		return err
	}

	tokenExists, err = setLdapAuthToken(ctx, v)
	if tokenExists {
		v.log.V(1).Info("Retrieved new token using LDAP auth")
		return err
	}

	tokenExists, err = setUserPassAuthToken(ctx, v)
	if tokenExists {
		v.log.V(1).Info("Retrieved new token using userPass auth")
		return err
	}
	tokenExists, err = setJwtAuthToken(ctx, v)
	if tokenExists {
		v.log.V(1).Info("Retrieved new token using JWT auth")
		return err
	}

	tokenExists, err = setCertAuthToken(ctx, v, cfg)
	if tokenExists {
		v.log.V(1).Info("Retrieved new token using certificate auth")
		return err
	}

	tokenExists, err = setIamAuthToken(ctx, v, vaultiamauth.DefaultJWTProvider, vaultiamauth.DefaultSTSProvider)
	if tokenExists {
		v.log.V(1).Info("Retrieved new token using IAM auth")
		return err
	}

	return errors.New(errAuthFormat)
}

func setSecretKeyToken(ctx context.Context, v *client) (bool, error) {
	tokenRef := v.store.Auth.TokenSecretRef
	if tokenRef != nil {
		token, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, tokenRef)
		if err != nil {
			return true, err
		}
		v.client.SetToken(token)
		return true, nil
	}
	return false, nil
}

func setAppRoleToken(ctx context.Context, v *client) (bool, error) {
	appRole := v.store.Auth.AppRole
	if appRole != nil {
		err := v.requestTokenWithAppRoleRef(ctx, appRole)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setKubernetesAuthToken(ctx context.Context, v *client) (bool, error) {
	kubernetesAuth := v.store.Auth.Kubernetes
	if kubernetesAuth != nil {
		err := v.requestTokenWithKubernetesAuth(ctx, kubernetesAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setLdapAuthToken(ctx context.Context, v *client) (bool, error) {
	ldapAuth := v.store.Auth.Ldap
	if ldapAuth != nil {
		err := v.requestTokenWithLdapAuth(ctx, ldapAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setUserPassAuthToken(ctx context.Context, v *client) (bool, error) {
	userPassAuth := v.store.Auth.UserPass
	if userPassAuth != nil {
		err := v.requestTokenWithUserPassAuth(ctx, userPassAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setJwtAuthToken(ctx context.Context, v *client) (bool, error) {
	jwtAuth := v.store.Auth.Jwt
	if jwtAuth != nil {
		err := v.requestTokenWithJwtAuth(ctx, jwtAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setCertAuthToken(ctx context.Context, v *client, cfg *vault.Config) (bool, error) {
	certAuth := v.store.Auth.Cert
	if certAuth != nil {
		err := v.requestTokenWithCertAuth(ctx, certAuth, cfg)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setIamAuthToken(ctx context.Context, v *client, jwtProvider util.JwtProviderFactory, assumeRoler vaultiamauth.STSProvider) (bool, error) {
	iamAuth := v.store.Auth.Iam
	isClusterKind := v.storeKind == esv1beta1.ClusterSecretStoreKind
	if iamAuth != nil {
		err := v.requestTokenWithIamAuth(ctx, iamAuth, isClusterKind, v.kube, v.namespace, jwtProvider, assumeRoler)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (v *client) secretKeyRefForServiceAccount(ctx context.Context, serviceAccountRef *esmeta.ServiceAccountSelector) (string, error) {
	serviceAccount := &corev1.ServiceAccount{}
	ref := types.NamespacedName{
		Namespace: v.namespace,
		Name:      serviceAccountRef.Name,
	}
	if (v.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		ref.Namespace = *serviceAccountRef.Namespace
	}
	err := v.kube.Get(ctx, ref, serviceAccount)
	if err != nil {
		return "", fmt.Errorf(errGetKubeSA, ref.Name, err)
	}
	if len(serviceAccount.Secrets) == 0 {
		return "", fmt.Errorf(errGetKubeSASecrets, ref.Name)
	}
	for _, tokenRef := range serviceAccount.Secrets {
		token, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, &esmeta.SecretKeySelector{
			Name:      tokenRef.Name,
			Namespace: &ref.Namespace,
			Key:       "token",
		})
		if err != nil {
			continue
		}
		return token, nil
	}
	return "", fmt.Errorf(errGetKubeSANoToken, ref.Name)
}

// checkToken does a lookup and checks if the provided token exists.
func checkToken(ctx context.Context, token util.Token) (bool, error) {
	// https://www.vaultproject.io/api-docs/auth/token#lookup-a-token-self
	resp, err := token.LookupSelfWithContext(ctx)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLookupSelf, err)
	if err != nil {
		return false, err
	}
	t, ok := resp.Data["type"]
	if !ok {
		return false, fmt.Errorf("could not assert token type")
	}
	tokenType := t.(string)
	if tokenType == "batch" {
		return false, nil
	}
	return true, nil
}

func revokeTokenIfValid(ctx context.Context, client util.Client) error {
	valid, err := checkToken(ctx, client.AuthToken())
	if err != nil {
		return fmt.Errorf(errVaultRevokeToken, err)
	}
	if valid {
		err = client.AuthToken().RevokeSelfWithContext(ctx, client.Token())
		metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultRevokeSelf, err)
		if err != nil {
			return fmt.Errorf(errVaultRevokeToken, err)
		}
		client.ClearToken()
	}
	return nil
}

func (v *client) requestTokenWithAppRoleRef(ctx context.Context, appRole *esv1beta1.VaultAppRole) error {
	var err error
	var roleID string // becomes the RoleID used to authenticate with HashiCorp Vault

	// prefer .auth.appRole.roleId, fallback to .auth.appRole.roleRef, give up after that.
	if appRole.RoleID != "" { // use roleId from CRD, if configured
		roleID = strings.TrimSpace(appRole.RoleID)
	} else if appRole.RoleRef != nil { // use RoleID from Secret, if configured
		roleID, err = resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, appRole.RoleRef)
		if err != nil {
			return err
		}
	} else { // we ran out of ways to get RoleID. return an appropriate error
		return fmt.Errorf(errInvalidAppRoleID)
	}

	secretID, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, &appRole.SecretRef)
	if err != nil {
		return err
	}
	secret := approle.SecretID{FromString: secretID}
	appRoleClient, err := approle.NewAppRoleAuth(roleID, &secret, approle.WithMountPath(appRole.Path))
	if err != nil {
		return err
	}
	_, err = v.auth.Login(ctx, appRoleClient)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}
	return nil
}

func (v *client) requestTokenWithKubernetesAuth(ctx context.Context, kubernetesAuth *esv1beta1.VaultKubernetesAuth) error {
	jwtString, err := getJwtString(ctx, v, kubernetesAuth)
	if err != nil {
		return err
	}
	k, err := authkubernetes.NewKubernetesAuth(kubernetesAuth.Role, authkubernetes.WithServiceAccountToken(jwtString), authkubernetes.WithMountPath(kubernetesAuth.Path))
	if err != nil {
		return err
	}
	_, err = v.auth.Login(ctx, k)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}
	return nil
}

func getJwtString(ctx context.Context, v *client, kubernetesAuth *esv1beta1.VaultKubernetesAuth) (string, error) {
	if kubernetesAuth.ServiceAccountRef != nil {
		// Kubernetes <v1.24 fetch token via ServiceAccount.Secrets[]
		// this behavior was removed in v1.24 and we must use TokenRequest API (see below)
		jwt, err := v.secretKeyRefForServiceAccount(ctx, kubernetesAuth.ServiceAccountRef)
		if jwt != "" {
			return jwt, err
		}
		if err != nil {
			v.log.V(1).Info("unable to fetch jwt from service account secret, trying service account token next")
		}
		// Kubernetes >=v1.24: fetch token via TokenRequest API
		// note: this is a massive change from vault perspective: the `iss` claim will very likely change.
		// Vault 1.9 deprecated issuer validation by default, and authentication with Vault clusters <1.9 will likely fail.
		jwt, err = createServiceAccountToken(
			ctx,
			v.corev1,
			v.storeKind,
			v.namespace,
			*kubernetesAuth.ServiceAccountRef,
			nil,
			600)
		if err != nil {
			return "", err
		}
		return jwt, nil
	} else if kubernetesAuth.SecretRef != nil {
		tokenRef := kubernetesAuth.SecretRef
		if tokenRef.Key == "" {
			tokenRef = kubernetesAuth.SecretRef.DeepCopy()
			tokenRef.Key = "token"
		}
		jwt, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, tokenRef)
		if err != nil {
			return "", err
		}
		return jwt, nil
	} else {
		// Kubernetes authentication is specified, but without a referenced
		// Kubernetes secret. We check if the file path for in-cluster service account
		// exists and attempt to use the token for Vault Kubernetes auth.
		if _, err := os.Stat(serviceAccTokenPath); err != nil {
			return "", fmt.Errorf(errServiceAccount, err)
		}
		jwtByte, err := os.ReadFile(serviceAccTokenPath)
		if err != nil {
			return "", fmt.Errorf(errServiceAccount, err)
		}
		return string(jwtByte), nil
	}
}

func (v *client) requestTokenWithLdapAuth(ctx context.Context, ldapAuth *esv1beta1.VaultLdapAuth) error {
	username := strings.TrimSpace(ldapAuth.Username)
	password, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, &ldapAuth.SecretRef)
	if err != nil {
		return err
	}
	pass := authldap.Password{FromString: password}
	l, err := authldap.NewLDAPAuth(username, &pass, authldap.WithMountPath(ldapAuth.Path))
	if err != nil {
		return err
	}
	_, err = v.auth.Login(ctx, l)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}
	return nil
}

func (v *client) requestTokenWithUserPassAuth(ctx context.Context, userPassAuth *esv1beta1.VaultUserPassAuth) error {
	username := strings.TrimSpace(userPassAuth.Username)
	password, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, &userPassAuth.SecretRef)
	if err != nil {
		return err
	}
	pass := authuserpass.Password{FromString: password}
	l, err := authuserpass.NewUserpassAuth(username, &pass, authuserpass.WithMountPath(userPassAuth.Path))
	if err != nil {
		return err
	}
	_, err = v.auth.Login(ctx, l)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}
	return nil
}

func (v *client) requestTokenWithJwtAuth(ctx context.Context, jwtAuth *esv1beta1.VaultJwtAuth) error {
	role := strings.TrimSpace(jwtAuth.Role)
	var jwt string
	var err error
	if jwtAuth.SecretRef != nil {
		jwt, err = resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, jwtAuth.SecretRef)
	} else if k8sServiceAccountToken := jwtAuth.KubernetesServiceAccountToken; k8sServiceAccountToken != nil {
		audiences := k8sServiceAccountToken.Audiences
		if audiences == nil {
			audiences = &[]string{"vault"}
		}
		expirationSeconds := k8sServiceAccountToken.ExpirationSeconds
		if expirationSeconds == nil {
			tmp := int64(600)
			expirationSeconds = &tmp
		}
		jwt, err = createServiceAccountToken(
			ctx,
			v.corev1,
			v.storeKind,
			v.namespace,
			k8sServiceAccountToken.ServiceAccountRef,
			*audiences,
			*expirationSeconds)
	} else {
		err = fmt.Errorf(errJwtNoTokenSource)
	}
	if err != nil {
		return err
	}

	parameters := map[string]interface{}{
		"role": role,
		"jwt":  jwt,
	}
	url := strings.Join([]string{"auth", jwtAuth.Path, "login"}, "/")
	vaultResult, err := v.logical.WriteWithContext(ctx, url, parameters)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
	if err != nil {
		return err
	}

	token, err := vaultResult.TokenID()
	if err != nil {
		return fmt.Errorf(errVaultToken, err)
	}
	v.client.SetToken(token)
	return nil
}

func (v *client) requestTokenWithCertAuth(ctx context.Context, certAuth *esv1beta1.VaultCertAuth, cfg *vault.Config) error {
	clientKey, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, &certAuth.SecretRef)
	if err != nil {
		return err
	}

	clientCert, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, &certAuth.ClientCert)
	if err != nil {
		return err
	}

	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return fmt.Errorf(errClientTLSAuth, err)
	}

	if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	url := strings.Join([]string{"auth", "cert", "login"}, "/")
	vaultResult, err := v.logical.WriteWithContext(ctx, url, nil)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
	if err != nil {
		return fmt.Errorf(errVaultRequest, err)
	}
	token, err := vaultResult.TokenID()
	if err != nil {
		return fmt.Errorf(errVaultToken, err)
	}
	v.client.SetToken(token)
	return nil
}

func createServiceAccountToken(
	ctx context.Context,
	corev1Client typedcorev1.CoreV1Interface,
	storeKind string,
	namespace string,
	serviceAccountRef esmeta.ServiceAccountSelector,
	additionalAud []string,
	expirationSeconds int64) (string, error) {
	audiences := serviceAccountRef.Audiences
	if len(additionalAud) > 0 {
		audiences = append(audiences, additionalAud...)
	}
	tokenRequest := &authv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: authv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}
	if (storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		tokenRequest.Namespace = *serviceAccountRef.Namespace
	}
	tokenResponse, err := corev1Client.ServiceAccounts(tokenRequest.Namespace).
		CreateToken(ctx, serviceAccountRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf(errGetKubeSATokenRequest, serviceAccountRef.Name, err)
	}
	return tokenResponse.Status.Token, nil
}

func (v *client) requestTokenWithIamAuth(ctx context.Context, iamAuth *esv1beta1.VaultIamAuth, isClusterKind bool, k kclient.Client, n string, jwtProvider util.JwtProviderFactory, assumeRoler vaultiamauth.STSProvider) error {
	jwtAuth := iamAuth.JWTAuth
	secretRefAuth := iamAuth.SecretRef
	regionAWS := defaultAWSRegion
	awsAuthMountPath := defaultAWSAuthMountPath
	if iamAuth.Region != "" {
		regionAWS = iamAuth.Region
	}
	if iamAuth.Path != "" {
		awsAuthMountPath = iamAuth.Path
	}
	var creds *credentials.Credentials
	var err error
	if jwtAuth != nil { // use credentials from a sa explicitly defined and referenced. Highest preference is given to this method/configuration.
		creds, err = vaultiamauth.CredsFromServiceAccount(ctx, *iamAuth, regionAWS, isClusterKind, k, n, jwtProvider)
		if err != nil {
			return err
		}
	} else if secretRefAuth != nil { // if jwtAuth is not defined, check if secretRef is defined. Second preference.
		logger.V(1).Info("using credentials from secretRef")
		creds, err = vaultiamauth.CredsFromSecretRef(ctx, *iamAuth, v.storeKind, k, n)
		if err != nil {
			return err
		}
	}

	// Neither of jwtAuth or secretRefAuth defined. Last preference.
	// Default to controller pod's identity
	if jwtAuth == nil && secretRefAuth == nil {
		// Checking if controller pod's service account is IRSA enabled and Web Identity token is available on pod
		tknFile, tknFileEnvVarPresent := os.LookupEnv(vaultiamauth.AWSWebIdentityTokenFileEnvVar)
		if !tknFileEnvVarPresent {
			return fmt.Errorf(errIrsaTokenEnvVarNotFoundOnPod, vaultiamauth.AWSWebIdentityTokenFileEnvVar) // No Web Identity(IRSA) token found on pod
		}

		// IRSA enabled service account, let's check that the jwt token filemount and file exists
		if _, err := os.Stat(tknFile); err != nil {
			return fmt.Errorf(errIrsaTokenFileNotFoundOnPod, tknFile, err)
		}

		// everything looks good so far, let's fetch the jwt token from AWS_WEB_IDENTITY_TOKEN_FILE
		jwtByte, err := os.ReadFile(tknFile)
		if err != nil {
			return fmt.Errorf(errIrsaTokenFileNotReadable, tknFile, err)
		}

		// let's parse the jwt token
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())

		token, _, err := parser.ParseUnverified(string(jwtByte), jwt.MapClaims{})
		if err != nil {
			return fmt.Errorf(errIrsaTokenNotValidJWT, tknFile, err) // JWT token parser error
		}

		var ns string
		var sa string

		// let's fetch the namespace and serviceaccount from parsed jwt token
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			ns = claims["kubernetes.io"].(map[string]interface{})["namespace"].(string)
			sa = claims["kubernetes.io"].(map[string]interface{})["serviceaccount"].(map[string]interface{})["name"].(string)
		} else {
			return fmt.Errorf(errPodInfoNotFoundOnToken, tknFile, err)
		}

		creds, err = vaultiamauth.CredsFromControllerServiceAccount(ctx, sa, ns, regionAWS, k, jwtProvider)
		if err != nil {
			return err
		}
	}

	config := aws.NewConfig().WithEndpointResolver(vaultiamauth.ResolveEndpoint())
	if creds != nil {
		config.WithCredentials(creds)
	}

	if regionAWS != "" {
		config.WithRegion(regionAWS)
	}

	sess, err := vaultiamauth.GetAWSSession(config)
	if err != nil {
		return err
	}
	if iamAuth.AWSIAMRole != "" {
		stsclient := assumeRoler(sess)
		if iamAuth.ExternalID != "" {
			var setExternalID = func(p *stscreds.AssumeRoleProvider) {
				p.ExternalID = aws.String(iamAuth.ExternalID)
			}
			sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, iamAuth.AWSIAMRole, setExternalID))
		} else {
			sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, iamAuth.AWSIAMRole))
		}
	}

	getCreds, err := sess.Config.Credentials.Get()
	if err != nil {
		return err
	}
	// Set environment variables. These would be fetched by Login
	os.Setenv("AWS_ACCESS_KEY_ID", getCreds.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", getCreds.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", getCreds.SessionToken)

	var awsAuthClient *authaws.AWSAuth

	if iamAuth.VaultAWSIAMServerID != "" {
		awsAuthClient, err = authaws.NewAWSAuth(authaws.WithRegion(regionAWS), authaws.WithIAMAuth(), authaws.WithRole(iamAuth.Role), authaws.WithMountPath(awsAuthMountPath), authaws.WithIAMServerIDHeader(iamAuth.VaultAWSIAMServerID))
		if err != nil {
			return err
		}
	} else {
		awsAuthClient, err = authaws.NewAWSAuth(authaws.WithRegion(regionAWS), authaws.WithIAMAuth(), authaws.WithRole(iamAuth.Role), authaws.WithMountPath(awsAuthMountPath))
		if err != nil {
			return err
		}
	}

	_, err = v.auth.Login(ctx, awsAuthClient)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	var vaultTokenCacheSize int
	fs := pflag.NewFlagSet("vault", pflag.ExitOnError)
	fs.BoolVar(&enableCache, "experimental-enable-vault-token-cache", false, "Enable experimental Vault token cache. External secrets will reuse the Vault token without creating a new one on each request.")
	// max. 265k vault leases with 30bytes each ~= 7MB
	fs.IntVar(&vaultTokenCacheSize, "experimental-vault-token-cache-size", 2<<17, "Maximum size of Vault token cache. When more tokens than Only used if --experimental-enable-vault-token-cache is set.")
	lateInit := func() {
		logger.Info("initializing vault cache", "size", vaultTokenCacheSize)
		clientCache = cache.Must(vaultTokenCacheSize, func(client util.Client) {
			err := revokeTokenIfValid(context.Background(), client)
			if err != nil {
				logger.Error(err, "unable to revoke cached token on eviction")
			}
		})
	}
	feature.Register(feature.Feature{
		Flags:      fs,
		Initialize: lateInit,
	})

	esv1beta1.Register(&Connector{
		NewVaultClient: NewVaultClient,
	}, &esv1beta1.SecretStoreProvider{
		Vault: &esv1beta1.VaultProvider{},
	})
}
