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

package bitwarden

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kube-openapi/pkg/validation/strfmt"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	// NoteMetadataKey defines the note for the pushed secret.
	NoteMetadataKey = "note"
)

// PushSecret will write a single secret into the provider.
// Note: We will refuse to overwrite ANY secrets, because we can never be completely
// sure if it's the same secret we are trying to push. We only have the Name and the value
// could be different. Therefore, we will always create a new secret. Except if, the value
// the key, the note, and organization ID all match.
// We only allow to push to a single project, because GET returns a single project ID
// the secret belongs to even though technically Create allows multiple projects. This is
// to ensure that we push to the same project always, and so we can determine reliably that
// we don't need to push again.
func (p *Provider) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	spec := p.store.GetSpec()
	if spec == nil || spec.Provider == nil {
		return errors.New("store does not have a provider")
	}

	if data.GetSecretKey() == "" {
		return errors.New("pushing the whole secret is not yet implemented")
	}

	if data.GetRemoteKey() == "" {
		return errors.New("remote key must be defined")
	}

	value, ok := secret.Data[data.GetSecretKey()]
	if !ok {
		return fmt.Errorf("failed to find secret key in secret with key: %s", data.GetSecretKey())
	}

	note, err := utils.FetchValueFromMetadata(NoteMetadataKey, data.GetMetadata(), "")
	if err != nil {
		return fmt.Errorf("failed to fetch note from metadata: %w", err)
	}

	// ListAll Secrets for an organization. If the key matches our key, we GetSecret that and do a compare.
	remoteSecrets, err := p.bitwardenSdkClient.ListSecrets(ctx, spec.Provider.BitwardenSecretsManager.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get all secrets: %w", err)
	}

	for _, d := range remoteSecrets.Data {
		if d.Key != data.GetRemoteKey() {
			continue
		}

		sec, err := p.bitwardenSdkClient.GetSecret(ctx, d.ID)
		if err != nil {
			return fmt.Errorf("failed to get secret: %w", err)
		}

		// If all pushed data matches, we won't push this secret.
		if sec.Key == data.GetRemoteKey() &&
			sec.Value == string(value) &&
			sec.Note == note &&
			sec.ProjectID != nil &&
			*sec.ProjectID == spec.Provider.BitwardenSecretsManager.ProjectID {
			// we have a complete match, skip pushing.
			return nil
		} else if sec.Key == data.GetRemoteKey() &&
			sec.Value != string(value) &&
			sec.Note == note &&
			sec.ProjectID != nil &&
			*sec.ProjectID == spec.Provider.BitwardenSecretsManager.ProjectID {
			// only the value is different, update the existing secret.
			_, err = p.bitwardenSdkClient.UpdateSecret(ctx, SecretPutRequest{
				ID:             sec.ID,
				Key:            data.GetRemoteKey(),
				Note:           note,
				OrganizationID: spec.Provider.BitwardenSecretsManager.OrganizationID,
				ProjectIDS:     []string{spec.Provider.BitwardenSecretsManager.ProjectID},
				Value:          string(value),
			})

			return err
		}
	}

	// no matching secret found, let's create it
	_, err = p.bitwardenSdkClient.CreateSecret(ctx, SecretCreateRequest{
		Key:            data.GetRemoteKey(),
		Note:           note,
		OrganizationID: spec.Provider.BitwardenSecretsManager.OrganizationID,
		ProjectIDS:     []string{spec.Provider.BitwardenSecretsManager.ProjectID},
		Value:          string(value),
	})

	return err
}

// GetSecret returns a single secret from the provider.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if strfmt.IsUUID(ref.Key) {
		resp, err := p.bitwardenSdkClient.GetSecret(ctx, ref.Key)
		if err != nil {
			return nil, fmt.Errorf("error getting secret: %w", err)
		}

		return []byte(resp.Value), nil
	}

	spec := p.store.GetSpec()
	if spec == nil || spec.Provider == nil {
		return nil, errors.New("store does not have a provider")
	}

	secret, err := p.findSecretByRef(ctx, ref.Key, spec.Provider.BitwardenSecretsManager.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("error getting secret: %w", err)
	}

	// we found our secret, return the value for it
	return []byte(secret.Value), nil
}

