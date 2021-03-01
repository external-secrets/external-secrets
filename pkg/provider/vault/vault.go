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
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

var (
	_ provider.Provider      = &connector{}
	_ provider.SecretsClient = &client{}
)

const (
	serviceAccTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	errVaultStore     = "received invalid Vault SecretStore resource: %w"
	errVaultClient    = "cannot setup new vault client: %w"
	errVaultCert      = "cannot set Vault CA certificate: %w"
	errReadSecret     = "cannot read secret data from Vault: %w"
	errAuthFormat     = "cannot initialize Vault client: no valid auth method specified: %w"
	errVaultData      = "cannot parse Vault response data: %w"
	errVaultToken     = "cannot parse Vault authentication token: %w"
	errVaultReqParams = "cannot set Vault request parameters: %w"
	errVaultRequest   = "error from Vault request: %w"
	errVaultResponse  = "cannot parse Vault response: %w"
	errServiceAccount = "cannot read Kubernetes service account token from file system: %w"

	errGetKubeSecret = "cannot get Kubernetes secret %q: %w"
	errSecretKeyFmt  = "cannot find secret data for key: %q"
)

type Client interface {
	NewRequest(method, requestPath string) *vault.Request
	RawRequestWithContext(ctx context.Context, r *vault.Request) (*vault.Response, error)
	SetToken(v string)
	SetNamespace(namespace string)
}

type client struct {
	kube      kclient.Client
	store     *esv1alpha1.VaultProvider
	log       logr.Logger
	client    Client
	namespace string
	storeKind string
}

func init() {
	schema.Register(&connector{
		newVaultClient: newVaultClient,
	}, &esv1alpha1.SecretStoreProvider{
		Vault: &esv1alpha1.VaultProvider{},
	})
}

func newVaultClient(c *vault.Config) (Client, error) {
	return vault.NewClient(c)
}

type connector struct {
	newVaultClient func(c *vault.Config) (Client, error)
}

func (c *connector) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube kclient.Client, namespace string) (provider.SecretsClient, error) {
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

	if err := vStore.setAuth(ctx, client); err != nil {
		return nil, err
	}

	vStore.client = client
	return vStore, nil
}

func (v *client) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	data, err := v.readSecret(ctx, ref.Key, ref.Version)
	if err != nil {
		return nil, err
	}
	value, exists := data[ref.Property]
	if !exists {
		return nil, fmt.Errorf(errSecretKeyFmt, ref.Property)
	}
	return value, nil
}

func (v *client) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return v.readSecret(ctx, ref.Key, ref.Version)
}

func (v *client) readSecret(ctx context.Context, path, version string) (map[string][]byte, error) {
	kvPath := v.store.Path

	if v.store.Version == esv1alpha1.VaultKVStoreV2 {
		if !strings.HasSuffix(kvPath, "/data") {
			kvPath = fmt.Sprintf("%s/data", kvPath)
		}
	}

	// path formated according to vault docs for v1 and v2 API
	// v1: https://www.vaultproject.io/api-docs/secret/kv/kv-v1#read-secret
	// v2: https://www.vaultproject.io/api/secret/kv/kv-v2#read-secret-version
	req := v.client.NewRequest(http.MethodGet, fmt.Sprintf("/v1/%s/%s", kvPath, path))
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
	if v.store.Version == esv1alpha1.VaultKVStoreV2 {
		// Vault KV2 has data embedded within sub-field
		// reference - https://www.vaultproject.io/api/secret/kv/kv-v2#read-secret-version
		dataInt, ok := vaultSecret.Data["data"]
		if !ok {
			return nil, errors.New(errVaultData)
		}
		secretData, ok = dataInt.(map[string]interface{})
		if !ok {
			return nil, errors.New(errVaultData)
		}
	}

	byteMap := make(map[string][]byte, len(secretData))
	for k, v := range secretData {
		switch t := v.(type) {
		case string:
			byteMap[k] = []byte(t)
		case []byte:
			byteMap[k] = t
		default:
			return nil, errors.New(errVaultData)
		}
	}

	return byteMap, nil
}

func (v *client) newConfig() (*vault.Config, error) {
	cfg := vault.DefaultConfig()
	cfg.Address = v.store.Server

	if len(v.store.CABundle) == 0 {
		return cfg, nil
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(v.store.CABundle)
	if !ok {
		return nil, errors.New(errVaultCert)
	}

	if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
		transport.TLSClientConfig.RootCAs = caCertPool
	}

	return cfg, nil
}

func (v *client) setAuth(ctx context.Context, client Client) error {
	tokenRef := v.store.Auth.TokenSecretRef
	if tokenRef != nil {
		token, err := v.secretKeyRef(ctx, tokenRef)
		if err != nil {
			return err
		}
		client.SetToken(token)
		return nil
	}

	appRole := v.store.Auth.AppRole
	if appRole != nil {
		token, err := v.requestTokenWithAppRoleRef(ctx, client, appRole)
		if err != nil {
			return err
		}
		client.SetToken(token)
		return nil
	}

	kubernetesAuth := v.store.Auth.Kubernetes
	if kubernetesAuth != nil {
		token, err := v.requestTokenWithKubernetesAuth(ctx, client, kubernetesAuth)
		if err != nil {
			return err
		}
		client.SetToken(token)
		return nil
	}

	return errors.New(errAuthFormat)
}

func (v *client) secretKeyRef(ctx context.Context, secretRef *esmeta.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	ref := types.NamespacedName{
		Namespace: v.namespace,
		Name:      secretRef.Name,
	}
	if (v.storeKind == esv1alpha1.ClusterSecretStoreKind) &&
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

// appRoleParameters creates the required body for Vault AppRole Auth.
// Reference - https://www.vaultproject.io/api-docs/auth/approle#login-with-approle
func appRoleParameters(role, secret string) map[string]string {
	return map[string]string{
		"role_id":   role,
		"secret_id": secret,
	}
}

func (v *client) requestTokenWithAppRoleRef(ctx context.Context, client Client, appRole *esv1alpha1.VaultAppRole) (string, error) {
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

func (v *client) requestTokenWithKubernetesAuth(ctx context.Context, client Client, kubernetesAuth *esv1alpha1.VaultKubernetesAuth) (string, error) {
	jwtString := ""
	if kubernetesAuth.SecretRef != nil {
		tokenRef := kubernetesAuth.SecretRef
		if tokenRef.Key == "" {
			tokenRef = kubernetesAuth.SecretRef.DeepCopy()
			tokenRef.Key = "token"
		}
		jwt, err := v.secretKeyRef(ctx, tokenRef)
		if err != nil {
			return "", err
		}
		jwtString = jwt
	} else {
		// Kubernetes authentication is specified, but without a referenced
		// Kubernetes secret. We check if the file path for in-cluster service account
		// exists and attempt to use the token for Vault Kubernetes auth.
		if _, err := os.Stat(serviceAccTokenPath); err != nil {
			return "", fmt.Errorf(errServiceAccount, err)
		}
		jwtByte, err := ioutil.ReadFile(serviceAccTokenPath)
		if err != nil {
			return "", fmt.Errorf(errServiceAccount, err)
		}
		jwtString = string(jwtByte)
	}

	parameters := kubeParameters(kubernetesAuth.Role, jwtString)
	url := strings.Join([]string{"/v1", "auth", kubernetesAuth.Path, "login"}, "/")
	request := client.NewRequest("POST", url)

	err := request.SetJSONBody(parameters)
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
