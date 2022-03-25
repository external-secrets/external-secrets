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
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

var (
	_ esv1beta1.Provider      = &connector{}
	_ esv1beta1.SecretsClient = &client{}
)

const (
	serviceAccTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	errVaultStore           = "received invalid Vault SecretStore resource: %w"
	errVaultClient          = "cannot setup new vault client: %w"
	errVaultCert            = "cannot set Vault CA certificate: %w"
	errReadSecret           = "cannot read secret data from Vault: %w"
	errAuthFormat           = "cannot initialize Vault client: no valid auth method specified"
	errInvalidCredentials   = "invalid vault credentials: %w"
	errDataField            = "failed to find data field"
	errJSONUnmarshall       = "failed to unmarshall JSON"
	errPathInvalid          = "provided Path isn't a valid kv v2 path"
	errSecretFormat         = "secret data not in expected format"
	errUnexpectedKey        = "unexpected key in data: %s"
	errVaultToken           = "cannot parse Vault authentication token: %w"
	errVaultReqParams       = "cannot set Vault request parameters: %w"
	errVaultRequest         = "error from Vault request: %w"
	errVaultResponse        = "cannot parse Vault response: %w"
	errServiceAccount       = "cannot read Kubernetes service account token from file system: %w"
	errUnsupportedKvVersion = "cannot perform find operations with kv version v1"
	errGetKubeSA            = "cannot get Kubernetes service account %q: %w"
	errGetKubeSASecrets     = "cannot find secrets bound to service account: %q"
	errGetKubeSANoToken     = "cannot find token in secrets bound to service account: %q"

	errGetKubeSecret = "cannot get Kubernetes secret %q: %w"
	errSecretKeyFmt  = "cannot find secret data for key: %q"
	errConfigMapFmt  = "cannot find config map data for key: %q"

	errClientTLSAuth = "error from Client TLS Auth: %q"

	errVaultRevokeToken = "error while revoking token: %w"

	errUnknownCAProvider = "unknown caProvider type given"
	errCANamespace       = "cannot read secret for CAProvider due to missing namespace on kind ClusterSecretStore"

	errInvalidStore      = "invalid store"
	errInvalidStoreSpec  = "invalid store spec"
	errInvalidStoreProv  = "invalid store provider"
	errInvalidVaultProv  = "invalid vault provider"
	errInvalidAppRoleSec = "invalid Auth.AppRole.SecretRef: %w"
	errInvalidClientCert = "invalid Auth.Cert.ClientCert: %w"
	errInvalidCertSec    = "invalid Auth.Cert.SecretRef: %w"
	errInvalidJwtSec     = "invalid Auth.Jwt.SecretRef: %w"
	errInvalidKubeSA     = "invalid Auth.Kubernetes.ServiceAccountRef: %w"
	errInvalidKubeSec    = "invalid Auth.Kubernetes.SecretRef: %w"
	errInvalidLdapSec    = "invalid Auth.Ldap.SecretRef: %w"
	errInvalidTokenRef   = "invalid Auth.TokenSecretRef: %w"
)

type Client interface {
	NewRequest(method, requestPath string) *vault.Request
	RawRequestWithContext(ctx context.Context, r *vault.Request) (*vault.Response, error)
	SetToken(v string)
	Token() string
	ClearToken()
	SetNamespace(namespace string)
	AddHeader(key, value string)
}

type client struct {
	kube      kclient.Client
	store     *esv1beta1.VaultProvider
	log       logr.Logger
	client    Client
	namespace string
	storeKind string
}

func init() {
	esv1beta1.Register(&connector{
		newVaultClient: newVaultClient,
	}, &esv1beta1.SecretStoreProvider{
		Vault: &esv1beta1.VaultProvider{},
	})
}

func newVaultClient(c *vault.Config) (Client, error) {
	return vault.NewClient(c)
}

type connector struct {
	newVaultClient func(c *vault.Config) (Client, error)
}

