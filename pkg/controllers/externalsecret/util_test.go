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

package externalsecret

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func TestGetExternalSecretCondition(t *testing.T) {
	status := esv1beta1.ExternalSecretStatus{
		Conditions: []esv1beta1.ExternalSecretStatusCondition{
			{
				Type:   esv1beta1.ExternalSecretReady,
				Status: corev1.ConditionFalse,
			},
			{
				Type:   esv1beta1.ExternalSecretReady,
				Status: corev1.ConditionTrue,
			},
		},
	}

	tests := []struct {
		name     string
		condType esv1beta1.ExternalSecretConditionType
		expected *esv1beta1.ExternalSecretStatusCondition
	}{
		{
			name:     "Status has a condition of the specified type",
			condType: esv1beta1.ExternalSecretReady,
			expected: &esv1beta1.ExternalSecretStatusCondition{
				Type:   esv1beta1.ExternalSecretReady,
				Status: corev1.ConditionFalse,
			},
		},
		{
			name:     "Status does not have a condition of the specified type",
			condType: esv1beta1.ExternalSecretDeleted,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExternalSecretCondition(status, tt.condType)

			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		})
	}
}

func TestSetExternalSecretCondition(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		current        *esv1beta1.ExternalSecret
		new            esv1beta1.ExternalSecretStatusCondition
		expectedStatus esv1beta1.ExternalSecretStatus
	}{
		{
			name: "ExternalSecret has no conditions",
			current: &esv1beta1.ExternalSecret{
				Status: esv1beta1.ExternalSecretStatus{},
			},
			new: esv1beta1.ExternalSecretStatusCondition{
				Type:               esv1beta1.ExternalSecretReady,
				Status:             corev1.ConditionTrue,
				Reason:             "TestReason",
				Message:            "TestMessage",
				LastTransitionTime: metav1.NewTime(now),
			},
			expectedStatus: esv1beta1.ExternalSecretStatus{
				Conditions: []esv1beta1.ExternalSecretStatusCondition{
					{
						Type:               esv1beta1.ExternalSecretReady,
						Status:             corev1.ConditionTrue,
						Reason:             "TestReason",
						Message:            "TestMessage",
						LastTransitionTime: metav1.NewTime(now),
					},
				},
			},
		},
		{
			name: "ExternalSecret already has the same condition",
			current: &esv1beta1.ExternalSecret{
				Status: esv1beta1.ExternalSecretStatus{
					Conditions: []esv1beta1.ExternalSecretStatusCondition{
						{
							Type:               esv1beta1.ExternalSecretReady,
							Status:             corev1.ConditionTrue,
							Reason:             "TestReason",
							Message:            "TestMessage",
							LastTransitionTime: metav1.NewTime(now),
						},
					},
				},
			},
			new: esv1beta1.ExternalSecretStatusCondition{
				Type:    esv1beta1.ExternalSecretReady,
				Status:  corev1.ConditionTrue,
				Reason:  "TestReason",
				Message: "TestMessage",
			},
			expectedStatus: esv1beta1.ExternalSecretStatus{
				Conditions: []esv1beta1.ExternalSecretStatusCondition{
					{
						Type:               esv1beta1.ExternalSecretReady,
						Status:             corev1.ConditionTrue,
						Reason:             "TestReason",
						Message:            "TestMessage",
						LastTransitionTime: metav1.NewTime(now),
					},
				},
			},
		},
		{
			name: "ExternalSecret has a different condition with the same type and status",
			current: &esv1beta1.ExternalSecret{
				Status: esv1beta1.ExternalSecretStatus{
					Conditions: []esv1beta1.ExternalSecretStatusCondition{
						{
							Type:               esv1beta1.ExternalSecretReady,
							Status:             corev1.ConditionTrue,
							Reason:             "TestReason",
							Message:            "TestMessage",
							LastTransitionTime: metav1.NewTime(now.Add(-1 * time.Minute)),
						},
					},
				},
			},
			new: esv1beta1.ExternalSecretStatusCondition{
				Type:               esv1beta1.ExternalSecretReady,
				Status:             corev1.ConditionTrue,
				Reason:             "NewReason",
				Message:            "NewMessage",
				LastTransitionTime: metav1.NewTime(now),
			},
			expectedStatus: esv1beta1.ExternalSecretStatus{
				Conditions: []esv1beta1.ExternalSecretStatusCondition{
					{
						Type:               esv1beta1.ExternalSecretReady,
						Status:             corev1.ConditionTrue,
						Reason:             "NewReason",
						Message:            "NewMessage",
						LastTransitionTime: metav1.NewTime(now.Add(-1 * time.Minute)),
					},
				},
			},
		},
		{
			name: "ExternalSecret has a different condition",
			current: &esv1beta1.ExternalSecret{
				Status: esv1beta1.ExternalSecretStatus{
					Conditions: []esv1beta1.ExternalSecretStatusCondition{
						{
							Type:               esv1beta1.ExternalSecretReady,
							Status:             corev1.ConditionTrue,
							Reason:             "TestReason",
							Message:            "TestMessage",
							LastTransitionTime: metav1.NewTime(now),
						},
					},
				},
			},
			new: esv1beta1.ExternalSecretStatusCondition{
				Type:               esv1beta1.ExternalSecretDeleted,
				Status:             corev1.ConditionTrue,
				Reason:             "NewReason",
				Message:            "NewMessage",
				LastTransitionTime: metav1.NewTime(now),
			},
			expectedStatus: esv1beta1.ExternalSecretStatus{
				Conditions: []esv1beta1.ExternalSecretStatusCondition{
					{
						Type:               esv1beta1.ExternalSecretReady,
						Status:             corev1.ConditionTrue,
						Reason:             "TestReason",
						Message:            "TestMessage",
						LastTransitionTime: metav1.NewTime(now),
					},
					{
						Type:               esv1beta1.ExternalSecretDeleted,
						Status:             corev1.ConditionTrue,
						Reason:             "NewReason",
						Message:            "NewMessage",
						LastTransitionTime: metav1.NewTime(now),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetExternalSecretCondition(tt.current, tt.new)

			if diff := cmp.Diff(tt.current.Status, tt.expectedStatus); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		})
	}
}
