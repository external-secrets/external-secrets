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

package uuid

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

type Generator struct{}

type generateFunc func() (string, error)

func (g *Generator) Generate(_ context.Context, jsonSpec *apiextensions.JSON, _ client.Client, _ string) (map[string][]byte, error) {
	return g.generate(
		jsonSpec,
		generateUUID,
	)
}

func (g *Generator) generate(_ *apiextensions.JSON, uuidGen generateFunc) (map[string][]byte, error) {
	uuid, err := uuidGen()
	if err != nil {
		return nil, fmt.Errorf("unable to generate UUID: %w", err)
	}
	return map[string][]byte{
		"uuid": []byte(uuid),
	}, nil
}

func generateUUID() (string, error) {
	uuid := uuid.New()
	return uuid.String(), nil
}

func init() {
	genv1alpha1.Register(genv1alpha1.UUIDKind, &Generator{})
}
