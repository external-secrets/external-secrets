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
	"context"
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

type Generator struct{}

const (
	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errGetToken  = "unable to get authorization token: %w"
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, error) {
	if jsonSpec == nil {
		return nil, fmt.Errorf(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, fmt.Errorf(errParseSpec, err)
	}
	out := make(map[string][]byte)
	for k, v := range res.Spec.Data {
		out[k] = []byte(v)
	}
	return out, nil
}

func parseSpec(data []byte) (*genv1alpha1.Fake, error) {
	var spec genv1alpha1.Fake
	err := json.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.FakeKind, &Generator{})
}