func (c *connector) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Vault == nil {
		return nil, errors.New(errVaultStore)
	}
	vaultSpec := storeSpec.Provider.Vault

	vStore := &client{
		kube:      kube,
		store:     vaultSpec,
		log:       ctrl.Log.WithName("provider").WithName("vault"),
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	cfg, err := vStore.newConfig()
	if err != nil {
		return nil, err
	}

	client, err := c.newVaultClient(cfg)
	if err != nil {
		return nil, fmt.Errorf(errVaultClient, err)
	}

	if vaultSpec.Namespace != nil {
		client.SetNamespace(*vaultSpec.Namespace)
	}

	if vaultSpec.ReadYourWrites && vaultSpec.ForwardInconsistent {
		client.AddHeader("X-Vault-Inconsistent", "forward-active-node")
	}

	if err := vStore.setAuth(ctx, client, cfg); err != nil {
		return nil, err
	}

	vStore.client = client

	return vStore, nil
}

func (c *connector) ValidateStore(store esv1beta1.GenericStore) error {
	if store == nil {
		return fmt.Errorf(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return fmt.Errorf(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return fmt.Errorf(errInvalidStoreProv)
	}
	p := spc.Provider.Vault
	if p == nil {
		return fmt.Errorf(errInvalidVaultProv)
	}
	if p.Auth.AppRole != nil {
		if err := utils.ValidateSecretSelector(store, p.Auth.AppRole.SecretRef); err != nil {
			return fmt.Errorf(errInvalidAppRoleSec, err)
		}
	}
	if p.Auth.Cert != nil {
		if err := utils.ValidateSecretSelector(store, p.Auth.Cert.ClientCert); err != nil {
			return fmt.Errorf(errInvalidClientCert, err)
		}
		if err := utils.ValidateSecretSelector(store, p.Auth.Cert.SecretRef); err != nil {
			return fmt.Errorf(errInvalidCertSec, err)
		}
	}
	if p.Auth.Jwt != nil {
		if err := utils.ValidateSecretSelector(store, p.Auth.Jwt.SecretRef); err != nil {
			return fmt.Errorf(errInvalidJwtSec, err)
		}
	}
	if p.Auth.Kubernetes != nil {
		if p.Auth.Kubernetes.ServiceAccountRef != nil {
			if err := utils.ValidateServiceAccountSelector(store, *p.Auth.Kubernetes.ServiceAccountRef); err != nil {
				return fmt.Errorf(errInvalidKubeSA, err)
			}
		}
		if p.Auth.Kubernetes.SecretRef != nil {
			if err := utils.ValidateSecretSelector(store, *p.Auth.Kubernetes.SecretRef); err != nil {
				return fmt.Errorf(errInvalidKubeSec, err)
			}
		}
	}
	if p.Auth.Ldap != nil {
		if err := utils.ValidateSecretSelector(store, p.Auth.Ldap.SecretRef); err != nil {
			return fmt.Errorf(errInvalidLdapSec, err)
		}
	}
	if p.Auth.TokenSecretRef != nil {
		if err := utils.ValidateSecretSelector(store, *p.Auth.TokenSecretRef); err != nil {
			return fmt.Errorf(errInvalidTokenRef, err)
		}
	}
	return nil
}

// Empty GetAllSecrets.
// GetAllSecrets
// First load all secrets from secretStore path configuration.
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
		return v.findSecretsFromName(ctx, potentialSecrets, *ref.Name, searchPath)
	}
	return v.findSecretsFromTags(ctx, potentialSecrets, ref.Tags, searchPath)
}

func (v *client) findSecretsFromTags(ctx context.Context, candidates []string, tags map[string]string, removeFromName string) (map[string][]byte, error) {
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
			if err != nil {
				return nil, err
			}
			if removeFromName != "" {
				name = strings.TrimPrefix(name, removeFromName)
			}
			secrets[name] = secret
		}
	}
	return secrets, nil
}

