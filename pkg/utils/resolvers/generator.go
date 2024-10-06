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

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

// GeneratorRef resolves a generator reference to a generator implementation.
func GeneratorRef(ctx context.Context, restConfig *rest.Config, namespace string, generatorRef *esv1beta1.GeneratorRef) (genv1alpha1.Generator, *apiextensions.JSON, error) {
	obj, err := getGeneratorDefinition(ctx, restConfig, namespace, generatorRef)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get generator definition: %w", err)
	}
	generator, err := genv1alpha1.GetGenerator(obj)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get generator: %w", err)
	}
	return generator, obj, nil
}

func getGeneratorDefinition(ctx context.Context, restConfig *rest.Config, namespace string, generatorRef *esv1beta1.GeneratorRef) (*apiextensions.JSON, error) {
	// client-go dynamic client needs a GVR to fetch the resource
	// But we only have the GVK in our generatorRef.
	//
	// TODO: there is no need to discover the GroupVersionResource
	//       this should be cached.
	c := discovery.NewDiscoveryClientForConfigOrDie(restConfig)
	groupResources, err := restmapper.GetAPIGroupResources(c)
	if err != nil {
		return nil, err
	}

	gv, err := schema.ParseGroupVersion(generatorRef.APIVersion)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	mapping, err := mapper.RESTMapping(schema.GroupKind{
		Group: gv.Group,
		Kind:  generatorRef.Kind,
	})
	if err != nil {
		return nil, err
	}
	d, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	res, err := d.Resource(mapping.Resource).
		Namespace(namespace).
		Get(ctx, generatorRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	jsonRes, err := res.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return &apiextensions.JSON{Raw: jsonRes}, nil
}
