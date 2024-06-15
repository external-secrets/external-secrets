package workflow

import (
	"context"
	"reflect"
	"testing"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestWorkflowRunner_Run(t *testing.T) {
	tests := []struct {
		name      string
		workflows []v1alpha1.WorkflowItem
		expected  map[string]interface{}
	}{
		{
			name: "SingleStepWorkflow",
			workflows: []v1alpha1.WorkflowItem{
				{
					Name: "workflow1",
					Steps: []v1alpha1.WorkflowStep{
						{
							Name: "step1",
							Pull: &v1alpha1.WorkflowStepPull{
								Source: v1beta1.StoreSourceRef{
									SecretStoreRef: v1beta1.SecretStoreRef{
										Name: "store1",
									},
								},
								Data: []v1beta1.ExternalSecretData{
									{
										SecretKey: "key1",
										RemoteRef: v1beta1.ExternalSecretDataRemoteRef{
											Key: "key1",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"workflow": map[string]interface{}{
					"data": map[string]string{
						"key1": "value1",
					},
					"metadata": map[string]map[string]string{
						"annotations": {},
						"labels":      {},
					},
				},
				"workflows": map[string]interface{}{
					"workflow1": map[string]interface{}{
						"data": map[string]string{
							"key1": "value1",
						},
						"metadata": map[string]map[string]string{
							"annotations": {},
							"labels":      {},
						},
					},
				},
			},
		},
		{

			name: "MultipleStepWorkflow",
			workflows: []v1alpha1.WorkflowItem{
				{
					Name: "workflow1",
					Steps: []v1alpha1.WorkflowStep{
						{
							Name: "step1",
							Pull: &v1alpha1.WorkflowStepPull{
								Source: v1beta1.StoreSourceRef{
									SecretStoreRef: v1beta1.SecretStoreRef{
										Name: "store1",
									},
								},
								Data: []v1beta1.ExternalSecretData{
									{
										SecretKey: "key1",
										RemoteRef: v1beta1.ExternalSecretDataRemoteRef{
											Key: "key1",
										},
									},
								},
							},
						},
						{
							Name: "step2",
							Pull: &v1alpha1.WorkflowStepPull{
								Source: v1beta1.StoreSourceRef{
									SecretStoreRef: v1beta1.SecretStoreRef{
										Name: "store1",
									},
								},
								Data: []v1beta1.ExternalSecretData{
									{
										SecretKey: "key2",
										RemoteRef: v1beta1.ExternalSecretDataRemoteRef{
											Key: "key2",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"workflow": map[string]interface{}{
					"data": map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
					"metadata": map[string]map[string]string{
						"annotations": {},
						"labels":      {},
					},
				},
				"workflows": map[string]interface{}{
					"workflow1": map[string]interface{}{
						"data": map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
						"metadata": map[string]map[string]string{
							"annotations": {},
							"labels":      {},
						},
					},
				},
			},
		},
		{

			name: "MultipleWorkflows",
			workflows: []v1alpha1.WorkflowItem{
				{
					Name: "workflow1",
					Steps: []v1alpha1.WorkflowStep{
						{
							Name: "step1",
							Pull: &v1alpha1.WorkflowStepPull{
								Source: v1beta1.StoreSourceRef{
									SecretStoreRef: v1beta1.SecretStoreRef{
										Name: "store1",
									},
								},
								Data: []v1beta1.ExternalSecretData{
									{
										SecretKey: "key1",
										RemoteRef: v1beta1.ExternalSecretDataRemoteRef{
											Key: "key1",
										},
									},
								},
							},
						},
					},
				},
				{
					Name: "workflow2",
					Steps: []v1alpha1.WorkflowStep{
						{
							Name: "step1",
							Pull: &v1alpha1.WorkflowStepPull{
								Source: v1beta1.StoreSourceRef{
									SecretStoreRef: v1beta1.SecretStoreRef{
										Name: "store1",
									},
								},
								Data: []v1beta1.ExternalSecretData{
									{
										SecretKey: "key2",
										RemoteRef: v1beta1.ExternalSecretDataRemoteRef{
											Key: "key2",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"workflow": map[string]interface{}{
					"data": map[string]string{
						"key2": "value2",
					},
					"metadata": map[string]map[string]string{
						"annotations": {},
						"labels":      {},
					},
				},
				"workflows": map[string]interface{}{
					"workflow1": map[string]interface{}{
						"data": map[string]string{
							"key1": "value1",
						},
						"metadata": map[string]map[string]string{
							"annotations": {},
							"labels":      {},
						},
					},
					"workflow2": map[string]interface{}{
						"data": map[string]string{
							"key2": "value2",
						},
						"metadata": map[string]map[string]string{
							"annotations": {},
							"labels":      {},
						},
					},
				},
			},
		},
		{

			name: "ChainWorkflows",
			workflows: []v1alpha1.WorkflowItem{
				{
					Name: "workflow1",
					Steps: []v1alpha1.WorkflowStep{
						{
							Name: "step1",
							Pull: &v1alpha1.WorkflowStepPull{
								Source: v1beta1.StoreSourceRef{
									SecretStoreRef: v1beta1.SecretStoreRef{
										Name: "store1",
									},
								},
								Data: []v1beta1.ExternalSecretData{
									{
										SecretKey: "key1",
										RemoteRef: v1beta1.ExternalSecretDataRemoteRef{
											Key: "key1",
										},
									},
								},
							},
						},
					},
				},
				{
					Name: "workflow2",
					Steps: []v1alpha1.WorkflowStep{
						{
							Name: "step1",
							Pull: &v1alpha1.WorkflowStepPull{
								Source: v1beta1.StoreSourceRef{
									SecretStoreRef: v1beta1.SecretStoreRef{
										Name: "store1",
									},
								},
								Data: []v1beta1.ExternalSecretData{
									{
										SecretKey: "key2",
										RemoteRef: v1beta1.ExternalSecretDataRemoteRef{
											Key: "key2",
										},
									},
								},
							},
						},
					},
				},
				{
					Name: "workflow3",
					Steps: []v1alpha1.WorkflowStep{
						{
							Name: "step1",
							Template: &v1alpha1.WorkflowTemplate{
								Data: map[string]string{
									"aggregated": "{{ .workflows.workflow1.data.key1 }} and {{ .workflows.workflow2.data.key2 }}",
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"workflow": map[string]interface{}{
					"data": map[string]string{
						"aggregated": "value1 and value2",
					},
					"metadata": map[string]map[string]string{
						"annotations": {},
						"labels":      {},
					},
				},
				"workflows": map[string]interface{}{
					"workflow1": map[string]interface{}{
						"data": map[string]string{
							"key1": "value1",
						},
						"metadata": map[string]map[string]string{
							"annotations": {},
							"labels":      {},
						},
					},
					"workflow2": map[string]interface{}{
						"data": map[string]string{
							"key2": "value2",
						},
						"metadata": map[string]map[string]string{
							"annotations": {},
							"labels":      {},
						},
					},
					"workflow3": map[string]interface{}{
						"data": map[string]string{
							"aggregated": "value1 and value2",
						},
						"metadata": map[string]map[string]string{
							"annotations": {},
							"labels":      {},
						},
					},
				},
			},
		},
	}

	ctx := context.TODO()
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	v1beta1.AddToScheme(scheme)

	namespace := "test-namespace"
	store1 := &v1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "store1",
			Namespace: namespace,
		},
		Status: v1beta1.SecretStoreStatus{
			Conditions: []v1beta1.SecretStoreStatusCondition{
				{
					Type:   v1beta1.SecretStoreReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		Spec: v1beta1.SecretStoreSpec{
			Provider: &v1beta1.SecretStoreProvider{
				Fake: &v1beta1.FakeProvider{
					Data: []v1beta1.FakeProviderData{
						{
							Key:   "key1",
							Value: "value1",
						},
						{
							Key:   "key2",
							Value: "value2",
						},
					},
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(store1).Build()
	log := logr.Discard()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewWorkflowRunner(ctx, client, namespace, tt.workflows, log)
			err := runner.Run()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(runner.inputs.ToMap(), tt.expected) {
				t.Errorf("unexpected result: got %v, want %v", runner.inputs.ToMap(), tt.expected)
			}
		})
	}
}
