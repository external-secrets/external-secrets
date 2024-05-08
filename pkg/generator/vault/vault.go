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

package vaultdynamic

import (
	"context"
	"encoding/json"
	"fmt"

	vault "github.com/hashicorp/vault/api"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	provider "github.com/external-secrets/external-secrets/pkg/provider/vault"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

type Generator struct{}

const (
	errNoSpec      = "no config spec provided"
	errParseSpec   = "unable to parse spec: %w"
	errVaultClient = "unable to setup Vault client: %w"
	errGetSecret   = "unable to get dynamic secret: %w"
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, error) {
	c := &provider.Provider{NewVaultClient: provider.NewVaultClient}

	// controller-runtime/client does not support TokenRequest or other subresource APIs
	// so we need to construct our own client and use it to fetch tokens
	// (for Kubernetes service account token auth)
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	return g.generate(ctx, c, jsonSpec, kube, clientset.CoreV1(), namespace)
}

func (g *Generator) generate(ctx context.Context, c *provider.Provider, jsonSpec *apiextensions.JSON, kube client.Client, corev1 typedcorev1.CoreV1Interface, namespace string) (map[string][]byte, error) {
	if jsonSpec == nil {
		return nil, fmt.Errorf(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, fmt.Errorf(errParseSpec, err)
	}
	if res == nil || res.Spec.Provider == nil {
		return nil, fmt.Errorf("no Vault provider config in spec")
	}
	cl, err := c.NewGeneratorClient(ctx, kube, corev1, res.Spec.Provider, namespace)
	if err != nil {
		return nil, fmt.Errorf(errVaultClient, err)
	}

	var result *vault.Secret
	if res.Spec.Method == "" || res.Spec.Method == "GET" {
		result, err = cl.Logical().ReadWithDataWithContext(ctx, res.Spec.Path, nil)
	} else if res.Spec.Method == "LIST" {
		result, err = cl.Logical().ListWithContext(ctx, res.Spec.Path)
	} else if res.Spec.Method == "DELETE" {
		result, err = cl.Logical().DeleteWithContext(ctx, res.Spec.Path)
	} else {
		params := make(map[string]any)
		if res.Spec.Parameters != nil {
			err = json.Unmarshal(res.Spec.Parameters.Raw, &params)
			if err != nil {
				return nil, err
			}
		}
		result, err = cl.Logical().WriteWithContext(ctx, res.Spec.Path, params)
	}
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf(errGetSecret, fmt.Errorf("empty response from Vault"))
	}

	data := make(map[string]any)
	response := make(map[string][]byte)
	if res.Spec.ResultType == genv1alpha1.VaultDynamicSecretResultTypeAuth {
		authJSON, err := json.Marshal(result.Auth)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(authJSON, &data)
		if err != nil {
			return nil, err
		}
	} else {
		data = result.Data
	}

	for k := range data {
		response[k], err = utils.GetByteValueFromMap(data, k)
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}

func parseSpec(data []byte) (*genv1alpha1.VaultDynamicSecret, error) {
	var spec genv1alpha1.VaultDynamicSecret
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.VaultDynamicSecretKind, &Generator{})
}
