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

package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
	"github.com/external-secrets/external-secrets/runtime/find"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	metaLabels      = "labels"
	metaAnnotations = "annotations"
)

// GetSecret retrieves a secret from the Kubernetes API server by its key.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.userSecretClient.Get(ctx, ref.Key, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// if property is not defined, we will return the json-serialized secret
	if ref.Property == "" {
		if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
			m := map[string]map[string]string{}
			m[metaLabels] = secret.Labels
			m[metaAnnotations] = secret.Annotations

			j, err := esutils.JSONMarshal(m)
			if err != nil {
				return nil, err
			}
			return j, nil
		}

		m := map[string]string{}
		for key, val := range secret.Data {
			m[key] = string(val)
		}
		j, err := esutils.JSONMarshal(m)
		if err != nil {
			return nil, err
		}
		return j, nil
	}

	return getSecret(secret, ref)
}

// DeleteSecret removes a secret value from Kubernetes.
// It requires a property to be specified in the RemoteRef.
func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	extSecret, getErr := c.userSecretClient.Get(ctx, remoteRef.GetRemoteKey(), metav1.GetOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesGetSecret, getErr)
	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// return gracefully if no secret exists
			return nil
		}
		return getErr
	}
	if remoteRef.GetProperty() != "" {
		if _, ok := extSecret.Data[remoteRef.GetProperty()]; !ok {
			// return gracefully if specified secret does not contain the given property
			return nil
		}

		if len(extSecret.Data) > 1 {
			return c.removeProperty(ctx, extSecret, remoteRef)
		}
	}
	return c.fullDelete(ctx, remoteRef.GetRemoteKey())
}

// SecretExists checks if a secret exists in Kubernetes.
// This method is not implemented and always returns an error.
func (c *Client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

// PushSecret creates or updates a secret in Kubernetes.
func (c *Client) PushSecret(ctx context.Context, secret *v1.Secret, data esv1.PushSecretData) error {
	if data.GetProperty() == "" && data.GetSecretKey() != "" {
		return errors.New("requires property in RemoteRef to push secret value if secret key is defined")
	}

	targetNamespace := c.store.RemoteNamespace
	pushMeta, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](data.GetMetadata())
	if err != nil {
		return fmt.Errorf("unable to parse metadata parameters: %w", err)
	}
	if pushMeta != nil && pushMeta.Spec.RemoteNamespace != "" {
		if c.storeKind != esv1.ClusterSecretStoreKind {
			return fmt.Errorf("remoteNamespace override is only supported with ClusterSecretStore")
		}
		targetNamespace = pushMeta.Spec.RemoteNamespace
	}

	secretClient := c.secretsClientFor(targetNamespace)
	remoteSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: targetNamespace,
			Name:      data.GetRemoteKey(),
		},
	}

	return c.createOrUpdate(ctx, secretClient, remoteSecret, func() error {
		return c.mergePushSecretData(data, pushMeta, remoteSecret, secret)
	})
}

func (c *Client) mergePushSecretData(remoteRef esv1.PushSecretData, pushMeta *metadata.PushSecretMetadata[PushSecretMetadataSpec], remoteSecret, localSecret *v1.Secret) error {
	secretType := v1.SecretTypeOpaque
	if localSecret.Type != "" {
		secretType = localSecret.Type
	}
	remoteSecret.Type = secretType

	if remoteSecret.Data == nil {
		remoteSecret.Data = make(map[string][]byte)
	}

	var targetLabels, targetAnnotations map[string]string
	sourceLabels, sourceAnnotations, err := mergeSourceMetadata(localSecret, pushMeta)
	if err != nil {
		return fmt.Errorf("failed to merge source metadata: %w", err)
	}
	targetLabels, targetAnnotations, err = mergeTargetMetadata(remoteSecret, pushMeta, sourceLabels, sourceAnnotations)
	if err != nil {
		return fmt.Errorf("failed to merge target metadata: %w", err)
	}
	remoteSecret.ObjectMeta.Labels = targetLabels
	remoteSecret.ObjectMeta.Annotations = targetAnnotations

	// case 1: push the whole secret
	if remoteRef.GetProperty() == "" {
		for k, v := range localSecret.Data {
			remoteSecret.Data[k] = v
		}
		return nil
	}

	// cases 2a + 2b: push into a property.
	// if secret key is empty, we will marshal the whole secret and put it into
	// the property defined in the remoteRef.
	if remoteRef.GetSecretKey() == "" {
		value, err := c.marshalData(localSecret)
		if err != nil {
			return err
		}
		remoteSecret.Data[remoteRef.GetProperty()] = value
	} else {
		// if secret key is defined, we will push that key from the local secret
		remoteSecret.Data[remoteRef.GetProperty()] = localSecret.Data[remoteRef.GetSecretKey()]
	}
	return nil
}

func (c *Client) createOrUpdate(ctx context.Context, secretClient KClient, targetSecret *v1.Secret, f func() error) error {
	target, err := secretClient.Get(ctx, targetSecret.Name, metav1.GetOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesGetSecret, err)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if err := f(); err != nil {
			return err
		}
		_, err := secretClient.Create(ctx, targetSecret, metav1.CreateOptions{})
		metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesCreateSecret, err)
		if err != nil {
			return err
		}
		return nil
	}

	*targetSecret = *target
	existing := targetSecret.DeepCopyObject()
	if err := f(); err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(existing, targetSecret) {
		return nil
	}

	_, err = secretClient.Update(ctx, targetSecret, metav1.UpdateOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesUpdateSecret, err)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) marshalData(secret *v1.Secret) ([]byte, error) {
	values := make(map[string]string)
	for k, v := range secret.Data {
		values[k] = string(v)
	}

	// marshal
	value, err := esutils.JSONMarshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets into a single property: %w", err)
	}

	return value, nil
}

