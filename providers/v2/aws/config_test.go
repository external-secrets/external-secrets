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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	awsv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/aws/v2alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

func TestGetSpecMapperMapsParameterStore(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}
	if err := awsv2alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&awsv2alpha1.ParameterStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ps-config",
			Namespace: "provider-ns",
		},
		Spec: awsv2alpha1.ParameterStoreSpec{
			Region: "eu-central-1",
			Role:   "arn:aws:iam::123456789012:role/eso-ssm",
			Prefix: "/team-a/",
			ExternalID: "ext-id",
		},
	}).Build()

	spec, err := GetSpecMapper(kubeClient)(&pb.ProviderReference{
		ApiVersion: awsv2alpha1.GroupVersion.String(),
		Kind:       awsv2alpha1.ParameterStoreKind,
		Name:       "ps-config",
		Namespace:  "provider-ns",
	}, "workload-ns")
	if err != nil {
		t.Fatalf("GetSpecMapper() error = %v", err)
	}
	if spec.Provider == nil || spec.Provider.AWS == nil {
		t.Fatal("expected AWS provider spec to be returned")
	}
	if spec.Provider.AWS.Service != esv1.AWSServiceParameterStore {
		t.Fatalf("expected service %q, got %q", esv1.AWSServiceParameterStore, spec.Provider.AWS.Service)
	}
	if spec.Provider.AWS.Region != "eu-central-1" {
		t.Fatalf("expected region to be preserved, got %q", spec.Provider.AWS.Region)
	}
	if spec.Provider.AWS.Role != "arn:aws:iam::123456789012:role/eso-ssm" {
		t.Fatalf("expected role to be preserved, got %q", spec.Provider.AWS.Role)
	}
	if spec.Provider.AWS.Prefix != "/team-a/" {
		t.Fatalf("expected prefix to be preserved, got %q", spec.Provider.AWS.Prefix)
	}
	if spec.Provider.AWS.ExternalID != "ext-id" {
		t.Fatalf("expected external ID to be preserved, got %q", spec.Provider.AWS.ExternalID)
	}
}

func TestGetSpecMapperUsesSourceNamespaceForParameterStore(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}
	if err := awsv2alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&awsv2alpha1.ParameterStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ps-source-ns",
			Namespace: "workload-ns",
		},
		Spec: awsv2alpha1.ParameterStoreSpec{
			Region: "eu-west-1",
		},
	}).Build()

	spec, err := GetSpecMapper(kubeClient)(&pb.ProviderReference{
		ApiVersion: awsv2alpha1.GroupVersion.String(),
		Kind:       awsv2alpha1.ParameterStoreKind,
		Name:       "ps-source-ns",
	}, "workload-ns")
	if err != nil {
		t.Fatalf("GetSpecMapper() error = %v", err)
	}
	if spec.Provider == nil || spec.Provider.AWS == nil {
		t.Fatal("expected AWS provider spec to be returned")
	}
	if spec.Provider.AWS.Service != esv1.AWSServiceParameterStore {
		t.Fatalf("expected service %q, got %q", esv1.AWSServiceParameterStore, spec.Provider.AWS.Service)
	}
}
