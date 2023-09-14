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
package kubernetes

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	metaLabels      = "labels"
	metaAnnotations = "annotations"
)

func (c *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.userSecretClient.Get(ctx, ref.Key, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var values map[string][]byte
	if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		values, err = getSecretMetadata(secret)
		if err != nil {
			return nil, err
		}
	} else {
		values = secret.Data
	}

	byteArr, err := getSecretValues(values, ref.MetadataPolicy)
	if err != nil {
		return nil, err
	}
	if ref.Property != "" {
		jsonStr := string(byteArr)
		// We need to search if a given key with a . exists before using gjson operations.
		idx := strings.Index(ref.Property, ".")
		if idx > -1 {
			refProperty := strings.ReplaceAll(ref.Property, ".", "\\.")
			val := gjson.Get(jsonStr, refProperty)
			if val.Exists() {
				return []byte(val.String()), nil
			}
		}
		val := gjson.Get(jsonStr, ref.Property)
		if !val.Exists() {
			return nil, fmt.Errorf("property %s does not exist in key %s", ref.Property, ref.Key)
		}

		return []byte(val.String()), nil
	}

	return byteArr, nil
}

func getSecretValues(secretMap map[string][]byte, policy esv1beta1.ExternalSecretMetadataPolicy) ([]byte, error) {
	var byteArr []byte
	var err error
	if policy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		data := make(map[string]json.RawMessage, len(secretMap))
		for k, v := range secretMap {
			data[k] = v
		}
		byteArr, err = json.Marshal(data)
	} else {
		strMap := make(map[string]string)
		for k, v := range secretMap {
			strMap[k] = string(v)
		}
		byteArr, err = json.Marshal(strMap)
	}

	if err != nil {
		return nil, fmt.Errorf("unabled to marshal json: %w", err)
	}

	return byteArr, nil
}

func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {
	if remoteRef.GetProperty() == "" {
		return fmt.Errorf("requires property in RemoteRef to delete secret value")
	}

	extSecret, getErr := c.userSecretClient.Get(ctx, remoteRef.GetRemoteKey(), metav1.GetOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesGetSecret, getErr)
	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// return gracefully if no secret exists
			return nil
		}
		return getErr
	}
	if _, ok := extSecret.Data[remoteRef.GetProperty()]; !ok {
		// return gracefully if specified secret does not contain the given property
		return nil
	}

	if len(extSecret.Data) > 1 {
		return c.removeProperty(ctx, extSecret, remoteRef)
	}
	return c.fullDelete(ctx, remoteRef.GetRemoteKey())
}

func (c *Client) PushSecret(ctx context.Context, value []byte, _ *apiextensionsv1.JSON, remoteRef esv1beta1.PushRemoteRef) error {
	if remoteRef.GetProperty() == "" {
		return fmt.Errorf("requires property in RemoteRef to push secret value")
	}
	extSecret, getErr := c.userSecretClient.Get(ctx, remoteRef.GetRemoteKey(), metav1.GetOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesGetSecret, getErr)
	if getErr != nil {
		// create if it not exists
		if apierrors.IsNotFound(getErr) {
			return c.createSecret(ctx, value, remoteRef)
		}
		return getErr
	}
	// return gracefully if data is already in sync
	if v, ok := extSecret.Data[remoteRef.GetProperty()]; ok && bytes.Equal(v, value) {
		return nil
	}

	// otherwise update remote property
	return c.updateProperty(ctx, extSecret, remoteRef, value)
}

func (c *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := c.userSecretClient.Get(ctx, ref.Key, metav1.GetOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesGetSecret, err)
	if apierrors.IsNotFound(err) {
		return nil, esv1beta1.NoSecretError{}
	}
	if err != nil {
		return nil, err
	}
	var tmpMap map[string][]byte
	if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		tmpMap, err = getSecretMetadata(secret)
		if err != nil {
			return nil, err
		}
	} else {
		tmpMap = secret.Data
	}

	if ref.Property != "" {
		retMap, err := getPropertyMap(ref.Key, ref.Property, tmpMap)
		if err != nil {
			return nil, err
		}
		return retMap, nil
	}

	return tmpMap, nil
}

