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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	keyName = "my-key"
)

func newExternalSecretV1Alpha1() *ExternalSecret {
	return &ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "full-es",
			Namespace: "my-ns",
		},
		Status: ExternalSecretStatus{
			SyncedResourceVersion: "123",
			Conditions: []ExternalSecretStatusCondition{
				{
					Type:    ExternalSecretReady,
					Status:  corev1.ConditionTrue,
					Reason:  "it's a mock, it's always ready",
					Message: "...why wouldn't it be?",
				},
			},
		},
		Spec: ExternalSecretSpec{
			SecretStoreRef: SecretStoreRef{
				Name: "test-secret-store",
				Kind: "ClusterSecretStore",
			},
			Target: ExternalSecretTarget{
				Name:           "test-target",
				CreationPolicy: Owner,
				Immutable:      false,
				Template: &ExternalSecretTemplate{
					Type: corev1.SecretTypeOpaque,
					Metadata: ExternalSecretTemplateMetadata{
						Annotations: map[string]string{
							"foo": "bar",
						},
						Labels: map[string]string{
							"foolbl": "barlbl",
						},
					},
					Data: map[string]string{
						keyName: "{{.data | toString}}",
					},
					TemplateFrom: []TemplateFrom{
						{
							ConfigMap: &TemplateRef{
								Name: "test-configmap",
								Items: []TemplateRefItem{
									{
										Key: keyName,
									},
								},
							},
							Secret: &TemplateRef{
								Name: "test-secret",
								Items: []TemplateRefItem{
									{
										Key: keyName,
									},
								},
							},
						},
					},
				},
			},
			Data: []ExternalSecretData{
				{
					SecretKey: keyName,
					RemoteRef: ExternalSecretDataRemoteRef{
						Key:      "datakey",
						Property: "dataproperty",
						Version:  "dataversion",
					},
				},
			},
			DataFrom: []ExternalSecretDataRemoteRef{
				{
					Key:      "key",
					Property: "property",
					Version:  "version",
				},
			},
		},
	}
}

func newExternalSecretV1Beta1() *esv1beta1.ExternalSecret {
	return &esv1beta1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "full-es",
			Namespace: "my-ns",
		},
		Status: esv1beta1.ExternalSecretStatus{
			SyncedResourceVersion: "123",
			Conditions: []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:    esv1beta1.ExternalSecretReady,
					Status:  corev1.ConditionTrue,
					Reason:  "it's a mock, it's always ready",
					Message: "...why wouldn't it be?",
				},
			},
		},
		Spec: esv1beta1.ExternalSecretSpec{
			SecretStoreRef: esv1beta1.SecretStoreRef{
				Name: "test-secret-store",
				Kind: "ClusterSecretStore",
			},
			Target: esv1beta1.ExternalSecretTarget{
				Name:           "test-target",
				CreationPolicy: esv1beta1.CreatePolicyOwner,
				Immutable:      false,
				Template: &esv1beta1.ExternalSecretTemplate{
					Type: corev1.SecretTypeOpaque,
					Metadata: esv1beta1.ExternalSecretTemplateMetadata{
						Annotations: map[string]string{
							"foo": "bar",
						},
						Labels: map[string]string{
							"foolbl": "barlbl",
						},
					},
					Data: map[string]string{
						keyName: "{{.data | toString}}",
					},
					TemplateFrom: []esv1beta1.TemplateFrom{
						{
							ConfigMap: &esv1beta1.TemplateRef{
								Name: "test-configmap",
								Items: []esv1beta1.TemplateRefItem{
									{
										Key: keyName,
									},
								},
							},
							Secret: &esv1beta1.TemplateRef{
								Name: "test-secret",
								Items: []esv1beta1.TemplateRefItem{
									{
										Key: keyName,
									},
								},
							},
						},
					},
				},
			},
			Data: []esv1beta1.ExternalSecretData{
				{
					SecretKey: keyName,
					RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
						Key:      "datakey",
						Property: "dataproperty",
						Version:  "dataversion",
					},
				},
			},
			DataFrom: []esv1beta1.ExternalSecretDataFromRemoteRef{
				{
					Extract: &esv1beta1.ExternalSecretDataRemoteRef{
						Key:      "key",
						Property: "property",
						Version:  "version",
					},
				},
			},
		},
	}
}

func TestExternalSecretConvertFrom(t *testing.T) {
	given := newExternalSecretV1Beta1()
	want := newExternalSecretV1Alpha1()
	got := &ExternalSecret{}
	err := got.ConvertFrom(given)
	if err != nil {
		t.Errorf("test failed with error: %v", err)
	}
	if !assert.Equal(t, want, got) {
		t.Errorf("test failed, expected: %v, got: %v", want, got)
	}
}

func TestExternalSecretConvertTo(t *testing.T) {
	want := newExternalSecretV1Beta1()
	given := newExternalSecretV1Alpha1()
	got := &esv1beta1.ExternalSecret{}
	err := given.ConvertTo(got)
	if err != nil {
		t.Errorf("test failed with error: %v", err)
	}
	if !assert.Equal(t, want, got) {
		t.Errorf("test failed, expected: %v, got: %v", want, got)
	}
}
