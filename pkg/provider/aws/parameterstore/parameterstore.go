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

	"github.com/aws/aws-sdk-go/service/ssm"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	awssess "github.com/external-secrets/external-secrets/pkg/provider/aws/session"
)

// ParameterStore is a provider for AWS ParameterStore.
type ParameterStore struct {
	stsProvider awssess.STSProvider
	// session     *session.Session
	// client      PMInterface
}

// PMInterface is a subset of the parameterstore api.
// see: https://docs.aws.amazon.com/sdk-for-go/api/service/ssm/ssmiface/
type PMInterface interface {
	GetParameter(*ssm.GetParameterInput) (*ssm.GetParameterOutput, error)
}

var log = ctrl.Log.WithName("provider").WithName("aws").WithName("parameterstore")

// New constructs a ParameterStore Provider that is specific to a store.
func New(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string, stsProvider awssess.STSProvider) (provider.SecretsClient, error) {
	pm := &ParameterStore{
		stsProvider: stsProvider,
	}
	return pm, nil
}

// GetSecret returns a single secret from the provider.
func (pm *ParameterStore) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	log.Info("fetching secret value", "key", ref.Key)
	return []byte("NOOP"), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (pm *ParameterStore) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return map[string][]byte{"NOOP": []byte("NOOP")}, nil
}
