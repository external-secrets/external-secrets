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
	approle "github.com/hashicorp/vault/api/auth/approle"
	authkubernetes "github.com/hashicorp/vault/api/auth/kubernetes"
	authldap "github.com/hashicorp/vault/api/auth/ldap"
	"github.com/tidwall/gjson"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

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
	errVaultRequest         = "error from Vault request: %w"
	errServiceAccount       = "cannot read Kubernetes service account token from file system: %w"
	errJwtNoTokenSource     = "neither `secretRef` nor `kubernetesServiceAccountToken` was supplied as token source for jwt authentication"
	errUnsupportedKvVersion = "cannot perform find operations with kv version v1"

	errGetKubeSA             = "cannot get Kubernetes service account %q: %w"
	errGetKubeSASecrets      = "cannot find secrets bound to service account: %q"
	errGetKubeSANoToken      = "cannot find token in secrets bound to service account: %q"
	errGetKubeSATokenRequest = "cannot request Kubernetes service account token for service account %q: %w"

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
	errInvalidJwtK8sSA   = "invalid Auth.Jwt.KubernetesServiceAccountToken.ServiceAccountRef: %w"
	errInvalidKubeSA     = "invalid Auth.Kubernetes.ServiceAccountRef: %w"
	errInvalidKubeSec    = "invalid Auth.Kubernetes.SecretRef: %w"
	errInvalidLdapSec    = "invalid Auth.Ldap.SecretRef: %w"
	errInvalidTokenRef   = "invalid Auth.TokenSecretRef: %w"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &client{}
var _ esv1beta1.Provider = &connector{}

type Auth interface {
	Login(ctx context.Context, authMethod vault.AuthMethod) (*vault.Secret, error)
}

type Token interface {
	RevokeSelfWithContext(ctx context.Context, token string) error
	LookupSelfWithContext(ctx context.Context) (*vault.Secret, error)
}

type Logical interface {
	ReadWithDataWithContext(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error)
	ListWithContext(ctx context.Context, path string) (*vault.Secret, error)
	WriteWithContext(ctx context.Context, path string, data map[string]interface{}) (*vault.Secret, error)
}

type Client interface {
	SetToken(v string)
	Token() string
	ClearToken()
	Auth() Auth
	Logical() Logical
	AuthToken() Token
	SetNamespace(namespace string)
	AddHeader(key, value string)
}

type VClient struct {
	setToken     func(v string)
	token        func() string
	clearToken   func()
	auth         Auth
	logical      Logical
	authToken    Token
	setNamespace func(namespace string)
	addHeader    func(key, value string)
}

func (v VClient) AddHeader(key, value string) {
	v.addHeader(key, value)
}

func (v VClient) SetNamespace(namespace string) {
	v.setNamespace(namespace)
}

func (v VClient) ClearToken() {
	v.clearToken()
}

func (v VClient) Token() string {
	return v.token()
}

func (v VClient) SetToken(token string) {
	v.setToken(token)
}

func (v VClient) Auth() Auth {
	return v.auth
}

func (v VClient) AuthToken() Token {
	return v.authToken
}

func (v VClient) Logical() Logical {
	return v.logical
}

type client struct {
	kube      kclient.Client
	store     *esv1beta1.VaultProvider
	log       logr.Logger
	corev1    typedcorev1.CoreV1Interface
	client    Client
	auth      Auth
	logical   Logical
	token     Token
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
	cl, err := vault.NewClient(c)
	if err != nil {
		return nil, err
	}
	auth := cl.Auth()
	logical := cl.Logical()
	token := cl.Auth().Token()
	out := VClient{
		setToken:     cl.SetToken,
		token:        cl.Token,
		clearToken:   cl.ClearToken,
		auth:         auth,
		authToken:    token,
		logical:      logical,
		setNamespace: cl.SetNamespace,
		addHeader:    cl.AddHeader,
	}
	return out, nil
}

type connector struct {
	newVaultClient func(c *vault.Config) (Client, error)
}

func (c *connector) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
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

