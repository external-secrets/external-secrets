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
package parameterstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/tidwall/gjson"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1alpha2 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha2"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// ParameterStore is a provider for AWS ParameterStore.
type ParameterStore struct {
	client PMInterface
}

// PMInterface is a subset of the parameterstore api.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/ssm/ssmiface/
type PMInterface interface {
	GetParameter(*ssm.GetParameterInput) (*ssm.GetParameterOutput, error)
}

var log = ctrl.Log.WithName("provider").WithName("aws").WithName("parameterstore")

// New constructs a ParameterStore Provider that is specific to a store.
func New(sess client.ConfigProvider) (*ParameterStore, error) {
	return &ParameterStore{
		client: ssm.New(sess),
	}, nil
}

// GetSecret returns a single secret from the provider.
func (pm *ParameterStore) GetSecret(ctx context.Context, ref esv1alpha2.ExternalSecretDataRemoteRef) ([]byte, error) {
	log.Info("fetching secret value", "key", ref.Key)
	out, err := pm.client.GetParameter(&ssm.GetParameterInput{
		Name:           &ref.Key,
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, util.SanitizeErr(err)
	}
	if ref.Property == "" {
		if out.Parameter.Value != nil {
			return []byte(*out.Parameter.Value), nil
		}
		return nil, fmt.Errorf("invalid secret received. parameter value is nil for key: %s", ref.Key)
	}
	val := gjson.Get(*out.Parameter.Value, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// Implements store.Client.GetAllSecrets Interface.
// New version of GetAllSecrets.
func (pm *ParameterStore) GetAllSecrets(ctx context.Context, ref esv1alpha2.ExternalSecretDataFromRemoteRef) (map[string][]byte, error) {
	// TO be implemented
	return nil, utils.ThrowNotImplemented()
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (pm *ParameterStore) GetSecretMap(ctx context.Context, ref esv1alpha2.ExternalSecretDataFromRemoteRef) (map[string][]byte, error) {
	log.Info("fetching secret map", "key", ref.Extract.Key)
	data, err := pm.GetSecret(ctx, ref.GetDataRemoteRef())
	if err != nil {
		return nil, err
	}
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Extract.Key, err)
	}
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}
	return secretData, nil
}

func (pm *ParameterStore) Close(ctx context.Context) error {
	return nil
}
