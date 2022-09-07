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

package doppler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/provider/doppler/client"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	customBaseURLEnvVar                                = "DOPPLER_BASE_URL"
	verifyTLSOverrideEnvVar                            = "DOPPLER_VERIFY_TLS"
	secretsDownloadFileKey                             = "DOPPLER_SECRETS_FILE"
	errDopplerStore                                    = "missing or invalid Doppler SecretStore"
	errDopplerTokenSecretName                          = "missing auth.secretRef.dopplerToken.name"
	errInvalidClusterStoreMissingDopplerTokenNamespace = "missing auth.secretRef.dopplerToken.namespace"
	errFetchDopplerTokenSecret                         = "unable to find find DopplerToken secret: %w"
	errMissingDopplerToken                             = "auth.secretRef.dopplerToken.key '%s' not found in secret '%s'"
	errNewClient                                       = "unable to create DopplerClient : %s"
	errInvalidStore                                    = "invalid store: %s"
	errGetSecret                                       = "could not get secret %s: %s"
	errGetSecrets                                      = "could not get secrets %s"
	errUnmarshalSecretMap                              = "unable to unmarshal secret %s: %w"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Doppler{}
var _ esv1beta1.Provider = &Doppler{}

type Doppler struct {
	client          SecretsClientInterface
	project         string
	config          string
	nameTransformer string
	format          string
}

// SecretsClientInterface is the required subset of the Doppler API Client.
type SecretsClientInterface interface {
	BaseURL() *url.URL
	Authenticate() error
	GetSecret(request client.SecretRequest) (*client.SecretResponse, error)
	GetSecrets(request client.SecretsRequest) (*client.SecretsResponse, error)
}

type kubeClient struct {
	kube         kclient.Client
	store        *esv1beta1.DopplerProvider
	namespace    string
	storeKind    string
	dopplerToken string
}