func getPropertyMap(key, property string, tmpMap map[string][]byte) (map[string][]byte, error) {
	byteArr, err := json.Marshal(tmpMap)
	if err != nil {
		return nil, err
	}
	var retMap map[string][]byte
	jsonStr := string(byteArr)
	// We need to search if a given key with a . exists before using gjson operations.
	idx := strings.Index(property, ".")
	if idx > -1 {
		refProperty := strings.ReplaceAll(property, ".", "\\.")
		retMap, err = getMapFromValues(refProperty, jsonStr)
		if err != nil {
			return nil, err
		}
		if retMap != nil {
			return retMap, nil
		}
	}
	retMap, err = getMapFromValues(property, jsonStr)
	if err != nil {
		return nil, err
	}
	if retMap == nil {
		return nil, fmt.Errorf("property %s does not exist in key %s", property, key)
	}
	return retMap, nil
}

func getMapFromValues(property, jsonStr string) (map[string][]byte, error) {
	val := gjson.Get(jsonStr, property)
	if val.Exists() {
		retMap := make(map[string][]byte)
		var tmpMap map[string]interface{}
		decoded, err := base64.StdEncoding.DecodeString(val.String())
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(decoded, &tmpMap)
		if err != nil {
			return nil, err
		}
		for k, v := range tmpMap {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			retMap[k] = b
		}
		return retMap, nil
	}
	return nil, nil
}

func getSecretMetadata(secret *v1.Secret) (map[string][]byte, error) {
	var err error
	tmpMap := make(map[string][]byte)
	tmpMap[metaLabels], err = json.Marshal(secret.ObjectMeta.Labels)
	if err != nil {
		return nil, err
	}
	tmpMap[metaAnnotations], err = json.Marshal(secret.ObjectMeta.Annotations)
	if err != nil {
		return nil, err
	}

	return tmpMap, nil
}

func (c *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return c.findByTags(ctx, ref)
	}
	if ref.Name != nil {
		return c.findByName(ctx, ref)
	}
	return nil, fmt.Errorf("unexpected find operator: %#v", ref)
}

func (c *Client) findByTags(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// empty/nil tags = everything
	sel, err := labels.ValidatedSelectorFromSet(ref.Tags)
	if err != nil {
		return nil, fmt.Errorf("unable to validate selector tags: %w", err)
	}
	secrets, err := c.userSecretClient.List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesListSecrets, err)
	if err != nil {
		return nil, fmt.Errorf("unable to list secrets: %w", err)
	}
	data := make(map[string][]byte)
	for _, secret := range secrets.Items {
		jsonStr, err := json.Marshal(convertMap(secret.Data))
		if err != nil {
			return nil, err
		}
		data[secret.Name] = jsonStr
	}
	return utils.ConvertKeys(ref.ConversionStrategy, data)
}

func (c *Client) findByName(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	secrets, err := c.userSecretClient.List(ctx, metav1.ListOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesListSecrets, err)
	if err != nil {
		return nil, fmt.Errorf("unable to list secrets: %w", err)
	}
	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}
	data := make(map[string][]byte)
	for _, secret := range secrets.Items {
		if !matcher.MatchName(secret.Name) {
			continue
		}
		jsonStr, err := json.Marshal(convertMap(secret.Data))
		if err != nil {
			return nil, err
		}
		data[secret.Name] = jsonStr
	}
	return utils.ConvertKeys(ref.ConversionStrategy, data)
}

func (c Client) Close(_ context.Context) error {
	return nil
}

func convertMap(in map[string][]byte) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = string(v)
	}
	return out
}

func (c *Client) createSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {
	s := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteRef.GetRemoteKey(),
			Namespace: c.store.RemoteNamespace,
		},
		Data: map[string][]byte{remoteRef.GetProperty(): value},
		Type: "Opaque",
	}
	_, err := c.userSecretClient.Create(ctx, &s, metav1.CreateOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesCreateSecret, err)
	return err
}

// fullDelete removes remote secret completely.
func (c *Client) fullDelete(ctx context.Context, secretName string) error {
	err := c.userSecretClient.Delete(ctx, secretName, metav1.DeleteOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesDeleteSecret, err)

	// gracefully return on not found
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// removeProperty removes single data property from remote secret.
func (c *Client) removeProperty(ctx context.Context, extSecret *v1.Secret, remoteRef esv1beta1.PushRemoteRef) error {
	delete(extSecret.Data, remoteRef.GetProperty())
	_, err := c.userSecretClient.Update(ctx, extSecret, metav1.UpdateOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesUpdateSecret, err)
	return err
}

func (c *Client) updateProperty(ctx context.Context, extSecret *v1.Secret, remoteRef esv1beta1.PushRemoteRef, value []byte) error {
	// otherwise update remote secret
	extSecret.Data[remoteRef.GetProperty()] = value
	_, uErr := c.userSecretClient.Update(ctx, extSecret, metav1.UpdateOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesUpdateSecret, uErr)
	return uErr
}
