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

package secretsmanager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/tidwall/gjson"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
)

// SecretsManager is a provider for AWS SecretsManager.
type SecretsManager struct {
	sess   *session.Session
	client SMInterface
	cache  map[string]*awssm.GetSecretValueOutput
}

// SMInterface is a subset of the smiface api.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/secretsmanager/secretsmanageriface/
type SMInterface interface {
	GetSecretValue(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
}

var log = ctrl.Log.WithName("provider").WithName("aws").WithName("secretsmanager")

// New creates a new SecretsManager client.
func New(sess *session.Session) (*SecretsManager, error) {
	return &SecretsManager{
		sess:   sess,
		client: awssm.New(sess),
		cache:  make(map[string]*awssm.GetSecretValueOutput),
	}, nil
}

func (sm *SecretsManager) fetch(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (*awssm.GetSecretValueOutput, error) {
	ver := "AWSCURRENT"
	if ref.Version != "" {
		ver = ref.Version
	}
	log.Info("fetching secret value", "key", ref.Key, "version", ver)

	cacheKey := fmt.Sprintf("%s#%s", ref.Key, ver)
	if secretOut, found := sm.cache[cacheKey]; found {
		log.Info("found secret in cache", "key", ref.Key, "version", ver)
		return secretOut, nil
	}
	secretOut, err := sm.client.GetSecretValue(&awssm.GetSecretValueInput{
		SecretId:     &ref.Key,
		VersionStage: &ver,
	})
	if err != nil {
		return nil, err
	}
	sm.cache[cacheKey] = secretOut

	return secretOut, nil
}

// Empty GetAllSecrets.
func (sm *SecretsManager) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (sm *SecretsManager) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secretOut, err := sm.fetch(ctx, ref)
	if err != nil {
		return nil, util.SanitizeErr(err)
	}
	if ref.Property == "" {
		if secretOut.SecretString != nil {
			return []byte(*secretOut.SecretString), nil
		}
		if secretOut.SecretBinary != nil {
			return secretOut.SecretBinary, nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string nor binary for key: %s", ref.Key)
	}
	var payload string
	if secretOut.SecretString != nil {
		payload = *secretOut.SecretString
	}
	if secretOut.SecretBinary != nil {
		payload = string(secretOut.SecretBinary)
	}

	val := gjson.Get(payload, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (sm *SecretsManager) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	log.Info("fetching secret map", "key", ref.Key)
	data, err := sm.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Key, err)
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

func (sm *SecretsManager) Close(ctx context.Context) error {
	return nil
}

func (sm *SecretsManager) Validate() error {
	_, err := sm.sess.Config.Credentials.Get()
	return err
}
