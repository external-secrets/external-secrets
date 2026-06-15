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

package externalsecret

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestSetExternalSecretCondition(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		current        *esv1.ExternalSecret
		new            esv1.ExternalSecretStatusCondition
		expectedStatus esv1.ExternalSecretStatus
	}{
		{
			name: "ExternalSecret has no conditions",
			current: &esv1.ExternalSecret{
				Status: esv1.ExternalSecretStatus{},
			},
			new: esv1.ExternalSecretStatusCondition{
				Type:               esv1.ExternalSecretReady,
				Status:             corev1.ConditionTrue,
				Reason:             "TestReason",
				Message:            "TestMessage",
				LastTransitionTime: metav1.NewTime(now),
			},
			expectedStatus: esv1.ExternalSecretStatus{
				Conditions: []esv1.ExternalSecretStatusCondition{
					{
						Type:               esv1.ExternalSecretReady,
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
			current: &esv1.ExternalSecret{
				Status: esv1.ExternalSecretStatus{
					Conditions: []esv1.ExternalSecretStatusCondition{
						{
							Type:               esv1.ExternalSecretReady,
							Status:             corev1.ConditionTrue,
							Reason:             "TestReason",
							Message:            "TestMessage",
							LastTransitionTime: metav1.NewTime(now),
						},
					},
				},
			},
			new: esv1.ExternalSecretStatusCondition{
				Type:    esv1.ExternalSecretReady,
				Status:  corev1.ConditionTrue,
				Reason:  "TestReason",
				Message: "TestMessage",
			},
			expectedStatus: esv1.ExternalSecretStatus{
				Conditions: []esv1.ExternalSecretStatusCondition{
					{
						Type:               esv1.ExternalSecretReady,
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
			current: &esv1.ExternalSecret{
				Status: esv1.ExternalSecretStatus{
					Conditions: []esv1.ExternalSecretStatusCondition{
						{
							Type:               esv1.ExternalSecretReady,
							Status:             corev1.ConditionTrue,
							Reason:             "TestReason",
							Message:            "TestMessage",
							LastTransitionTime: metav1.NewTime(now.Add(-1 * time.Minute)),
						},
					},
				},
			},
			new: esv1.ExternalSecretStatusCondition{
				Type:               esv1.ExternalSecretReady,
				Status:             corev1.ConditionTrue,
				Reason:             "NewReason",
				Message:            "NewMessage",
				LastTransitionTime: metav1.NewTime(now),
			},
			expectedStatus: esv1.ExternalSecretStatus{
				Conditions: []esv1.ExternalSecretStatusCondition{
					{
						Type:               esv1.ExternalSecretReady,
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
			current: &esv1.ExternalSecret{
				Status: esv1.ExternalSecretStatus{
					Conditions: []esv1.ExternalSecretStatusCondition{
						{
							Type:               esv1.ExternalSecretReady,
							Status:             corev1.ConditionTrue,
							Reason:             "TestReason",
							Message:            "TestMessage",
							LastTransitionTime: metav1.NewTime(now),
						},
					},
				},
			},
			new: esv1.ExternalSecretStatusCondition{
				Type:               esv1.ExternalSecretDeleted,
				Status:             corev1.ConditionTrue,
				Reason:             "NewReason",
				Message:            "NewMessage",
				LastTransitionTime: metav1.NewTime(now),
			},
			expectedStatus: esv1.ExternalSecretStatus{
				Conditions: []esv1.ExternalSecretStatusCondition{
					{
						Type:               esv1.ExternalSecretReady,
						Status:             corev1.ConditionTrue,
						Reason:             "TestReason",
						Message:            "TestMessage",
						LastTransitionTime: metav1.NewTime(now),
					},
					{
						Type:               esv1.ExternalSecretDeleted,
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

func TestDiffSecretDataKeys(t *testing.T) {
	tests := []struct {
		name        string
		oldData     map[string][]byte
		newData     map[string][]byte
		wantAdded   []string
		wantUpdated []string
		wantRemoved []string
		wantEmptied []string
	}{
		{
			name:    "no change",
			oldData: map[string][]byte{"a": []byte("1"), "b": []byte("2")},
			newData: map[string][]byte{"a": []byte("1"), "b": []byte("2")},
		},
		{
			name:      "key added",
			oldData:   map[string][]byte{"a": []byte("1")},
			newData:   map[string][]byte{"a": []byte("1"), "b": []byte("2")},
			wantAdded: []string{"b"},
		},
		{
			name:        "key removed",
			oldData:     map[string][]byte{"a": []byte("1"), "b": []byte("2")},
			newData:     map[string][]byte{"a": []byte("1")},
			wantRemoved: []string{"b"},
		},
		{
			name:        "value changed",
			oldData:     map[string][]byte{"a": []byte("1")},
			newData:     map[string][]byte{"a": []byte("2")},
			wantUpdated: []string{"a"},
		},
		{
			name:        "existing value emptied",
			oldData:     map[string][]byte{"a": []byte("1")},
			newData:     map[string][]byte{"a": []byte("")},
			wantUpdated: []string{"a"},
			wantEmptied: []string{"a"},
		},
		{
			name:      "new key added empty",
			oldData:   map[string][]byte{"a": []byte("1")},
			newData:   map[string][]byte{"a": []byte("1"), "b": []byte("")},
			wantAdded: []string{"b"},
			// b is both added and emptied; emptied flags the empty value.
			wantEmptied: []string{"b"},
		},
		{
			name:      "results are sorted",
			oldData:   map[string][]byte{},
			newData:   map[string][]byte{"c": []byte("1"), "a": []byte("2"), "b": []byte("3")},
			wantAdded: []string{"a", "b", "c"},
		},
		{
			name:        "mixed add update remove empty",
			oldData:     map[string][]byte{"keep": []byte("x"), "change": []byte("old"), "gone": []byte("y")},
			newData:     map[string][]byte{"keep": []byte("x"), "change": []byte("new"), "fresh": []byte(""), "added": []byte("z")},
			wantAdded:   []string{"added", "fresh"},
			wantUpdated: []string{"change"},
			wantRemoved: []string{"gone"},
			wantEmptied: []string{"fresh"},
		},
		{
			name:    "nil maps are safe",
			oldData: nil,
			newData: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, updated, removed, emptied := diffSecretDataKeys(tt.oldData, tt.newData)
			if diff := cmp.Diff(tt.wantAdded, added); diff != "" {
				t.Errorf("added (-want, +got)\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantUpdated, updated); diff != "" {
				t.Errorf("updated (-want, +got)\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantRemoved, removed); diff != "" {
				t.Errorf("removed (-want, +got)\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantEmptied, emptied); diff != "" {
				t.Errorf("emptied (-want, +got)\n%s", diff)
			}
		})
	}
}
