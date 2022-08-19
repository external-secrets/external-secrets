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

package fake

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/google/go-cmp/cmp"
)

// Client implements the aws secretsmanager interface.
type Client struct {
	ExecutionCounter int
	valFn            map[string]func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
}

type GetSecretValueFn func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)
type ListSecretsFn func(*awssm.ListSecretsInput) (*awssm.ListSecretsOutput, error)
type CreateSecretWithContextFn func(aws.Context, *awssm.CreateSecretInput, ...request.Option) (*awssm.CreateSecretOutput, error)

type SMInterface struct {
	GetSecretValueFn          GetSecretValueFn
	ListSecretsFn             ListSecretsFn
	CreateSecretWithContextFn CreateSecretWithContextFn
}

func (sm SMInterface) CreateSecretWithContext(ctx aws.Context, input *awssm.CreateSecretInput, options ...request.Option) (*awssm.CreateSecretOutput, error) {
	return sm.CreateSecretWithContextFn(ctx, input, options...)
}

func NewCreateSecretWithContextFn(output *awssm.CreateSecretOutput, err error) CreateSecretWithContextFn {
	return func(ctx aws.Context, input *awssm.CreateSecretInput, options ...request.Option) (*awssm.CreateSecretOutput, error) {
		return output, err
	}
}

func (sm SMInterface) GetSecretValue(input *awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
	return sm.GetSecretValueFn(input)
}

func NewGetSecretValueFn(output *awssm.GetSecretValueOutput, err error) GetSecretValueFn {
	return func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
		return output, err
	}
}

func (sm SMInterface) ListSecrets(input *awssm.ListSecretsInput) (*awssm.ListSecretsOutput, error) {
	return sm.ListSecretsFn(input)
}

func NewListSecretsFn(listOutput *awssm.ListSecretsOutput, err error) ListSecretsFn {
	return func(*awssm.ListSecretsInput) (*awssm.ListSecretsOutput, error) {
		return listOutput, err
	}
}

// NewClient init a new fake client.
func NewClient() *Client {
	return &Client{
		valFn: make(map[string]func(*awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error)),
	}
}

func (sm *Client) CreateSecretWithContext(aws.Context, *awssm.CreateSecretInput, ...request.Option) (*awssm.CreateSecretOutput, error) {
	value := "I'm a key"
	output := awssm.CreateSecretOutput{
		Name: &value,
	}
	return &output, nil
}

func (sm *Client) GetSecretValue(in *awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
	sm.ExecutionCounter++
	if entry, found := sm.valFn[sm.cacheKeyForInput(in)]; found {
		return entry(in)
	}
	return nil, fmt.Errorf("test case not found")
}

func (sm *Client) ListSecrets(*awssm.ListSecretsInput) (*awssm.ListSecretsOutput, error) {
	return nil, nil
}

func (sm *Client) cacheKeyForInput(in *awssm.GetSecretValueInput) string {
	var secretID, versionID string
	if in.SecretId != nil {
		secretID = *in.SecretId
	}
	if in.VersionId != nil {
		versionID = *in.VersionId
	}
	return fmt.Sprintf("%s#%s", secretID, versionID)
}

func (sm *Client) WithValue(in *awssm.GetSecretValueInput, val *awssm.GetSecretValueOutput, err error) {
	sm.valFn[sm.cacheKeyForInput(in)] = func(paramIn *awssm.GetSecretValueInput) (*awssm.GetSecretValueOutput, error) {
		if !cmp.Equal(paramIn, in) {
			return nil, fmt.Errorf("unexpected test argument")
		}
		return val, err
	}
}

// func makeValidSecretStoreWithVersion(v esv1beta1.VaultKVStoreVersion) *esv1beta1.SecretStore {
// 	return &esv1beta1.SecretStore{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "vault-store",
// 			Namespace: "default",
// 		},
// 		Spec: esv1beta1.SecretStoreSpec{
// 			Provider: &esv1beta1.SecretStoreProvider{
// 				Vault: &esv1beta1.VaultProvider{
// 					Server:  "vault.example.com",
// 					Path:    &secretStorePath,
// 					Version: v,
// 					Auth: esv1beta1.VaultAuth{
// 						Kubernetes: &esv1beta1.VaultKubernetesAuth{
// 							Path: "kubernetes",
// 							Role: "kubernetes-auth-role",
// 							ServiceAccountRef: &esmeta.ServiceAccountSelector{
// 								Name: "example-sa",
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}
// }
