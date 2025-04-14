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

package resolvers

import (
	"context"
	"fmt"
	"reflect"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

// these errors are explicitly defined so we can detect them with `errors.Is()`.
var (
	// ErrUnableToGetGenerator is returned when a generator reference cannot be resolved.
	ErrUnableToGetGenerator = fmt.Errorf("unable to get generator")
)

// GeneratorRef resolves a generator reference to a generator implementation.
func GeneratorRef(ctx context.Context, cl client.Client, scheme *runtime.Scheme, namespace string, generatorRef *esv1beta1.GeneratorRef) (genv1alpha1.Generator, *apiextensions.JSON, error) {
	generator, jsonObj, err := getGenerator(ctx, cl, scheme, namespace, generatorRef)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrUnableToGetGenerator, err)
	}
	return generator, jsonObj, nil
}

func getGenerator(ctx context.Context, cl client.Client, scheme *runtime.Scheme, namespace string, generatorRef *esv1beta1.GeneratorRef) (genv1alpha1.Generator, *apiextensions.JSON, error) {
	// get a GVK from the generatorRef
	gv, err := schema.ParseGroupVersion(generatorRef.APIVersion)
	if err != nil {
		return nil, nil, reconcile.TerminalError(fmt.Errorf("generatorRef has invalid APIVersion: %w", err))
	}
	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    generatorRef.Kind,
	}

	// fail if the GVK does not use the generator group
	if gvk.Group != genv1alpha1.Group {
		return nil, nil, reconcile.TerminalError(fmt.Errorf("generatorRef may only reference the generators group, but got %s", gvk.Group))
	}

	// get a client Object from the GVK
	t, exists := scheme.AllKnownTypes()[gvk]
	if !exists {
		return nil, nil, reconcile.TerminalError(fmt.Errorf("generatorRef references unknown GVK %s", gvk))
	}
	obj := reflect.New(t).Interface().(client.Object)

	// this interface provides the Generate() method used by the controller
	// NOTE: all instances of a generator kind use the same instance of this interface
	var generator genv1alpha1.Generator

	// ClusterGenerator is a special case because it's a cluster-scoped resource
	// to use it, we create a "virtual" namespaced generator for the current namespace, as if one existed in the API
	if gvk.Kind == genv1alpha1.ClusterGeneratorKind {
		clusterGenerator := obj.(*genv1alpha1.ClusterGenerator)

		// get the cluster generator resource from the API
		// NOTE: it's important that we use the structured client so we use the cache
		err = cl.Get(ctx, client.ObjectKey{Name: generatorRef.Name}, clusterGenerator)
		if err != nil {
			return nil, nil, err
		}

		// convert the cluster generator to a virtual namespaced generator object
		obj, err = clusterGeneratorToVirtual(clusterGenerator)
		if err != nil {
			return nil, nil, reconcile.TerminalError(fmt.Errorf("invalid ClusterGenerator: %w", err))
		}

		// get the generator interface
		var ok bool
		generator, ok = genv1alpha1.GetGeneratorByName(string(clusterGenerator.Spec.Kind))
		if !ok {
			return nil, nil, reconcile.TerminalError(fmt.Errorf("ClusterGenerator has unknown kind %s", clusterGenerator.Spec.Kind))
		}
	} else {
		// get the generator resource from the API
		// NOTE: it's important that we use the structured client so we use the cache
		err = cl.Get(ctx, types.NamespacedName{
			Name:      generatorRef.Name,
			Namespace: namespace,
		}, obj)
		if err != nil {
			return nil, nil, err
		}

		// get the generator interface
		var ok bool
		generator, ok = genv1alpha1.GetGeneratorByName(gvk.Kind)
		if !ok {
			return nil, nil, reconcile.TerminalError(fmt.Errorf("generatorRef has unknown kind %s", gvk.Kind))
		}
	}

	// convert the generator to unstructured object
	u := &unstructured.Unstructured{}
	u.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, nil, err
	}

	// convert the unstructured object to JSON
	// NOTE: we do this for backwards compatibility with how this API works, not because it's a good idea
	//       we should refactor the generator API to use the normal typed objects
	jsonObj, err := u.MarshalJSON()
	if err != nil {
		return nil, nil, err
	}

	return generator, &apiextensions.JSON{Raw: jsonObj}, nil
}