func (v *client) findSecretsFromName(ctx context.Context, candidates []string, ref esv1beta1.FindName, removeFromName string) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	matcher, err := find.New(ref)
	if err != nil {
		return nil, err
	}
	for _, name := range candidates {
		ok := matcher.MatchName(name)
		if ok {
			secret, err := v.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: name})
			if err != nil {
				return nil, err
			}
			if removeFromName != "" {
				name = strings.TrimPrefix(name, removeFromName)
			}
			secrets[name] = secret
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
	r := v.client.NewRequest(http.MethodGet, url)
	r.Params.Set("list", "true")
	resp, err := v.client.RawRequestWithContext(ctx, r)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
	}
	secret, parseErr := vault.ParseSecret(resp.Body)
	if parseErr != nil {
		return nil, parseErr
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
	r := v.client.NewRequest(http.MethodGet, url)
	resp, err := v.client.RawRequestWithContext(ctx, r)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
	}
	secret, parseErr := vault.ParseSecret(resp.Body)
	if parseErr != nil {
		return nil, parseErr
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
// 1. get the full secret as json-encoded value
//    by leaving the ref.Property empty.
// 2. get a key from the secret.
//    Nested values are supported by specifying a gjson expression
func (v *client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	data, err := v.readSecret(ctx, ref.Key, ref.Version)
	if err != nil {
		return nil, err
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
		return getTypedKey(data, ref.Property)
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
		byteMap[k], err = getTypedKey(secretData, k)
		if err != nil {
			return nil, err
		}
	}

	return byteMap, nil
}

func getTypedKey(data map[string]interface{}, key string) ([]byte, error) {
	v, ok := data[key]
	if !ok {
		return nil, fmt.Errorf(errUnexpectedKey, key)
	}
	switch t := v.(type) {
	case string:
		return []byte(t), nil
	case map[string]interface{}:
		return json.Marshal(t)
	case []byte:
		return t, nil
	// also covers int and float32 due to json.Marshal
	case float64:
		return []byte(strconv.FormatFloat(t, 'f', -1, 64)), nil
	case bool:
		return []byte(strconv.FormatBool(t)), nil
	case nil:
		return []byte(nil), nil
	default:
		return nil, errors.New(errSecretFormat)
	}
}

func (v *client) Close(ctx context.Context) error {
	// Revoke the token if we have one set and it wasn't sourced from a TokenSecretRef
	if v.client.Token() != "" && v.store.Auth.TokenSecretRef == nil {
		req := v.client.NewRequest(http.MethodPost, "/v1/auth/token/revoke-self")
		_, err := v.client.RawRequestWithContext(ctx, req)
		if err != nil {
			return fmt.Errorf(errVaultRevokeToken, err)
		}
		v.client.ClearToken()
	}
	return nil
}

func (v *client) Validate() error {
	err := checkToken(context.Background(), v)
	if err != nil {
		return fmt.Errorf(errInvalidCredentials, err)
	}
	return nil
}

func (v *client) buildMetadataPath(path string) (string, error) {
	var url string
	if v.store.Path == nil && !strings.Contains(path, "data") {
		return "", fmt.Errorf(errPathInvalid)
	}
	if v.store.Path == nil {
		path = strings.Replace(path, "data", "metadata", 1)
		url = fmt.Sprintf("/v1/%s", path)
	} else {
		url = fmt.Sprintf("/v1/%s/metadata/%s", *v.store.Path, path)
	}
	return url, nil
}
func (v *client) buildPath(path string) string {
	optionalMount := v.store.Path
	origPath := strings.Split(path, "/")
	newPath := make([]string, 0)
	cursor := 0

	if optionalMount != nil && origPath[0] != *optionalMount {
		// Default case before path was optional
		// Ensure that the requested path includes the SecretStores paths as prefix
		newPath = append(newPath, *optionalMount)
	} else {
		newPath = append(newPath, origPath[cursor])
		cursor++
	}

	if v.store.Version == esv1beta1.VaultKVStoreV2 {
		// Add the required `data` part of the URL for the v2 API
		if len(origPath) < 2 || origPath[1] != "data" {
			newPath = append(newPath, "data")
		}
	}
	newPath = append(newPath, origPath[cursor:]...)
	returnPath := strings.Join(newPath, "/")

	return returnPath
}

func (v *client) readSecret(ctx context.Context, path, version string) (map[string]interface{}, error) {
	dataPath := v.buildPath(path)

	// path formated according to vault docs for v1 and v2 API
	// v1: https://www.vaultproject.io/api-docs/secret/kv/kv-v1#read-secret
	// v2: https://www.vaultproject.io/api/secret/kv/kv-v2#read-secret-version
	req := v.client.NewRequest(http.MethodGet, fmt.Sprintf("/v1/%s", dataPath))
	if version != "" {
		req.Params.Set("version", version)
	}

	resp, err := v.client.RawRequestWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
	}

	vaultSecret, err := vault.ParseSecret(resp.Body)
	if err != nil {
		return nil, err
	}

	secretData := vaultSecret.Data
	if v.store.Version == esv1beta1.VaultKVStoreV2 {
		// Vault KV2 has data embedded within sub-field
		// reference - https://www.vaultproject.io/api/secret/kv/kv-v2#read-secret-version
		dataInt, ok := vaultSecret.Data["data"]

		if !ok {
			return nil, errors.New(errDataField)
		}
		secretData, ok = dataInt.(map[string]interface{})
		if !ok {
			return nil, errors.New(errJSONUnmarshall)
		}
	}

	return secretData, nil
}

