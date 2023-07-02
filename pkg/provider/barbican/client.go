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
package barbican

import (
	"context"
	"encoding/json"
	"fmt"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/artashesbalabekyan/barbican-sdk-go/client"
	"github.com/artashesbalabekyan/barbican-sdk-go/xhttp"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
)

const (
	errBarbicanStore        = "received invalid Barbican resource"
	errFetchSAKSecret       = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK           = "missing SecretAccessKey"
	errJSONSecretUnmarshal  = "unable to unmarshal secret: %w"
	errInvalidStore         = "invalid store"
	errInvalidStoreSpec     = "invalid store spec"
	errInvalidStoreProv     = "invalid store provider"
	errInvalidBarbicanProv  = "invalid barbican secrets manager provider"
	errInvalidAuthSecretRef = "invalid auth secret ref: %w"
)

type Client struct {
	config    *xhttp.Config
	client    client.Conn
	kube      kclient.Client
	store     *esv1beta1.BarbicanProvider
	storeKind string

	// namespace of the external secret
	namespace string
}

func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {
	err := c.client.DeleteSecret(ctx, remoteRef.GetRemoteKey())
	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanDeleteSecret, err)
	return err
}

// GetAllSecrets syncs multiple secrets from barbican provider into a single Kubernetes Secret.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	iterator, err := c.client.ListSecrets(ctx)
	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanGetAllSecrets, err)
	if err != nil {
		return nil, err
	}

	names := []string{}

	defer iterator.Close()

	for {
		name, ok := iterator.Next()
		if !ok {
			break
		}
		names = append(names, name)
	}

	result := make(map[string][]byte, len(names))

	for _, name := range names {
		secret, err := c.client.GetSecretWithPayload(ctx, name)
		metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanGetAllSecrets, err)
		if err != nil {
			return nil, err
		}
		result[name] = secret.Payload
	}

	return result, nil
}

// GetSecret returns a single secret from the provider.
func (c *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.client.GetSecretWithPayload(ctx, ref.Key)
	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanGetSecret, err)
	if err != nil {
		return nil, err
	}
	val := secret.Payload
	return val, nil
}

func (c *Client) Close(_ context.Context) error {
	return nil
}

func (c *Client) Validate() (esv1beta1.ValidationResult, error) {
	if c.storeKind == esv1beta1.ClusterSecretStoreKind && isReferentSpec(c.store) {
		return esv1beta1.ValidationResultUnknown, nil
	}
	return esv1beta1.ValidationResultReady, nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := c.GetSecret(ctx, ref)
	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanGetSecret, err)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
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

// PushSecret pushes a kubernetes secret key into barbican provider Secret.
func (c *Client) PushSecret(ctx context.Context, payload []byte, remoteRef esv1beta1.PushRemoteRef) error {
	err := c.client.Create(ctx, remoteRef.GetRemoteKey(), payload)
	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanCreateSecret, err)
	return err
}