func (p *Provider) DeleteSecret(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) error {
	if strfmt.IsUUID(ref.GetRemoteKey()) {
		return p.deleteSecret(ctx, ref.GetRemoteKey())
	}

	spec := p.store.GetSpec()
	if spec == nil || spec.Provider == nil {
		return errors.New("store does not have a provider")
	}

	secret, err := p.findSecretByRef(ctx, ref.GetRemoteKey(), spec.Provider.BitwardenSecretsManager.ProjectID)
	if err != nil {
		return fmt.Errorf("error getting secret: %w", err)
	}

	return p.deleteSecret(ctx, secret.ID)
}

func (p *Provider) deleteSecret(ctx context.Context, id string) error {
	resp, err := p.bitwardenSdkClient.DeleteSecret(ctx, []string{id})
	if err != nil {
		return fmt.Errorf("error deleting secret: %w", err)
	}

	var errs error
	for _, data := range resp.Data {
		if data.Error != nil {
			errs = errors.Join(errs, fmt.Errorf("error deleting secret with id %s: %s", data.ID, *data.Error))
		}
	}

	if errs != nil {
		return fmt.Errorf("there were one or more errors deleting secrets: %w", errs)
	}
	return nil
}

func (p *Provider) SecretExists(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) (bool, error) {
	if strfmt.IsUUID(ref.GetRemoteKey()) {
		_, err := p.bitwardenSdkClient.GetSecret(ctx, ref.GetRemoteKey())
		if err != nil {
			return false, fmt.Errorf("error getting secret: %w", err)
		}

		return true, nil
	}

	spec := p.store.GetSpec()
	if spec == nil || spec.Provider == nil {
		return false, errors.New("store does not have a provider")
	}

	if _, err := p.findSecretByRef(ctx, ref.GetRemoteKey(), spec.Provider.BitwardenSecretsManager.ProjectID); err != nil {
		return false, fmt.Errorf("error getting secret: %w", err)
	}

	return true, nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := p.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling secret: %w", err)
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

// GetAllSecrets gets multiple secrets from the provider and loads into a kubernetes secret.
// First load all secrets from secretStore path configuration
// Then, gets secrets from a matching name or matching custom_metadata.
func (p *Provider) GetAllSecrets(ctx context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	spec := p.store.GetSpec()
	if spec == nil {
		return nil, errors.New("store does not have a provider")
	}

	secrets, err := p.bitwardenSdkClient.ListSecrets(ctx, spec.Provider.BitwardenSecretsManager.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all secrets: %w", err)
	}

	result := map[string][]byte{}
	for _, d := range secrets.Data {
		sec, err := p.bitwardenSdkClient.GetSecret(ctx, d.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret: %w", err)
		}

		result[d.ID] = []byte(sec.Value)
	}

	return result, nil
}

// Validate validates the provider.
func (p *Provider) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

// Close closes the provider.
func (p *Provider) Close(_ context.Context) error {
	return nil
}

func (p *Provider) findSecretByRef(ctx context.Context, key, projectID string) (*SecretResponse, error) {
	spec := p.store.GetSpec()
	if spec == nil || spec.Provider == nil {
		return nil, errors.New("store does not have a provider")
	}

	// ListAll Secrets for an organization. If the key matches our key, we GetSecret that and do a compare.
	secrets, err := p.bitwardenSdkClient.ListSecrets(ctx, spec.Provider.BitwardenSecretsManager.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all secrets: %w", err)
	}

	var remoteSecret *SecretResponse
	for _, d := range secrets.Data {
		if d.Key != key {
			continue
		}

		sec, err := p.bitwardenSdkClient.GetSecret(ctx, d.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret: %w", err)
		}

		if sec.ProjectID != nil && *sec.ProjectID == projectID {
			if remoteSecret != nil {
				return nil, fmt.Errorf("more than one secret found for project %s with key %s", projectID, key)
			}

			// We don't break here because we WANT TO MAKE SURE that there is ONLY ONE
			// such secret.
			remoteSecret = sec
		}
	}

	if remoteSecret == nil {
		return nil, fmt.Errorf("no secret found for project id %s and name %s", projectID, key)
	}

	return remoteSecret, nil
}
