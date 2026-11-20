/*
Copyright © The ESO Authors

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

package ibm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"

	"github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/metrics"
	"github.com/external-secrets/external-secrets/runtime/util/locks"
)

const (
	managedByKey   = "managed-by"
	managedByValue = "external-secrets"

	// defaultSecretGroup is both the name and the ID of the default secret group.
	defaultSecretGroup = "default"
)

var errNotManagedByESO = errors.New("secret not managed by external-secrets")

// remoteSecret holds the parts of an arbitrary or kv secret that push and delete need.
type remoteSecret struct {
	id             string
	customMetadata map[string]any
	payload        string
	data           map[string]any
}

func isNotFound(resp *core.DetailedResponse) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
}

func isManagedByESO(customMetadata map[string]any) bool {
	v, ok := customMetadata[managedByKey].(string)
	return ok && v == managedByValue
}

// parsePushRef splits a remote key using the same group/type/name format as GetSecret,
// limited to the secret types that hold user-provided values.
func parsePushRef(remoteKey, property string) (secretGroupName, secretType, secretName string, err error) {
	secretGroupName, secretType, secretName = parseSecretReference(remoteKey)
	switch secretType {
	case sm.Secret_SecretType_Arbitrary:
		if property != "" {
			return "", "", "", fmt.Errorf("property is not supported for secret type %s", secretType)
		}
	case sm.Secret_SecretType_Kv:
	default:
		return "", "", "", fmt.Errorf("push is not supported for secret type %s", secretType)
	}
	return secretGroupName, secretType, secretName, nil
}

func (ibm *providerIBM) getRemoteSecret(secretGroupName, secretType, secretName string) (*remoteSecret, error) {
	response, err := getSecretData(ibm, &secretName, secretType, secretGroupName)
	if err != nil {
		return nil, err
	}
	// a lookup by ID returns whatever type the secret has, so check it matches the key
	switch s := response.(type) {
	case *sm.ArbitrarySecret:
		if secretType == sm.Secret_SecretType_Arbitrary {
			return &remoteSecret{id: *s.ID, customMetadata: s.CustomMetadata, payload: core.StringNilMapper(s.Payload)}, nil
		}
	case *sm.KVSecret:
		if secretType == sm.Secret_SecretType_Kv {
			return &remoteSecret{id: *s.ID, customMetadata: s.CustomMetadata, data: s.Data}, nil
		}
	}
	return nil, fmt.Errorf(errExtractingSecret, secretName, secretType, "PushSecret")
}

// kvData builds the data of a kv secret: the value is a JSON object when no property is
// given, otherwise it is stored as a string under that property, keeping the other keys.
func kvData(existing map[string]any, value []byte, property string) (map[string]any, error) {
	if property == "" {
		data := make(map[string]any)
		if err := json.Unmarshal(value, &data); err != nil {
			return nil, fmt.Errorf("kv secrets need a JSON object when no property is set: %w", err)
		}
		return data, nil
	}
	data := maps.Clone(existing)
	if data == nil {
		data = make(map[string]any)
	}
	data[property] = string(value)
	return data, nil
}

// PushSecret creates or updates an arbitrary or kv secret in IBM Cloud Secrets Manager.
func (ibm *providerIBM) PushSecret(ctx context.Context, secret *corev1.Secret, psd esv1.PushSecretData) error {
	if esutils.IsNil(ibm.IBMClient) {
		return errors.New(errUninitializedIBMProvider)
	}
	secretGroupName, secretType, secretName, err := parsePushRef(psd.GetRemoteKey(), psd.GetProperty())
	if err != nil {
		return err
	}
	value, err := esutils.ExtractSecretData(psd, secret)
	if err != nil {
		return err
	}

	// kv secrets can be shared by several PushSecrets and IBM has no conditional writes
	unlock, err := locks.TryLock(ProviderIBMSM, psd.GetRemoteKey())
	if err != nil {
		return err
	}
	defer unlock()

	existing, err := ibm.getRemoteSecret(secretGroupName, secretType, secretName)
	if errors.Is(err, esv1.NoSecretErr) {
		if _, uuidErr := uuid.Parse(secretName); uuidErr == nil {
			return fmt.Errorf("secret %s does not exist and cannot be created by ID, use <secret_group_name>/%s/<secret_name>", secretName, secretType)
		}
		return ibm.createSecret(ctx, secretGroupName, secretType, secretName, value, psd.GetProperty())
	}
	if err != nil {
		return err
	}
	if !isManagedByESO(existing.customMetadata) {
		return errNotManagedByESO
	}

	var version sm.SecretVersionPrototypeIntf
	switch secretType {
	case sm.Secret_SecretType_Arbitrary:
		payload := string(value)
		if payload == existing.payload {
			return nil
		}
		version = &sm.ArbitrarySecretVersionPrototype{Payload: &payload}
	case sm.Secret_SecretType_Kv:
		data, err := kvData(existing.data, value, psd.GetProperty())
		if err != nil {
			return err
		}
		if reflect.DeepEqual(data, existing.data) {
			return nil
		}
		version = &sm.KVSecretVersionPrototype{Data: data}
	}
	return ibm.createSecretVersion(ctx, existing.id, version)
}

func (ibm *providerIBM) createSecret(ctx context.Context, secretGroupName, secretType, secretName string, value []byte, property string) error {
	groupID, err := ibm.secretGroupID(ctx, secretGroupName)
	if err != nil {
		return err
	}
	customMetadata := map[string]any{managedByKey: managedByValue}

	var prototype sm.SecretPrototypeIntf
	switch secretType {
	case sm.Secret_SecretType_Arbitrary:
		prototype = &sm.ArbitrarySecretPrototype{
			Name:           &secretName,
			SecretType:     new(sm.ArbitrarySecretPrototype_SecretType_Arbitrary),
			SecretGroupID:  &groupID,
			Payload:        new(string(value)),
			CustomMetadata: customMetadata,
		}
	case sm.Secret_SecretType_Kv:
		data, err := kvData(nil, value, property)
		if err != nil {
			return err
		}
		prototype = &sm.KVSecretPrototype{
			Name:           &secretName,
			SecretType:     new(sm.KVSecretPrototype_SecretType_Kv),
			SecretGroupID:  &groupID,
			Data:           data,
			CustomMetadata: customMetadata,
		}
	}

	ctx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()
	_, _, err = ibm.IBMClient.CreateSecretWithContext(ctx, &sm.CreateSecretOptions{SecretPrototype: prototype})
	metrics.ObserveAPICall(ProviderIBMSM, CallIBMSMCreateSecret, err)
	return err
}

func (ibm *providerIBM) createSecretVersion(ctx context.Context, secretID string, version sm.SecretVersionPrototypeIntf) error {
	ctx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()
	_, _, err := ibm.IBMClient.CreateSecretVersionWithContext(ctx, &sm.CreateSecretVersionOptions{
		SecretID:               &secretID,
		SecretVersionPrototype: version,
	})
	metrics.ObserveAPICall(ProviderIBMSM, CallIBMSMCreateSecretVersion, err)
	return err
}

// secretGroupID resolves a secret group name to the ID that CreateSecret needs.
func (ibm *providerIBM) secretGroupID(ctx context.Context, secretGroupName string) (string, error) {
	if secretGroupName == defaultSecretGroup {
		return defaultSecretGroup, nil
	}
	ctx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()
	groups, _, err := ibm.IBMClient.ListSecretGroupsWithContext(ctx, &sm.ListSecretGroupsOptions{})
	metrics.ObserveAPICall(ProviderIBMSM, CallIBMSMListSecretGroups, err)
	if err != nil {
		return "", err
	}
	for _, group := range groups.SecretGroups {
		if group.Name != nil && *group.Name == secretGroupName && group.ID != nil {
			return *group.ID, nil
		}
	}
	return "", fmt.Errorf("secret group %s not found", secretGroupName)
}

// DeleteSecret deletes a secret created by PushSecret, or removes one property from a kv secret.
func (ibm *providerIBM) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	if esutils.IsNil(ibm.IBMClient) {
		return errors.New(errUninitializedIBMProvider)
	}
	secretGroupName, secretType, secretName, err := parsePushRef(remoteRef.GetRemoteKey(), remoteRef.GetProperty())
	if err != nil {
		return err
	}
	unlock, err := locks.TryLock(ProviderIBMSM, remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}
	defer unlock()

	existing, err := ibm.getRemoteSecret(secretGroupName, secretType, secretName)
	if errors.Is(err, esv1.NoSecretErr) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isManagedByESO(existing.customMetadata) {
		return nil
	}

	if property := remoteRef.GetProperty(); property != "" {
		if _, ok := existing.data[property]; !ok {
			return nil
		}
		data := maps.Clone(existing.data)
		delete(data, property)
		// other properties may belong to other PushSecrets, so keep the secret until the last one goes
		if len(data) > 0 {
			return ibm.createSecretVersion(ctx, existing.id, &sm.KVSecretVersionPrototype{Data: data})
		}
	}

	ctx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()
	resp, err := ibm.IBMClient.DeleteSecretWithContext(ctx, &sm.DeleteSecretOptions{ID: &existing.id})
	metrics.ObserveAPICall(ProviderIBMSM, CallIBMSMDeleteSecret, err)
	if isNotFound(resp) {
		return nil
	}
	return err
}

// SecretExists reports whether the secret behind a PushSecret remote key exists,
// and for a kv property, whether the secret holds that key.
func (ibm *providerIBM) SecretExists(_ context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	if esutils.IsNil(ibm.IBMClient) {
		return false, errors.New(errUninitializedIBMProvider)
	}
	secretGroupName, secretType, secretName, err := parsePushRef(remoteRef.GetRemoteKey(), remoteRef.GetProperty())
	if err != nil {
		return false, err
	}
	existing, err := ibm.getRemoteSecret(secretGroupName, secretType, secretName)
	if errors.Is(err, esv1.NoSecretErr) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if property := remoteRef.GetProperty(); property != "" {
		_, ok := existing.data[property]
		return ok, nil
	}
	return true, nil
}
