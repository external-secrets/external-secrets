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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	gcpsmv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/gcp/v2alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

func TestGetSpecMapperMapsSecretManagerSpec(t *testing.T) {
	t.Parallel()

	serviceAccountNamespace := "identity-ns"

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}
	if err := gcpsmv2alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&gcpsmv2alpha1.SecretManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gcp-config",
			Namespace: "provider-ns",
		},
		Spec: gcpsmv2alpha1.SecretManagerSpec{
			ProjectID: "project-a",
			Location:  "europe-west3",
			Auth: esv1.GCPSMAuth{
				SecretRef: &esv1.GCPSMAuthSecretRef{
					SecretAccessKey: esmeta.SecretKeySelector{
						Name: "gcp-creds",
						Key:  "service-account.json",
					},
				},
				WorkloadIdentity: &esv1.GCPWorkloadIdentity{
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name:      "eso-gcp",
						Namespace: &serviceAccountNamespace,
						Audiences: []string{"https://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/provider"},
					},
					ClusterLocation:  "europe-west3",
					ClusterName:      "cluster-a",
					ClusterProjectID: "cluster-project",
				},
			},
		},
	}).Build()

	spec, err := GetSpecMapper(kubeClient)(&pb.ProviderReference{
		ApiVersion: gcpsmv2alpha1.GroupVersion.String(),
		Kind:       gcpsmv2alpha1.SecretManagerKind,
		Name:       "gcp-config",
		Namespace:  "provider-ns",
	}, "workload-ns")
	if err != nil {
		t.Fatalf("GetSpecMapper() error = %v", err)
	}
	if spec.Provider == nil || spec.Provider.GCPSM == nil {
		t.Fatal("expected GCP Secret Manager provider spec to be returned")
	}
	if spec.Provider.GCPSM.ProjectID != "project-a" {
		t.Fatalf("expected project ID to be preserved, got %q", spec.Provider.GCPSM.ProjectID)
	}
	if spec.Provider.GCPSM.Location != "europe-west3" {
		t.Fatalf("expected location to be preserved, got %q", spec.Provider.GCPSM.Location)
	}
	if spec.Provider.GCPSM.Auth.SecretRef == nil {
		t.Fatal("expected secretRef auth to be preserved")
	}
	if spec.Provider.GCPSM.Auth.SecretRef.SecretAccessKey.Name != "gcp-creds" {
		t.Fatalf("expected secretRef name to be preserved, got %q", spec.Provider.GCPSM.Auth.SecretRef.SecretAccessKey.Name)
	}
	if spec.Provider.GCPSM.Auth.SecretRef.SecretAccessKey.Key != "service-account.json" {
		t.Fatalf("expected secretRef key to be preserved, got %q", spec.Provider.GCPSM.Auth.SecretRef.SecretAccessKey.Key)
	}
	if spec.Provider.GCPSM.Auth.WorkloadIdentity == nil {
		t.Fatal("expected workload identity auth to be preserved")
	}
	if spec.Provider.GCPSM.Auth.WorkloadIdentity.ServiceAccountRef.Name != "eso-gcp" {
		t.Fatalf("expected workload identity service account name to be preserved, got %q", spec.Provider.GCPSM.Auth.WorkloadIdentity.ServiceAccountRef.Name)
	}
	if spec.Provider.GCPSM.Auth.WorkloadIdentity.ServiceAccountRef.Namespace == nil || *spec.Provider.GCPSM.Auth.WorkloadIdentity.ServiceAccountRef.Namespace != serviceAccountNamespace {
		t.Fatalf("expected workload identity service account namespace %q, got %v", serviceAccountNamespace, spec.Provider.GCPSM.Auth.WorkloadIdentity.ServiceAccountRef.Namespace)
	}
}

func TestGetSpecMapperUsesSourceNamespaceForSecretManager(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}
	if err := gcpsmv2alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&gcpsmv2alpha1.SecretManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gcp-source-ns",
			Namespace: "workload-ns",
		},
		Spec: gcpsmv2alpha1.SecretManagerSpec{
			ProjectID: "project-b",
			Location:  "us-central1",
		},
	}).Build()

	spec, err := GetSpecMapper(kubeClient)(&pb.ProviderReference{
		ApiVersion: gcpsmv2alpha1.GroupVersion.String(),
		Kind:       gcpsmv2alpha1.SecretManagerKind,
		Name:       "gcp-source-ns",
	}, "workload-ns")
	if err != nil {
		t.Fatalf("GetSpecMapper() error = %v", err)
	}
	if spec.Provider == nil || spec.Provider.GCPSM == nil {
		t.Fatal("expected GCP Secret Manager provider spec to be returned")
	}
	if spec.Provider.GCPSM.ProjectID != "project-b" {
		t.Fatalf("expected project ID to be read from source namespace, got %q", spec.Provider.GCPSM.ProjectID)
	}
	if spec.Provider.GCPSM.Location != "us-central1" {
		t.Fatalf("expected location to be read from source namespace, got %q", spec.Provider.GCPSM.Location)
	}
}
