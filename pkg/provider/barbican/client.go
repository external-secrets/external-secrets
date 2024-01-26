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
	"strings"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/keymanager/v1/secrets"
	corev1 "k8s.io/api/core/v1"
)

const (
	errBarbicanStore        = "received invalid Barbican resource"
	errFetchSAKSecret       = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK           = "missing SecretAccessKey"
	errKeyNotFound          = "key '%s' does not exist"
	errKeyAlreadyExists     = "key '%s' already exists"
	errJSONSecretUnmarshal  = "unable to unmarshal secret: %w"
	errInvalidStore         = "invalid store"
	errInvalidStoreSpec     = "invalid store spec"
	errInvalidStoreProv     = "invalid store provider"
	errInvalidBarbicanProv  = "invalid barbican secrets manager provider"
	errInvalidAuthSecretRef = "invalid auth secret ref: %w"
)

type Client struct {
	client    *gophercloud.ServiceClient
	kube      kclient.Client
	store     *esv1beta1.BarbicanProvider
	storeKind string

	// namespace of the external secret
	namespace string
}

func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	var (
		name   = remoteRef.GetRemoteKey()
		uuid   = getUUIDFromRemoteRef(name)
		secret *secrets.Secret
		err    error
	)

	if uuid == "" {
		secret, err = c.getByName(ctx, name)
		if err != nil {
			return err
		}
		uuid = extractIdFromRef(secret.SecretRef)
	} else {
		_, err = c.getByUUID(ctx, uuid)
		if err != nil {
			return err
		}
	}

	err = secrets.Delete(c.client, uuid).Err
	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanDeleteSecret, err)
	return err
}

// GetAllSecrets syncs multiple secrets from barbican provider into a single Kubernetes Secret.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	allPages, err := secrets.List(c.client, secrets.ListOpts{
		Sort: "created:desc",
	}).AllPages()
	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanGetAllSecrets, err)
	if err != nil {
		return nil, err
	}

	allSecrets, err := secrets.ExtractSecrets(allPages)
	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanGetAllSecrets, err)
	if err != nil {
		return nil, err
	}

	mapUUIDByUniqueNames := make(map[string]string, len(allSecrets))
	for _, v := range allSecrets {
		if _, ok := mapUUIDByUniqueNames[v.Name]; !ok {
			mapUUIDByUniqueNames[v.Name] = extractIdFromRef(v.SecretRef)
		}
	}

	result := make(map[string][]byte, len(mapUUIDByUniqueNames))

	for name, id := range mapUUIDByUniqueNames {
		payload, err := secrets.GetPayload(c.client, id, secrets.GetPayloadOpts{PayloadContentType: "*/*"}).Extract()
		metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanGetAllSecrets, err)
		if err != nil {
			return nil, err
		}

		result[name] = payload
	}

	return result, nil
}

// GetSecret returns a single secret from the provider.
func (c *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	var (
		name   = ref.Key
		uuid   = getUUIDFromRemoteRef(name)
		secret *secrets.Secret
		err    error
	)

	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanGetSecret, err)
	if err != nil {
		return nil, err
	}

	if uuid == "" {
		secret, err = c.getByName(ctx, name)
		if err != nil {
			return nil, err
		}
		uuid = extractIdFromRef(secret.SecretRef)
	}

	return secrets.GetPayload(c.client, uuid, secrets.GetPayloadOpts{PayloadContentType: "*/*"}).Extract()
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
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, pushSecretData esv1beta1.PushSecretData) error {
	payload := secret.Data[pushSecretData.GetSecretKey()]
	name := pushSecretData.GetRemoteKey()

	_, err := c.getByName(ctx, name)
	if err == nil {
		return nil
	}

	createOpts := secrets.CreateOpts{
		Algorithm:          "aes",
		BitLength:          256,
		Mode:               "cbc",
		Name:               name,
		Payload:            string(payload),
		PayloadContentType: "text/plain",
		SecretType:         secrets.OpaqueSecret,
	}

	_, err = secrets.Create(c.client, createOpts).Extract()
	metrics.ObserveAPICall(constants.ProviderBarbican, constants.CallBarbicanCreateSecret, err)
	return err
}

func (s *Client) getByName(ctx context.Context, name string) (*secrets.Secret, error) {
	allPages, err := secrets.List(s.client, secrets.ListOpts{
		Name: name,
		Sort: "created:desc",
	}).AllPages()
	if err != nil {
		return nil, err
	}

	allSecrets, err := secrets.ExtractSecrets(allPages)
	if err != nil {
		return nil, err
	}

	if len(allSecrets) == 0 {
		return nil, fmt.Errorf(errKeyNotFound, name)
	}

	id := extractIdFromRef(allSecrets[0].SecretRef)

	return secrets.Get(s.client, id).Extract()
}

func (s *Client) getByUUID(ctx context.Context, uuid string) (*secrets.Secret, error) {
	return secrets.Get(s.client, uuid).Extract()
}

func extractIdFromRef(ref string) string {
	splitedRef := strings.Split(ref, "/")
	return splitedRef[len(splitedRef)-1]
}

func getUUIDFromRemoteRef(ref string) string {
	splitedRef := strings.Split(ref, "/")
	if len(splitedRef) == 2 && splitedRef[0] == "uuid" {
		return splitedRef[1]
	}
	return ""
}