func (kc *kubeClient) setAuth(ctx context.Context) error {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := kc.store.Auth.SecretRef.DopplerToken.Name
	if credentialsSecretName == "" {
		return fmt.Errorf(errDopplerTokenSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: kc.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if kc.storeKind == esv1beta1.ClusterSecretStoreKind {
		if kc.store.Auth.SecretRef.DopplerToken.Namespace == nil {
			return fmt.Errorf(errInvalidClusterStoreMissingDopplerTokenNamespace)
		}
		objectKey.Namespace = *kc.store.Auth.SecretRef.DopplerToken.Namespace
	}

	err := kc.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return fmt.Errorf(errFetchDopplerTokenSecret, err)
	}

	dopplerToken := credentialsSecret.Data[kc.store.Auth.SecretRef.DopplerToken.Key]
	if (dopplerToken == nil) || (len(dopplerToken) == 0) {
		return fmt.Errorf(errMissingDopplerToken, kc.store.Auth.SecretRef.DopplerToken.Key, credentialsSecretName)
	}

	kc.dopplerToken = string(dopplerToken)

	return nil
}

func (d *Doppler) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Doppler == nil {
		return nil, fmt.Errorf(errDopplerStore)
	}

	dopplerStoreSpec := storeSpec.Provider.Doppler

	// Default Key to dopplerToken if not specified
	if dopplerStoreSpec.Auth.SecretRef.DopplerToken.Key == "" {
		storeSpec.Provider.Doppler.Auth.SecretRef.DopplerToken.Key = "dopplerToken"
	}

	kubeClient := &kubeClient{
		kube:      kube,
		store:     dopplerStoreSpec,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	if err := kubeClient.setAuth(ctx); err != nil {
		return nil, err
	}

	client, err := client.NewDopplerClient(kubeClient.dopplerToken)
	if err != nil {
		return nil, fmt.Errorf(errNewClient, err)
	}

	if customBaseURL, found := os.LookupEnv(customBaseURLEnvVar); found {
		if err := client.SetBaseURL(customBaseURL); err != nil {
			return nil, fmt.Errorf(errNewClient, err)
		}
	}

	if customVerifyTLS, found := os.LookupEnv(verifyTLSOverrideEnvVar); found {
		customVerifyTLS, err := strconv.ParseBool(customVerifyTLS)
		if err == nil {
			client.VerifyTLS = customVerifyTLS
		}
	}

	d.client = client
	d.project = kubeClient.store.Project
	d.config = kubeClient.store.Config
	d.nameTransformer = kubeClient.store.NameTransformer
	d.format = kubeClient.store.Format

	return d, nil
}

func (d *Doppler) Validate() (esv1beta1.ValidationResult, error) {
	timeout := 15 * time.Second
	clientURL := d.client.BaseURL().String()

	if err := utils.NetworkValidate(clientURL, timeout); err != nil {
		return esv1beta1.ValidationResultError, err
	}

	if err := d.client.Authenticate(); err != nil {
		return esv1beta1.ValidationResultError, err
	}

	return esv1beta1.ValidationResultReady, nil
}

func (d *Doppler) ValidateStore(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	dopplerStoreSpec := storeSpec.Provider.Doppler
	dopplerTokenSecretRef := dopplerStoreSpec.Auth.SecretRef.DopplerToken
	if err := utils.ValidateSecretSelector(store, dopplerTokenSecretRef); err != nil {
		return fmt.Errorf(errInvalidStore, err)
	}

	if dopplerTokenSecretRef.Name == "" {
		return fmt.Errorf(errInvalidStore, "dopplerToken.name cannot be empty")
	}

	return nil
}

func (d *Doppler) GetSecrets(_ context.Context) (map[string][]byte, error) {
	request := client.SecretsRequest{
		Project:         d.project,
		Config:          d.config,
		NameTransformer: d.nameTransformer,
		Format:          d.format,
	}

	response, err := d.client.GetSecrets(request)
	if err != nil {
		return nil, fmt.Errorf(errGetSecrets, err)
	}

	if d.format != "" {
		return map[string][]byte{
			secretsDownloadFileKey: response.Body,
		}, nil
	}

	return ExternalSecretsFormat(response.Secrets), nil
}

func (d *Doppler) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	request := client.SecretRequest{
		Name:    ref.Key,
		Project: d.project,
		Config:  d.config,
	}

	secret, err := d.client.GetSecret(request)
	if err != nil {
		return nil, fmt.Errorf(errGetSecret, ref.Key, err)
	}

	return []byte(secret.Value), nil
}

func (d *Doppler) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := d.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalSecretMap, ref.Key, err)
	}

	secretData := make(map[string][]byte)
	for k, v := range kv {
		var strVal string
		err = json.Unmarshal(v, &strVal)
		if err == nil {
			secretData[k] = []byte(strVal)
		} else {
			secretData[k] = v
		}
	}
	return secretData, nil
}

func (d *Doppler) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	secrets, err := d.GetSecrets(ctx)
	selected := map[string][]byte{}

	if err != nil {
		return nil, err
	}

	if ref.Name == nil && ref.Path == nil {
		return secrets, nil
	}

	var matcher *find.Matcher
	if ref.Name != nil {
		m, err := find.New(*ref.Name)
		if err != nil {
			return nil, err
		}
		matcher = m
	}

	for key, value := range secrets {
		if (matcher != nil && !matcher.MatchName(key)) || (ref.Path != nil && !strings.HasPrefix(key, *ref.Path)) {
			continue
		}
		selected[key] = value
	}

	return selected, nil
}

func (d *Doppler) Close(_ context.Context) error {
	return nil
}

func ExternalSecretsFormat(secrets client.Secrets) map[string][]byte {
	converted := make(map[string][]byte, len(secrets))
	for key, value := range secrets {
		converted[key] = []byte(value)
	}
	return converted
}

func init() {
	esv1beta1.Register(&Doppler{}, &esv1beta1.SecretStoreProvider{
		Doppler: &esv1beta1.DopplerProvider{},
	})
}
