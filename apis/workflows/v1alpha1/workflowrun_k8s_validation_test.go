// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestValidateKubernetesResourceValidation(t *testing.T) {
	// Create a test scheme
	scheme := runtime.NewScheme()
	_ = AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = esv1.AddToScheme(scheme)
	_ = genv1alpha1.AddToScheme(scheme)
	_ = scanv1alpha1.AddToScheme(scheme)

	// Create the test namespace
	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Create a test SecretStore
	testSecretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-store",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AWS: &esv1.AWSProvider{
					Region: "us-west-2",
				},
			},
		},
	}

	// Create a test SecretStore
	testSecondSecretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-second-store",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AWS: &esv1.AWSProvider{
					Region: "us-west-2",
				},
			},
		},
	}

	// Create a test ClusterSecretStore
	testClusterSecretStore := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster-store",
			Labels: map[string]string{
				"env": "test",
			},
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AWS: &esv1.AWSProvider{
					Region: "us-east-1",
				},
			},
		},
	}

	// Create a test Generator
	testGenerator := &genv1alpha1.Password{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-password-gen",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"env": "test",
			},
		},
	}

	testSecondGenerator := &genv1alpha1.Fake{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-password-gen-2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"env": "test",
			},
		},
	}

	// Create a test Finding
	testFinding := &scanv1alpha1.Finding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-finding",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"env": "test",
			},
		},
		Spec: scanv1alpha1.FindingSpec{
			Hash:           "abc123",
			RunTemplateRef: &scanv1alpha1.RunTemplateReference{Name: "sample-template"},
		},
		Status: scanv1alpha1.FindingStatus{
			Locations: []scanv1alpha1.SecretInStoreRef{}, // empty list acceptable
		},
	}

	testSecondFinding := &scanv1alpha1.Finding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-finding-2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"env": "test",
			},
		},
		Spec: scanv1alpha1.FindingSpec{
			Hash:           "def456",
			RunTemplateRef: &scanv1alpha1.RunTemplateReference{Name: "another-template"},
		},
		Status: scanv1alpha1.FindingStatus{
			Locations: []scanv1alpha1.SecretInStoreRef{},
		},
	}

	// Create a test template with Kubernetes resource parameters
	template := &WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-resource-template",
			Namespace: "test-namespace",
		},
		Spec: WorkflowTemplateSpec{
			Version: "v1",
			Name:    "K8s Resource Test",
			ParameterGroups: []ParameterGroup{
				{
					Name: "Resources",
					Parameters: []Parameter{
						{
							Name:     "targetNamespace",
							Type:     ParameterTypeNamespace,
							Required: true,
						},
						{
							Name:     "secretStore",
							Type:     ParameterTypeSecretStore,
							Required: true,
							ResourceConstraints: &ResourceConstraints{
								Namespace: "test-namespace",
								LabelSelector: map[string]string{
									"app": "test",
								},
							},
						},
						{
							Name:     "secretStoreArray",
							Type:     ParameterTypeSecretStoreArray,
							Required: false,
						},
						{
							Name:     "clusterSecretStore",
							Type:     ParameterTypeClusterSecretStore,
							Required: false,
							ResourceConstraints: &ResourceConstraints{
								LabelSelector: map[string]string{
									"env": "test",
								},
							},
						},
						{
							Name:     "secretStoreNoCrossNS",
							Type:     ParameterTypeSecretStore,
							Required: false,
							ResourceConstraints: &ResourceConstraints{
								Namespace:           "test-namespace",
								AllowCrossNamespace: false,
							},
						},
						{
							Name:     "generator",
							Type:     ParameterType("generator[Password]"),
							Required: false,
						},
						{
							Name:     "generatorArray",
							Type:     ParameterType("array[generator[any]]"),
							Required: false,
						},
						{
							Name:     "secretlocation",
							Type:     ParameterTypeSecretLocation,
							Required: false,
						},
						{
							Name:     "secretlocationArray",
							Type:     ParameterTypeSecretLocationArray,
							Required: false,
						},
						{
							Name:     "finding",
							Type:     ParameterTypeFinding,
							Required: false,
						},
						{
							Name:     "findingArray",
							Type:     ParameterTypeFindingArray,
							Required: false,
						},
						{
							Name:     "objectFinding",
							Type:     ParameterType("object[customName]finding"),
							Required: false,
						},
						{
							Name:     "objectFindingArray",
							Type:     ParameterType("object[customName]array[finding]"),
							Required: false,
						},
					},
				},
			},
			Jobs: map[string]Job{
				"test": {
					Standard: &StandardJob{
						Steps: []Step{
							{
								Name: "test",
								Debug: &DebugStep{
									Message: "Test",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create a fake client with test objects
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			testNamespace,
			testSecretStore,
			testSecondSecretStore,
			testClusterSecretStore,
			testGenerator,
			testSecondGenerator,
			testFinding,
			testSecondFinding,
			template).
		Build()

	// Set the validation client
	SetValidationClient(client)

	// Test cases
	tests := []struct {
		name        string
		workflowRun *WorkflowRun
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid secretStoreArray",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"secretStoreArray": [{"name": "test-store"}, {"name": "test-second-store"}]
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid secretStoreArray with one element",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"secretStoreArray": [{"name": "test-second-store"}]
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid StoreArray",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"secretStoreArray": [{"name": "test-second-store"}, {"name": "unexisting-store"}]
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource unexisting-store of type secretstore not found in namespace test-namespace",
		},

		{
			name: "valid k8s resources",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":    "test-namespace",
							"secretStore":        {"name": "test-store"},
							"clusterSecretStore": {"name": "test-cluster-store"}
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "non-existent namespace",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-ns-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace": "non-existent-namespace",
							"secretStore":     {"name": "test-store"}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-namespace of type namespace not found",
		},
		{
			name: "non-existent secret store",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-store-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace": "test-namespace",
							"secretStore":     {"name": "non-existent-store"}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-store of type secretstore not found in namespace test-namespace",
		},
		{
			name: "secret store label selector mismatch",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "label-mismatch-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace": "test-namespace",
							"secretStore":     {"name": "test-store"}
						}`),
					},
				},
			},
			wantErr: false, // This should pass because the test-store has the correct label
		},
		{
			name: "cluster secret store label selector mismatch",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-label-mismatch-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":    "test-namespace",
							"secretStore":        {"name": "test-store"},
							"clusterSecretStore": {"name": "test-cluster-store"}
						}`),
					},
				},
			},
			wantErr: false, // This should pass because test-cluster-store has the correct label
		},
		{
			name: "cross-namespace not allowed",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cross-ns-not-allowed-run",
					Namespace: "default", // Different namespace
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name:      "k8s-resource-template",
						Namespace: "test-namespace",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":      "test-namespace",
							"secretStore":          {"name": "test-store"},
							"secretStoreNoCrossNS": {"name": "test-store"}
						}`),
					}, // This should fail because it's in a different namespace
				},
			},
			wantErr: true,
			errMsg:  "cross-namespace resource references are not allowed for this parameter",
		},
		{
			name: "valid generator",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"generator": {"name": "test-password-gen-2", "kind": "Fake"}
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid Generator without kind",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"generator": {"name": "test-password-gen-2"}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "must be an object of the format {\"name\": \"store-name\", \"kind\":\"Kind\"}",
		},
		{
			name: "invalid Generator with wrong kind",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"generator": {"name": "test-password-gen-2", "kind": "Password"}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource test-password-gen-2 of type generator[Password] not found in namespace test-namespace",
		},
		{
			name: "invalid Generator with inexistent generator",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"generator": {"name": "test-password-gen-3", "kind": "Fake"}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource test-password-gen-3 of type generator[Fake] not found in namespace test-namespace",
		},
		{
			name: "valid generatorArray",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"generatorArray": [
								{"name": "test-password-gen", "kind": "Password"},
								{"name": "test-password-gen-2", "kind": "Fake"}
							]
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid generatorArray with one element",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"generatorArray": [{"name": "test-password-gen-2", "kind": "Fake"}]
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid GeneratorArray without kind",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"generatorArray": [{"name": "test-password-gen-2"}]
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "must be an object of the format {\"name\": \"store-name\", \"kind\":\"Kind\"}",
		},
		{
			name: "invalid GeneratorArray with wrong kind",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"generatorArray": [{"name": "test-password-gen-2", "kind": "Password"}]
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource test-password-gen-2 of type generator[Password] not found in namespace test-namespace",
		},
		{
			name: "invalid GeneratorArray with inexistent generator",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"generatorArray": [{"name": "test-password-gen-3", "kind": "Fake"}]
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource test-password-gen-3 of type generator[Fake] not found in namespace test-namespace",
		},
		{
			name: "valid secretlocation",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"secretlocation": {
								"name":       "test-store",
								"apiVersion": "external-secrets.io/v1",
								"kind":       "SecretStore",
								"remoteRef": {
									"key": "/foo/bar"
								}
							}
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid secretlocation - no remoteRef",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"secretlocation": {
								"name":       "test-store",
								"apiVersion": "external-secrets.io/v1",
								"kind":       "SecretStore"
							}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "must be an object of the format {\"name\": \"store-name\", \"apiVersion\": \"v1\", \"kind\": \"Kind\", \"remoteRef\": {\"key\": \"remote-key\"}}",
		},
		{
			name: "inexistent secretlocation",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"secretlocation": {
								"name":       "non-existent-store",
								"apiVersion": "external-secrets.io/v1",
								"kind":       "SecretStore",
								"remoteRef": {
									"key": "/foo/bar"
								}
							}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-store of type secretlocation not found in namespace test-namespace",
		},
		{
			name: "valid secretlocation array",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"secretlocationArray": [{
								"name":       "test-store",
								"apiVersion": "external-secrets.io/v1",
								"kind":       "SecretStore",
								"remoteRef": {
									"key": "/foo/bar"
								}
							}, {
								"name":       "test-second-store",
								"apiVersion": "external-secrets.io/v1",
								"kind":       "SecretStore",
								"remoteRef": {
									"key": "/foo/bar"
								}
							}]
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid secretlocation array - no remoteRef",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"secretlocationArray": [{
								"name":       "test-store",
								"apiVersion": "external-secrets.io/v1",
								"kind":       "SecretStore"
							}]
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "must be an object of the format {\"name\": \"store-name\", \"apiVersion\": \"v1\", \"kind\": \"Kind\", \"remoteRef\": {\"key\": \"remote-key\"}}",
		},
		{
			name: "inexistent secretlocation array",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"secretlocationArray": [{
								"name":       "non-existent-store",
								"apiVersion": "external-secrets.io/v1",
								"kind":       "SecretStore",
								"remoteRef": {
									"key": "/foo/bar"
								}
							}]
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-store of type secretlocation not found in namespace test-namespace",
		},
		{
			name: "valid finding",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"finding": {"name": "test-finding"}
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "inexistent finding",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"finding": {"name": "non-existent-finding"}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-finding of type finding not found in namespace test-namespace",
		},
		{
			name: "valid finding array",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"findingArray": [{"name": "test-finding"}, {"name": "test-finding-2"}]
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "inexistent finding in array",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"findingArray": [{"name": "non-existent-finding"}]
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-finding of type finding not found in namespace test-namespace",
		},
		{
			name: "valid object[finding]",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"objectFinding": {"key": {"name": "test-finding"}}
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "inexistent object[finding]",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"objectFinding": {"key": {"name": "non-existent-finding"}}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-finding of type finding not found in namespace test-namespace",
		},
		{
			name: "valid object[array[finding]]",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"objectFindingArray": {"key": [{"name": "test-finding"}, {"name": "test-finding-2"}]}
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "inexistent object[array[finding]]",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"targetNamespace":  "test-namespace",
							"secretStore":      {"name": "test-store"},
							"objectFindingArray": {"key": [{"name": "non-existent-finding"}]}
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-finding of type finding not found in namespace test-namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkflowRunParameters(tt.workflowRun)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateKubernetesResourceTypes(t *testing.T) {
	// Test the parameter type helper methods
	tests := []struct {
		name          string
		paramType     ParameterType
		isK8sResource bool
		apiVersion    string
		kind          string
	}{
		{
			name:          "namespace type",
			paramType:     ParameterTypeNamespace,
			isK8sResource: true,
			apiVersion:    "v1",
			kind:          "Namespace",
		},
		{
			name:          "secretstore type",
			paramType:     ParameterTypeSecretStore,
			isK8sResource: true,
			apiVersion:    "external-secrets.io/v1",
			kind:          "SecretStore",
		},
		{
			name:          "clustersecretstore type",
			paramType:     ParameterTypeClusterSecretStore,
			isK8sResource: true,
			apiVersion:    "external-secrets.io/v1",
			kind:          "ClusterSecretStore",
		},
		{
			name:          "externalsecret type",
			paramType:     ParameterTypeExternalSecret,
			isK8sResource: true,
			apiVersion:    "external-secrets.io/v1",
			kind:          "ExternalSecret",
		},
		{
			name:          "generator type",
			paramType:     ParameterType("generator[Password]"),
			isK8sResource: true,
			apiVersion:    "v1alpha1",
			kind:          "Password",
		},
		{
			name:          "string type",
			paramType:     ParameterTypeString,
			isK8sResource: false,
			apiVersion:    "",
			kind:          "",
		},
		{
			name:          "number type",
			paramType:     ParameterTypeNumber,
			isK8sResource: false,
			apiVersion:    "",
			kind:          "",
		},
		{
			name:          "secret store array type",
			paramType:     ParameterTypeSecretStoreArray,
			isK8sResource: true,
			apiVersion:    "external-secrets.io/v1",
			kind:          "SecretStore",
		},
		{
			name:          "generator array type",
			paramType:     ParameterType("array[generator[Password]]"),
			isK8sResource: true,
			apiVersion:    "v1alpha1",
			kind:          "Password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isK8sResource, tt.paramType.IsKubernetesResource())
			if tt.isK8sResource {
				assert.Equal(t, tt.apiVersion, tt.paramType.GetAPIVersion())
				assert.Equal(t, tt.kind, tt.paramType.GetKind())
			}
		})
	}
}

func TestValidateCustomObjectTypeValidation(t *testing.T) {
	// Create a test scheme
	scheme := runtime.NewScheme()
	_ = AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = esv1.AddToScheme(scheme)

	// Create a test template with Kubernetes resource parameters
	template := &WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-resource-template",
			Namespace: "test-namespace",
		},
		Spec: WorkflowTemplateSpec{
			Version: "v1",
			Name:    "K8s Resource Test",
			ParameterGroups: []ParameterGroup{
				{
					Name: "Resources",
					Parameters: []Parameter{
						{
							Name:     "invalidCustomObject",
							Type:     ParameterType("object[name]invalid-type"),
							Required: true,
						},
					},
				},
			},
			Jobs: map[string]Job{
				"test": {
					Standard: &StandardJob{
						Steps: []Step{
							{
								Name: "test",
								Debug: &DebugStep{
									Message: "Test",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create a fake client with test objects
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			template).
		Build()

	// Set the validation client
	SetValidationClient(client)

	// Test cases
	tests := []struct {
		name        string
		workflowRun *WorkflowRun
		wantErr     bool
		errMsg      string
	}{
		{
			name: "invalid custom object type",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"invalidCustomObject": [{"name": "test-store"}, {"name": "test-second-store"}]
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid value for argument \"invalidCustomObject\": invalid custom object type: object[name]invalid-type. Expected format: object[<arg>]<resource> or object[<arg>]array[<resource>], where <arg> is the name of a previous argument and<resource> is one of: namespace, secretstore, externalsecret, clustersecretstore, secretlocation, finding, or generator[<kind>]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkflowRunParameters(tt.workflowRun)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