// clusterGeneratorToVirtual converts a ClusterGenerator to a "virtual" namespaced generator that doesn't actually exist in the API.
func clusterGeneratorToVirtual(gen *genv1alpha1.ClusterGenerator) (client.Object, error) {
	switch gen.Spec.Kind {
	case genv1alpha1.GeneratorKindACRAccessToken:
		if gen.Spec.Generator.ACRAccessTokenSpec == nil {
			return nil, fmt.Errorf("when kind is %s, ACRAccessTokenSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.ACRAccessToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.ACRAccessTokenKind,
			},
			Spec: *gen.Spec.Generator.ACRAccessTokenSpec,
		}, nil
	case genv1alpha1.GeneratorKindECRAuthorizationToken:
		if gen.Spec.Generator.ECRAuthorizationTokenSpec == nil {
			return nil, fmt.Errorf("when kind is %s, ECRAuthorizationTokenSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.ECRAuthorizationToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.ECRAuthorizationTokenKind,
			},
			Spec: *gen.Spec.Generator.ECRAuthorizationTokenSpec,
		}, nil
	case genv1alpha1.GeneratorKindFake:
		if gen.Spec.Generator.FakeSpec == nil {
			return nil, fmt.Errorf("when kind is %s, FakeSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.Fake{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.FakeKind,
			},
			Spec: *gen.Spec.Generator.FakeSpec,
		}, nil
	case genv1alpha1.GeneratorKindGCRAccessToken:
		if gen.Spec.Generator.GCRAccessTokenSpec == nil {
			return nil, fmt.Errorf("when kind is %s, GCRAccessTokenSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.GCRAccessToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.GCRAccessTokenKind,
			},
			Spec: *gen.Spec.Generator.GCRAccessTokenSpec,
		}, nil
	case genv1alpha1.GeneratorKindGithubAccessToken:
		if gen.Spec.Generator.GithubAccessTokenSpec == nil {
			return nil, fmt.Errorf("when kind is %s, GithubAccessTokenSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.GithubAccessToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.GithubAccessTokenKind,
			},
			Spec: *gen.Spec.Generator.GithubAccessTokenSpec,
		}, nil
	case genv1alpha1.GeneratorKindQuayAccessToken:
		if gen.Spec.Generator.QuayAccessTokenSpec == nil {
			return nil, fmt.Errorf("when kind is %s, QuayAccessTokenSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.QuayAccessToken{
			Spec: *gen.Spec.Generator.QuayAccessTokenSpec,
		}, nil
	case genv1alpha1.GeneratorKindPassword:
		if gen.Spec.Generator.PasswordSpec == nil {
			return nil, fmt.Errorf("when kind is %s, PasswordSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.Password{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.PasswordKind,
			},
			Spec: *gen.Spec.Generator.PasswordSpec,
		}, nil
	case genv1alpha1.GeneratorKindSTSSessionToken:
		if gen.Spec.Generator.STSSessionTokenSpec == nil {
			return nil, fmt.Errorf("when kind is %s, STSSessionTokenSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.STSSessionToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.STSSessionTokenKind,
			},
			Spec: *gen.Spec.Generator.STSSessionTokenSpec,
		}, nil
	case genv1alpha1.GeneratorKindUUID:
		if gen.Spec.Generator.UUIDSpec == nil {
			return nil, fmt.Errorf("when kind is %s, UUIDSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.UUID{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.UUIDKind,
			},
			Spec: *gen.Spec.Generator.UUIDSpec,
		}, nil
	case genv1alpha1.GeneratorKindVaultDynamicSecret:
		if gen.Spec.Generator.VaultDynamicSecretSpec == nil {
			return nil, fmt.Errorf("when kind is %s, VaultDynamicSecretSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.VaultDynamicSecret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.VaultDynamicSecretKind,
			},
			Spec: *gen.Spec.Generator.VaultDynamicSecretSpec,
		}, nil
	case genv1alpha1.GeneratorKindWebhook:
		if gen.Spec.Generator.WebhookSpec == nil {
			return nil, fmt.Errorf("when kind is %s, WebhookSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.Webhook{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.WebhookKind,
			},
			Spec: *gen.Spec.Generator.WebhookSpec,
		}, nil
	case genv1alpha1.GeneratorKindGrafana:
		if gen.Spec.Generator.GrafanaSpec == nil {
			return nil, fmt.Errorf("when kind is %s, GrafanaSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.Grafana{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.GrafanaKind,
			},
			Spec: *gen.Spec.Generator.GrafanaSpec,
		}, nil
	default:
		return nil, fmt.Errorf("unknown kind %s", gen.Spec.Kind)
	}
}
