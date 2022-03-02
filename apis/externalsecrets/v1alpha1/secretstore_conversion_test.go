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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	storeName                = "secret-store"
	storeNamespace           = "my-namespace"
	storeReason              = "it's a mock, it's always ready"
	storeMessage             = "...why wouldn't it be?"
	storeAWSRegion           = "us-east-1"
	storeAWSRole             = "arn:aws:iam::123456789012:role/my-role"
	storeAccessName          = "my-access"
	storeKey                 = "my-key"
	storeSecretName          = "my-secret"
	defaultErrorMessage      = "test failed with error: %v"
	defaultComparisonMessage = "test failed, expected: %v, got: %v"
)

func newSecretStoreV1Alpha1() *SecretStore {
	return &SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storeName,
			Namespace: storeNamespace,
		},
		Status: SecretStoreStatus{
			Conditions: []SecretStoreStatusCondition{
				{
					Type:    SecretStoreReady,
					Status:  corev1.ConditionTrue,
					Reason:  storeReason,
					Message: storeMessage,
				},
			},
		},
		Spec: SecretStoreSpec{
			Controller: "dev",
			Provider: &SecretStoreProvider{
				AWS: &AWSProvider{
					Service: AWSServiceSecretsManager,
					Region:  storeAWSRegion,
					Role:    storeAWSRole,
					Auth: AWSAuth{
						SecretRef: &AWSAuthSecretRef{
							AccessKeyID: esmeta.SecretKeySelector{
								Name: storeAccessName,
								Key:  storeKey,
							},
							SecretAccessKey: esmeta.SecretKeySelector{
								Name: storeSecretName,
								Key:  storeKey,
							},
						},
					},
				},
			},
		},
	}
}

func newSecretStoreV1Beta1() *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storeName,
			Namespace: storeNamespace,
		},
		Status: esv1beta1.SecretStoreStatus{
			Conditions: []esv1beta1.SecretStoreStatusCondition{
				{
					Type:    esv1beta1.SecretStoreReady,
					Status:  corev1.ConditionTrue,
					Reason:  storeReason,
					Message: storeMessage,
				},
			},
		},
		Spec: esv1beta1.SecretStoreSpec{
			Controller: "dev",
			Provider: &esv1beta1.SecretStoreProvider{
				AWS: &esv1beta1.AWSProvider{
					Service: esv1beta1.AWSServiceSecretsManager,
					Region:  storeAWSRegion,
					Role:    storeAWSRole,
					Auth: esv1beta1.AWSAuth{
						SecretRef: &esv1beta1.AWSAuthSecretRef{
							AccessKeyID: esmeta.SecretKeySelector{
								Name: storeAccessName,
								Key:  storeKey,
							},
							SecretAccessKey: esmeta.SecretKeySelector{
								Name: storeSecretName,
								Key:  storeKey,
							},
						},
					},
				},
			},
		},
	}
}

func newClusterSecretStoreV1Alpha1() *ClusterSecretStore {
	ns := storeNamespace
	return &ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: storeName,
		},
		Status: SecretStoreStatus{
			Conditions: []SecretStoreStatusCondition{
				{
					Type:    SecretStoreReady,
					Status:  corev1.ConditionTrue,
					Reason:  storeReason,
					Message: storeMessage,
				},
			},
		},
		Spec: SecretStoreSpec{
			Controller: "dev",
			Provider: &SecretStoreProvider{
				AWS: &AWSProvider{
					Service: AWSServiceSecretsManager,
					Region:  storeAWSRegion,
					Role:    storeAWSRole,
					Auth: AWSAuth{
						SecretRef: &AWSAuthSecretRef{
							AccessKeyID: esmeta.SecretKeySelector{
								Name:      storeAccessName,
								Key:       storeKey,
								Namespace: &ns,
							},
							SecretAccessKey: esmeta.SecretKeySelector{
								Name:      storeSecretName,
								Key:       storeKey,
								Namespace: &ns,
							},
						},
					},
				},
			},
		},
	}
}

func newClusterSecretStoreV1Beta1() *esv1beta1.ClusterSecretStore {
	ns := storeNamespace
	return &esv1beta1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: storeName,
		},
		Status: esv1beta1.SecretStoreStatus{
			Conditions: []esv1beta1.SecretStoreStatusCondition{
				{
					Type:    esv1beta1.SecretStoreReady,
					Status:  corev1.ConditionTrue,
					Reason:  storeReason,
					Message: storeMessage,
				},
			},
		},
		Spec: esv1beta1.SecretStoreSpec{
			Controller: "dev",
			Provider: &esv1beta1.SecretStoreProvider{
				AWS: &esv1beta1.AWSProvider{
					Service: esv1beta1.AWSServiceSecretsManager,
					Region:  storeAWSRegion,
					Role:    storeAWSRole,
					Auth: esv1beta1.AWSAuth{
						SecretRef: &esv1beta1.AWSAuthSecretRef{
							AccessKeyID: esmeta.SecretKeySelector{
								Name:      storeAccessName,
								Key:       storeKey,
								Namespace: &ns,
							},
							SecretAccessKey: esmeta.SecretKeySelector{
								Name:      storeSecretName,
								Key:       storeKey,
								Namespace: &ns,
							},
						},
					},
				},
			},
		},
	}
}
func TestSecretStoreConvertFrom(t *testing.T) {
	given := newSecretStoreV1Beta1()
	want := newSecretStoreV1Alpha1()
	got := &SecretStore{}
	err := got.ConvertFrom(given)
	if err != nil {
		t.Errorf(defaultErrorMessage, err)
	}
	if !assert.Equal(t, want, got) {
		t.Errorf("test failed, expected: %v, got: %v", want, got)
	}
}

func TestSecretStoreConvertTo(t *testing.T) {
	want := newSecretStoreV1Beta1()
	given := newSecretStoreV1Alpha1()
	got := &esv1beta1.SecretStore{}
	err := given.ConvertTo(got)
	if err != nil {
		t.Errorf(defaultErrorMessage, err)
	}
	if !assert.Equal(t, want, got) {
		t.Errorf(defaultComparisonMessage, want, got)
	}
}

func TestClusterSecretStoreConvertFrom(t *testing.T) {
	given := newClusterSecretStoreV1Beta1()
	want := newClusterSecretStoreV1Alpha1()
	got := &ClusterSecretStore{}
	err := got.ConvertFrom(given)
	if err != nil {
		t.Errorf(defaultErrorMessage, err)
	}
	if !assert.Equal(t, want, got) {
		t.Errorf(defaultComparisonMessage, want, got)
	}
}

func TestClusterSecretStoreConvertTo(t *testing.T) {
	want := newClusterSecretStoreV1Beta1()
	given := newClusterSecretStoreV1Alpha1()
	got := &esv1beta1.ClusterSecretStore{}
	err := given.ConvertTo(got)
	if err != nil {
		t.Errorf(defaultErrorMessage, err)
	}
	if !assert.Equal(t, want, got) {
		t.Errorf(defaultComparisonMessage, want, got)
	}
}