// GetSecretMap retrieves a secret from Kubernetes and returns it as a map.
// The secret data is converted to a map of key/value pairs.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := c.userSecretClient.Get(ctx, ref.Key, metav1.GetOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesGetSecret, err)
	if apierrors.IsNotFound(err) {
		return nil, esv1.NoSecretError{}
	}
	if err != nil {
		return nil, err
	}
	var tmpMap map[string][]byte
	if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
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
	byteArr, err := esutils.JSONMarshal(tmpMap)
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
		var tmpMap map[string]any
		decoded, err := base64.StdEncoding.DecodeString(val.String())
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(decoded, &tmpMap)
		if err != nil {
			// Do not return the raw error as json.Unmarshal errors may contain
			// sensitive secret data in the error message
			return nil, errors.New("failed to unmarshal secret: invalid JSON format")
		}
		for k, v := range tmpMap {
			b, err := esutils.JSONMarshal(v)
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
	tmpMap[metaLabels], err = esutils.JSONMarshal(secret.ObjectMeta.Labels)
	if err != nil {
		return nil, err
	}
	tmpMap[metaAnnotations], err = esutils.JSONMarshal(secret.ObjectMeta.Annotations)
	if err != nil {
		return nil, err
	}

	return tmpMap, nil
}

// GetAllSecrets retrieves multiple secrets from Kubernetes based on the search criteria.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return c.findByTags(ctx, ref)
	}
	if ref.Name != nil {
		return c.findByName(ctx, ref)
	}
	return nil, fmt.Errorf("unexpected find operator: %#v", ref)
}

func (c *Client) findByTags(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
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
		jsonStr, err := esutils.JSONMarshal(convertMap(secret.Data))
		if err != nil {
			return nil, err
		}
		data[secret.Name] = jsonStr
	}
	return esutils.ConvertKeys(ref.ConversionStrategy, data)
}

func (c *Client) findByName(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
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
		jsonStr, err := esutils.JSONMarshal(convertMap(secret.Data))
		if err != nil {
			return nil, err
		}
		data[secret.Name] = jsonStr
	}
	return esutils.ConvertKeys(ref.ConversionStrategy, data)
}

// Close implements cleanup operations for the Kubernetes client.
func (c *Client) Close(_ context.Context) error {
	return nil
}

func convertMap(in map[string][]byte) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = string(v)
	}
	return out
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
func (c *Client) removeProperty(ctx context.Context, extSecret *v1.Secret, remoteRef esv1.PushSecretRemoteRef) error {
	delete(extSecret.Data, remoteRef.GetProperty())
	_, err := c.userSecretClient.Update(ctx, extSecret, metav1.UpdateOptions{})
	metrics.ObserveAPICall(constants.ProviderKubernetes, constants.CallKubernetesUpdateSecret, err)
	return err
}

func getSecret(secret *v1.Secret, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
		s, found, err := getFromSecretMetadata(secret, ref)
		if err != nil {
			return nil, err
		}

		if !found {
			return nil, fmt.Errorf("property %s does not exist in metadata of secret %q", ref.Property, ref.Key)
		}

		return s, nil
	}

	s, found := getFromSecretData(secret, ref)
	if !found {
		return nil, fmt.Errorf("property %s does not exist in data of secret %q", ref.Property, ref.Key)
	}

	return s, nil
}

func getFromSecretData(secret *v1.Secret, ref esv1.ExternalSecretDataRemoteRef) ([]byte, bool) {
	// Check if a property with "." exists first such as file.png
	v, ok := secret.Data[ref.Property]
	if ok {
		return v, true
	}

	idx := strings.Index(ref.Property, ".")
	if idx == -1 || idx == 0 || idx == len(ref.Property)-1 {
		return nil, false
	}

	v, ok = secret.Data[ref.Property[:idx]]
	if !ok {
		return nil, false
	}

	val := gjson.Get(string(v), ref.Property[idx+1:])
	if !val.Exists() {
		return nil, false
	}

	return []byte(val.String()), true
}

func getFromSecretMetadata(secret *v1.Secret, ref esv1.ExternalSecretDataRemoteRef) ([]byte, bool, error) {
	path := strings.Split(ref.Property, ".")

	var metadata map[string]string
	switch path[0] {
	case metaLabels:
		metadata = secret.Labels
	case metaAnnotations:
		metadata = secret.Annotations
	default:
		return nil, false, nil
	}

	if len(path) == 1 {
		j, err := esutils.JSONMarshal(metadata)
		if err != nil {
			return nil, false, err
		}
		return j, true, nil
	}

	v, ok := metadata[path[1]]
	if !ok {
		return nil, false, nil
	}
	if len(path) == 2 {
		return []byte(v), true, nil
	}

	val := gjson.Get(v, strings.Join(path[2:], ""))
	if !val.Exists() {
		return nil, false, nil
	}

	return []byte(val.String()), true, nil
}