func (v *client) newConfig() (*vault.Config, error) {
	cfg := vault.DefaultConfig()
	cfg.Address = v.store.Server

	if len(v.store.CABundle) == 0 && v.store.CAProvider == nil {
		return cfg, nil
	}

	caCertPool := x509.NewCertPool()

	if len(v.store.CABundle) > 0 {
		ok := caCertPool.AppendCertsFromPEM(v.store.CABundle)
		if !ok {
			return nil, errors.New(errVaultCert)
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
			return nil, errors.New(errVaultCert)
		}
	}

	if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
		transport.TLSClientConfig.RootCAs = caCertPool
	}

	// If either read-after-write consistency feature is enabled, enable ReadYourWrites
	cfg.ReadYourWrites = v.store.ReadYourWrites || v.store.ForwardInconsistent

	return cfg, nil
}

func getCertFromSecret(v *client) ([]byte, error) {
	secretRef := esmeta.SecretKeySelector{
		Name: v.store.CAProvider.Name,
		Key:  v.store.CAProvider.Key,
	}

	if v.store.CAProvider.Namespace != nil {
		secretRef.Namespace = v.store.CAProvider.Namespace
	}

	ctx := context.Background()
	res, err := v.secretKeyRef(ctx, &secretRef)
	if err != nil {
		return nil, fmt.Errorf(errVaultCert, err)
	}

	return []byte(res), nil
}

