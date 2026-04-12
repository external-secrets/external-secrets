/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	k8sv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

func TestGetSpecMapperUsesProviderRefNamespaceBeforeSourceNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(k8sv2alpha1.AddToScheme(scheme))

	referenced := &k8sv2alpha1.Kubernetes{}
	referenced.Name = "backend"
	referenced.Namespace = "provider-config-ns"
	referenced.Spec.RemoteNamespace = "remote-from-provider-ref"

	fallback := &k8sv2alpha1.Kubernetes{}
	fallback.Name = "backend"
	fallback.Namespace = "tenant-a"
	fallback.Spec.RemoteNamespace = "remote-from-source-namespace"

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(referenced, fallback).
		Build()

	mapper := GetSpecMapper(kubeClient)

	spec, err := mapper(&pb.ProviderReference{
		Name:      "backend",
		Namespace: "provider-config-ns",
	}, "tenant-a")
	if err != nil {
		t.Fatalf("mapper() error = %v", err)
	}

	if spec.Provider == nil || spec.Provider.Kubernetes == nil {
		t.Fatalf("expected kubernetes provider spec, got %#v", spec.Provider)
	}
	if spec.Provider.Kubernetes.RemoteNamespace != "remote-from-provider-ref" {
		t.Fatalf("expected provider-ref namespace object, got %#v", spec.Provider.Kubernetes)
	}
}

func TestGetSpecMapperFallsBackToSourceNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(k8sv2alpha1.AddToScheme(scheme))

	fallback := &k8sv2alpha1.Kubernetes{}
	fallback.Name = "backend"
	fallback.Namespace = "tenant-a"
	fallback.Spec.RemoteNamespace = "remote-from-source-namespace"

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(fallback).
		Build()

	mapper := GetSpecMapper(kubeClient)

	spec, err := mapper(&pb.ProviderReference{
		Name: "backend",
	}, "tenant-a")
	if err != nil {
		t.Fatalf("mapper() error = %v", err)
	}

	if spec.Provider == nil || spec.Provider.Kubernetes == nil {
		t.Fatalf("expected kubernetes provider spec, got %#v", spec.Provider)
	}
	if spec.Provider.Kubernetes.RemoteNamespace != "remote-from-source-namespace" {
		t.Fatalf("expected source namespace object, got %#v", spec.Provider.Kubernetes)
	}
}
