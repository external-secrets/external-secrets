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

package generator

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	genpb "github.com/external-secrets/external-secrets/proto/generator"
)

// Server wraps v1 providers and generators and exposes them as v2 gRPC services.
// This allows existing v1 provider and generator implementations to be used in the v2 architecture.
type Server struct {
	genpb.UnimplementedGeneratorProviderServer
	kubeClient client.Client
	scheme     *runtime.Scheme

	// we support multiple v1 generators, so we need to map the v2 generator
	// with apiVersion+kind to the corresponding v1 generator
	generatorMapping GeneratorMapping
}

// GeneratorMapping maps Kubernetes resources to their generator implementations.
type GeneratorMapping map[schema.GroupVersionKind]genv1alpha1.Generator

// NewServer creates a new generator adapter server.
func NewServer(kubeClient client.Client, scheme *runtime.Scheme, generatorMapping GeneratorMapping) *Server {
	return &Server{
		kubeClient:       kubeClient,
		scheme:           scheme,
		generatorMapping: generatorMapping,
	}
}

// Generate creates a new secret or set of secrets using a v1 generator.
func (s *Server) Generate(ctx context.Context, req *genpb.GenerateRequest) (*genpb.GenerateResponse, error) {
	if req == nil || req.GeneratorRef == nil {
		return nil, fmt.Errorf("request or generator ref is nil")
	}

	// Resolve the generator implementation and resource
	generator, jsonObj, err := s.getGenerator(ctx, req.GeneratorRef, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get generator: %w", err)
	}

	// Call the v1 generator's Generate method
	secretMap, state, err := generator.Generate(ctx, jsonObj, s.kubeClient, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to generate secrets: %w", err)
	}

	// Convert state to bytes
	var stateBytes []byte
	if state != nil {
		stateBytes = state.Raw
	}

	return &genpb.GenerateResponse{
		Secrets: secretMap,
		State:   stateBytes,
	}, nil
}

// Cleanup deletes any resources created during the Generate phase.
func (s *Server) Cleanup(ctx context.Context, req *genpb.CleanupRequest) (*genpb.CleanupResponse, error) {
	if req == nil || req.GeneratorRef == nil {
		return nil, fmt.Errorf("request or generator ref is nil")
	}

	// Resolve the generator implementation and resource
	generator, jsonObj, err := s.getGenerator(ctx, req.GeneratorRef, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get generator: %w", err)
	}

	// Unmarshal the state
	var state genv1alpha1.GeneratorProviderState
	if len(req.State) > 0 {
		state = &apiextensions.JSON{Raw: req.State}
	}

	// Call the v1 generator's Cleanup method
	err = generator.Cleanup(ctx, jsonObj, state, s.kubeClient, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to cleanup generator resources: %w", err)
	}

	return &genpb.CleanupResponse{}, nil
}

// getGenerator retrieves a generator implementation and the generator resource.
// This is similar to runtime/esutils/resolvers/generator.go getGenerator function.
func (s *Server) getGenerator(ctx context.Context, generatorRef *genpb.GeneratorRef, namespace string) (genv1alpha1.Generator, *apiextensions.JSON, error) {
	if generatorRef == nil {
		return nil, nil, fmt.Errorf("generator reference is nil")
	}

	// Parse the API version
	splitted := strings.Split(generatorRef.ApiVersion, "/")
	if len(splitted) != 2 {
		return nil, nil, fmt.Errorf("invalid api version: %s", generatorRef.ApiVersion)
	}
	group := splitted[0]
	version := splitted[1]

	// Construct the GVK
	gvk := schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    generatorRef.Kind,
	}

	// Fail if the GVK does not use the generator group
	if gvk.Group != genv1alpha1.Group {
		return nil, nil, fmt.Errorf("generatorRef may only reference the generators group, but got %s", gvk.Group)
	}

	// Lookup the generator implementation from the mapping
	generator, ok := s.generatorMapping[gvk]
	if !ok {
		return nil, nil, fmt.Errorf("generator mapping not found for %q", gvk)
	}

	// Get a client Object from the GVK
	t, exists := s.scheme.AllKnownTypes()[gvk]
	if !exists {
		return nil, nil, fmt.Errorf("generatorRef references unknown GVK %s", gvk)
	}
	obj := reflect.New(t).Interface().(client.Object)

	// Handle ClusterGenerator specially
	if gvk.Kind == genv1alpha1.ClusterGeneratorKind {
		clusterGenerator := obj.(*genv1alpha1.ClusterGenerator)

		// Get the cluster generator resource from the API
		err := s.kubeClient.Get(ctx, client.ObjectKey{Name: generatorRef.Name}, clusterGenerator)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get ClusterGenerator: %w", err)
		}

		// Convert the cluster generator to a virtual namespaced generator object
		obj, err = s.clusterGeneratorToVirtual(clusterGenerator)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid ClusterGenerator: %w", err)
		}
	} else {
		// Get the generator resource from the API
		nsName := types.NamespacedName{
			Name:      generatorRef.Name,
			Namespace: namespace,
		}
		// Use the namespace from the ref if provided (for flexibility)
		if generatorRef.Namespace != "" {
			nsName.Namespace = generatorRef.Namespace
		}

		err := s.kubeClient.Get(ctx, nsName, obj)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get generator resource: %w", err)
		}
	}

	// Convert the generator to unstructured object
	u := &unstructured.Unstructured{}
	var unstructuredObj map[string]interface{}
	var err error
	unstructuredObj, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert to unstructured: %w", err)
	}
	u.Object = unstructuredObj

	// Convert the unstructured object to JSON
	jsonObj, err := u.MarshalJSON()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return generator, &apiextensions.JSON{Raw: jsonObj}, nil
}

// clusterGeneratorToVirtual converts a ClusterGenerator to a virtual namespaced generator.
// This is adapted from runtime/esutils/resolvers/generator.go.
func (s *Server) clusterGeneratorToVirtual(gen *genv1alpha1.ClusterGenerator) (client.Object, error) {
	switch gen.Spec.Kind {
	case genv1alpha1.GeneratorKindFake:
		if gen.Spec.Generator.FakeSpec == nil {
			return nil, fmt.Errorf("when kind is %s, FakeSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.Fake{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.FakeKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: gen.Name,
			},
			Spec: *gen.Spec.Generator.FakeSpec,
		}, nil
	// Add more generator kinds here as needed
	default:
		return nil, fmt.Errorf("unsupported generator kind: %s", gen.Spec.Kind)
	}
}