func getCertFromConfigMap(v *client) ([]byte, error) {
	objKey := types.NamespacedName{
		Name: v.store.CAProvider.Name,
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

func (v *client) setAuth(ctx context.Context, client Client, cfg *vault.Config) error {
	tokenExists, err := setSecretKeyToken(ctx, v, client)
	if tokenExists {
		return err
	}

	tokenExists, err = setAppRoleToken(ctx, v, client)
	if tokenExists {
		return err
	}

	tokenExists, err = setKubernetesAuthToken(ctx, v, client)
	if tokenExists {
		return err
	}

	tokenExists, err = setLdapAuthToken(ctx, v, client)
	if tokenExists {
		return err
	}

	tokenExists, err = setJwtAuthToken(ctx, v, client)
	if tokenExists {
		return err
	}

	tokenExists, err = setCertAuthToken(ctx, v, client, cfg)
	if tokenExists {
		return err
	}

	return errors.New(errAuthFormat)
}

func setAppRoleToken(ctx context.Context, v *client, client Client) (bool, error) {
	tokenRef := v.store.Auth.TokenSecretRef
	if tokenRef != nil {
		token, err := v.secretKeyRef(ctx, tokenRef)
		if err != nil {
			return true, err
		}
		client.SetToken(token)
		return true, nil
	}
	return false, nil
}

func setSecretKeyToken(ctx context.Context, v *client, client Client) (bool, error) {
	appRole := v.store.Auth.AppRole
	if appRole != nil {
		token, err := v.requestTokenWithAppRoleRef(ctx, client, appRole)
		if err != nil {
			return true, err
		}
		client.SetToken(token)
		return true, nil
	}
	return false, nil
}

func setKubernetesAuthToken(ctx context.Context, v *client, client Client) (bool, error) {
	kubernetesAuth := v.store.Auth.Kubernetes
	if kubernetesAuth != nil {
		token, err := v.requestTokenWithKubernetesAuth(ctx, client, kubernetesAuth)
		if err != nil {
			return true, err
		}
		client.SetToken(token)
		return true, nil
	}
	return false, nil
}

func setLdapAuthToken(ctx context.Context, v *client, client Client) (bool, error) {
	ldapAuth := v.store.Auth.Ldap
	if ldapAuth != nil {
		token, err := v.requestTokenWithLdapAuth(ctx, client, ldapAuth)
		if err != nil {
			return true, err
		}
		client.SetToken(token)
		return true, nil
	}
	return false, nil
}

func setJwtAuthToken(ctx context.Context, v *client, client Client) (bool, error) {
	jwtAuth := v.store.Auth.Jwt
	if jwtAuth != nil {
		token, err := v.requestTokenWithJwtAuth(ctx, client, jwtAuth)
		if err != nil {
			return true, err
		}
		client.SetToken(token)
		return true, nil
	}
	return false, nil
}

func setCertAuthToken(ctx context.Context, v *client, client Client, cfg *vault.Config) (bool, error) {
	certAuth := v.store.Auth.Cert
	if certAuth != nil {
		token, err := v.requestTokenWithCertAuth(ctx, client, certAuth, cfg)
		if err != nil {
			return true, err
		}
		client.SetToken(token)
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
		retval, err := v.secretKeyRef(ctx, &esmeta.SecretKeySelector{
			Name:      tokenRef.Name,
			Namespace: &ref.Namespace,
			Key:       "token",
		})

		if err != nil {
			continue
		}

		return retval, nil
	}
	return "", fmt.Errorf(errGetKubeSANoToken, ref.Name)
}

func (v *client) secretKeyRef(ctx context.Context, secretRef *esmeta.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	ref := types.NamespacedName{
		Namespace: v.namespace,
		Name:      secretRef.Name,
	}
	if (v.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(secretRef.Namespace != nil) {
		ref.Namespace = *secretRef.Namespace
	}
	err := v.kube.Get(ctx, ref, secret)
	if err != nil {
		return "", fmt.Errorf(errGetKubeSecret, ref.Name, err)
	}

	keyBytes, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", fmt.Errorf(errSecretKeyFmt, secretRef.Key)
	}

	value := string(keyBytes)
	valueStr := strings.TrimSpace(value)
	return valueStr, nil
}

// checkToken does a lookup and checks if the provided token exists.
func checkToken(ctx context.Context, vStore *client) error {
	// https://www.vaultproject.io/api-docs/auth/token#lookup-a-token-self
	req := vStore.client.NewRequest("GET", "/v1/auth/token/lookup-self")
	_, err := vStore.client.RawRequestWithContext(ctx, req)
	return err
}

// appRoleParameters creates the required body for Vault AppRole Auth.
// Reference - https://www.vaultproject.io/api-docs/auth/approle#login-with-approle
func appRoleParameters(role, secret string) map[string]string {
	return map[string]string{
		"role_id":   role,
		"secret_id": secret,
	}
}

func (v *client) requestTokenWithAppRoleRef(ctx context.Context, client Client, appRole *esv1beta1.VaultAppRole) (string, error) {
	roleID := strings.TrimSpace(appRole.RoleID)

	secretID, err := v.secretKeyRef(ctx, &appRole.SecretRef)
	if err != nil {
		return "", err
	}

	parameters := appRoleParameters(roleID, secretID)
	url := strings.Join([]string{"/v1", "auth", appRole.Path, "login"}, "/")
	request := client.NewRequest("POST", url)

	err = request.SetJSONBody(parameters)
	if err != nil {
		return "", fmt.Errorf(errVaultReqParams, err)
	}

	resp, err := client.RawRequestWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf(errVaultRequest, err)
	}

	defer resp.Body.Close()

	vaultResult := vault.Secret{}
	if err = resp.DecodeJSON(&vaultResult); err != nil {
		return "", fmt.Errorf(errVaultResponse, err)
	}

	token, err := vaultResult.TokenID()
	if err != nil {
		return "", fmt.Errorf(errVaultToken, err)
	}

	return token, nil
}

// kubeParameters creates the required body for Vault Kubernetes auth.
// Reference - https://www.vaultproject.io/api/auth/kubernetes#login
func kubeParameters(role, jwt string) map[string]string {
	return map[string]string{
		"role": role,
		"jwt":  jwt,
	}
}

func (v *client) requestTokenWithKubernetesAuth(ctx context.Context, client Client, kubernetesAuth *esv1beta1.VaultKubernetesAuth) (string, error) {
	jwtString, err := getJwtString(ctx, v, kubernetesAuth)
	if err != nil {
		return "", err
	}

	parameters := kubeParameters(kubernetesAuth.Role, jwtString)
	url := strings.Join([]string{"/v1", "auth", kubernetesAuth.Path, "login"}, "/")
	request := client.NewRequest("POST", url)

	err = request.SetJSONBody(parameters)
	if err != nil {
		return "", fmt.Errorf(errVaultReqParams, err)
	}

	resp, err := client.RawRequestWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf(errVaultRequest, err)
	}

	defer resp.Body.Close()
	vaultResult := vault.Secret{}
	err = resp.DecodeJSON(&vaultResult)
	if err != nil {
		return "", fmt.Errorf(errVaultResponse, err)
	}

	token, err := vaultResult.TokenID()
	if err != nil {
		return "", fmt.Errorf(errVaultToken, err)
	}

	return token, nil
}