func (c *connector) newClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, corev1 typedcorev1.CoreV1Interface, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Vault == nil {
		return nil, errors.New(errVaultStore)
	}
	vaultSpec := storeSpec.Provider.Vault

	vStore := &client{
		kube:      kube,
		corev1:    corev1,
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
	vStore.client = client
	vStore.auth = client.Auth()
	vStore.logical = client.Logical()
	vStore.token = client.AuthToken()

	if err := vStore.setAuth(ctx, cfg); err != nil {
		return nil, err
	}

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
		if p.Auth.Jwt.SecretRef != nil {
			if err := utils.ValidateSecretSelector(store, *p.Auth.Jwt.SecretRef); err != nil {
				return fmt.Errorf(errInvalidJwtSec, err)
			}
		} else if p.Auth.Jwt.KubernetesServiceAccountToken != nil {
			if err := utils.ValidateServiceAccountSelector(store, p.Auth.Jwt.KubernetesServiceAccountToken.ServiceAccountRef); err != nil {
				return fmt.Errorf(errInvalidJwtK8sSA, err)
			}
		} else {
			return fmt.Errorf(errJwtNoTokenSource)
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
			if err != nil {
				return nil, err
			}
			secrets[name] = secret
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
			if err != nil {
				return nil, err
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
	secret, err := v.logical.ListWithContext(ctx, url)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
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
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
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
		revoke, err := checkToken(ctx, v)
		if err != nil {
			return fmt.Errorf(errVaultRevokeToken, err)
		}
		if revoke {
			err = v.token.RevokeSelfWithContext(ctx, v.client.Token())
			if err != nil {
				return fmt.Errorf(errVaultRevokeToken, err)
			}
			v.client.ClearToken()
		}
	}
	return nil
}

func (v *client) Validate() (esv1beta1.ValidationResult, error) {
	_, err := checkToken(context.Background(), v)
	if err != nil {
		return esv1beta1.ValidationResultError, fmt.Errorf(errInvalidCredentials, err)
	}
	return esv1beta1.ValidationResultReady, nil
}

func (v *client) buildMetadataPath(path string) (string, error) {
	var url string
	if v.store.Path == nil && !strings.Contains(path, "data") {
		return "", fmt.Errorf(errPathInvalid)
	}
	if v.store.Path == nil {
		path = strings.Replace(path, "data", "metadata", 1)
		url = path
	} else {
		url = fmt.Sprintf("%s/metadata/%s", *v.store.Path, path)
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
	var params map[string][]string
	if version != "" {
		params = make(map[string][]string)
		params["version"] = []string{version}
	}
	vaultSecret, err := v.logical.ReadWithDataWithContext(ctx, dataPath, params)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
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
	// In a controller-runtime context, we rely on the reconciliation process for retrying
	cfg.MaxRetries = 0

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

func (v *client) setAuth(ctx context.Context, cfg *vault.Config) error {
	tokenExists, err := setSecretKeyToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setAppRoleToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setKubernetesAuthToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setLdapAuthToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setJwtAuthToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setCertAuthToken(ctx, v, cfg)
	if tokenExists {
		return err
	}

	return errors.New(errAuthFormat)
}

func setAppRoleToken(ctx context.Context, v *client) (bool, error) {
	tokenRef := v.store.Auth.TokenSecretRef
	if tokenRef != nil {
		token, err := v.secretKeyRef(ctx, tokenRef)
		if err != nil {
			return true, err
		}
		v.client.SetToken(token)
		return true, nil
	}
	return false, nil
}

func setSecretKeyToken(ctx context.Context, v *client) (bool, error) {
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

func (v *client) serviceAccountToken(ctx context.Context, serviceAccountRef esmeta.ServiceAccountSelector, audiences []string, expirationSeconds int64) (string, error) {
	tokenRequest := &authenticationv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v.namespace,
		},
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}
	if (v.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		tokenRequest.Namespace = *serviceAccountRef.Namespace
	}
	tokenResponse, err := v.corev1.ServiceAccounts(tokenRequest.Namespace).CreateToken(ctx, serviceAccountRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf(errGetKubeSATokenRequest, serviceAccountRef.Name, err)
	}
	return tokenResponse.Status.Token, nil
}

// checkToken does a lookup and checks if the provided token exists.
func checkToken(ctx context.Context, vStore *client) (bool, error) {
	// https://www.vaultproject.io/api-docs/auth/token#lookup-a-token-self
	resp, err := vStore.token.LookupSelfWithContext(ctx)
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

func (v *client) requestTokenWithAppRoleRef(ctx context.Context, appRole *esv1beta1.VaultAppRole) error {
	roleID := strings.TrimSpace(appRole.RoleID)

	secretID, err := v.secretKeyRef(ctx, &appRole.SecretRef)
	if err != nil {
		return err
	}
	secret := approle.SecretID{FromString: secretID}
	appRoleClient, err := approle.NewAppRoleAuth(roleID, &secret, approle.WithMountPath(appRole.Path))
	if err != nil {
		return err
	}
	_, err = v.auth.Login(ctx, appRoleClient)
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
	if err != nil {
		return err
	}
	return nil
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

func (v *client) requestTokenWithLdapAuth(ctx context.Context, ldapAuth *esv1beta1.VaultLdapAuth) error {
	username := strings.TrimSpace(ldapAuth.Username)

	password, err := v.secretKeyRef(ctx, &ldapAuth.SecretRef)
	if err != nil {
		return err
	}
	pass := authldap.Password{FromString: password}
	l, err := authldap.NewLDAPAuth(username, &pass, authldap.WithMountPath(ldapAuth.Path))
	if err != nil {
		return err
	}
	_, err = v.auth.Login(ctx, l)
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
		jwt, err = v.secretKeyRef(ctx, jwtAuth.SecretRef)
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
		jwt, err = v.serviceAccountToken(ctx, k8sServiceAccountToken.ServiceAccountRef, *audiences, *expirationSeconds)
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
	clientKey, err := v.secretKeyRef(ctx, &certAuth.SecretRef)
	if err != nil {
		return err
	}

	clientCert, err := v.secretKeyRef(ctx, &certAuth.ClientCert)
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
