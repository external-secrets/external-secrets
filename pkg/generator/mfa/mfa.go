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

package mfa

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

type Generator struct{}

const (
	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, c client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	var opts []GeneratorOptionsFunc
	if res.Spec.Length > 0 {
		opts = append(opts, WithLength(res.Spec.Length))
	}
	if res.Spec.TimePeriod > 0 {
		opts = append(opts, WithTimePeriod(int64(res.Spec.TimePeriod)))
	}
	if res.Spec.When != nil {
		opts = append(opts, WithWhen(res.Spec.When.Time))
	}

	secret := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: res.Spec.Secret.Name}, secret); err != nil {
		return nil, nil, fmt.Errorf("failed to find secret for token key: %w", err)
	}

	seed, ok := secret.Data[res.Spec.Secret.Key]
	if !ok {
		return nil, nil, fmt.Errorf("secret key %s does not exist in secret data map", res.Spec.Secret.Key)
	}

	opts = append(opts, WithToken(string(seed)))

	token, timeLeft, err := generateCode(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate code for token key: %w", err)
	}

	return map[string][]byte{
		"token":    []byte(token),
		"timeLeft": []byte(timeLeft),
	}, nil, nil
}

func (g *Generator) Cleanup(_ context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func parseSpec(data []byte) (*genv1alpha1.MFA, error) {
	var spec genv1alpha1.MFA
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.MFAKind, &Generator{})
}