func getJwtString(ctx context.Context, v *client, kubernetesAuth *esv1beta1.VaultKubernetesAuth) (string, error) {
	if kubernetesAuth.ServiceAccountRef != nil {
		jwt, err := v.secretKeyRefForServiceAccount(ctx, kubernetesAuth.ServiceAccountRef)
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
		jwt, err := v.secretKeyRef(ctx, tokenRef)
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

func (v *client) requestTokenWithLdapAuth(ctx context.Context, client Client, ldapAuth *esv1beta1.VaultLdapAuth) (string, error) {
	username := strings.TrimSpace(ldapAuth.Username)

	password, err := v.secretKeyRef(ctx, &ldapAuth.SecretRef)
	if err != nil {
		return "", err
	}

	parameters := map[string]string{
		"password": password,
	}
	url := strings.Join([]string{"/v1", "auth", ldapAuth.Path, "login", username}, "/")
	request := client.NewRequest("POST", url)

	err = request.SetJSONBody(parameters)
	if err != nil {
		return "", fmt.Errorf(errVaultReqParams, err)
	}

	resp, err := client.RawRequestWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf(errVaultRequest, err)
	}

	defer resp.Body.Close()

	vaultResult := vault.Secret{}
	if err = resp.DecodeJSON(&vaultResult); err != nil {
		return "", fmt.Errorf(errVaultResponse, err)
	}

	token, err := vaultResult.TokenID()
	if err != nil {
		return "", fmt.Errorf(errVaultToken, err)
	}

	return token, nil
}

func (v *client) requestTokenWithJwtAuth(ctx context.Context, client Client, jwtAuth *esv1beta1.VaultJwtAuth) (string, error) {
	role := strings.TrimSpace(jwtAuth.Role)

	jwt, err := v.secretKeyRef(ctx, &jwtAuth.SecretRef)
	if err != nil {
		return "", err
	}

	parameters := map[string]string{
		"role": role,
		"jwt":  jwt,
	}
	url := strings.Join([]string{"/v1", "auth", jwtAuth.Path, "login"}, "/")
	request := client.NewRequest("POST", url)

	err = request.SetJSONBody(parameters)
	if err != nil {
		return "", fmt.Errorf(errVaultReqParams, err)
	}

	resp, err := client.RawRequestWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf(errVaultRequest, err)
	}

	defer resp.Body.Close()

	vaultResult := vault.Secret{}
	if err = resp.DecodeJSON(&vaultResult); err != nil {
		return "", fmt.Errorf(errVaultResponse, err)
	}

	token, err := vaultResult.TokenID()
	if err != nil {
		return "", fmt.Errorf(errVaultToken, err)
	}

	return token, nil
}

func (v *client) requestTokenWithCertAuth(ctx context.Context, client Client, certAuth *esv1beta1.VaultCertAuth, cfg *vault.Config) (string, error) {
	clientKey, err := v.secretKeyRef(ctx, &certAuth.SecretRef)
	if err != nil {
		return "", err
	}

	clientCert, err := v.secretKeyRef(ctx, &certAuth.ClientCert)
	if err != nil {
		return "", err
	}

	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return "", fmt.Errorf(errClientTLSAuth, err)
	}

	if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	url := strings.Join([]string{"/v1", "auth", "cert", "login"}, "/")
	request := client.NewRequest("POST", url)

	resp, err := client.RawRequestWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf(errVaultRequest, err)
	}

	defer resp.Body.Close()

	vaultResult := vault.Secret{}
	if err = resp.DecodeJSON(&vaultResult); err != nil {
		return "", fmt.Errorf(errVaultResponse, err)
	}

	token, err := vaultResult.TokenID()
	if err != nil {
		return "", fmt.Errorf(errVaultToken, err)
	}

	return token, nil
}
