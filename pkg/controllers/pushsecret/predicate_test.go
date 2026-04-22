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

package pushsecret

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func TestPushSecretWatchPredicate_Update(t *testing.T) {
	now := metav1.NewTime(time.Now())

	base := func() *esapi.PushSecret {
		return &esapi.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "ps",
				Namespace:   "ns",
				Generation:  1,
				Labels:      map[string]string{"app": "demo"},
				Annotations: map[string]string{"owner": "guppi"},
			},
		}
	}

	tests := []struct {
		name   string
		mutate func(newObj *esapi.PushSecret)
		want   bool
	}{
		{
			name: "status-only change is filtered out",
			mutate: func(newObj *esapi.PushSecret) {
				newObj.Status.SyncedResourceVersion = "rv-2"
				newObj.Status.RefreshTime = now
			},
			want: false,
		},
		{
			name: "generation bump triggers reconcile",
			mutate: func(newObj *esapi.PushSecret) {
				newObj.Generation = 2
			},
			want: true,
		},
		{
			name: "label change triggers reconcile",
			mutate: func(newObj *esapi.PushSecret) {
				newObj.Labels["app"] = "demo2"
			},
			want: true,
		},
		{
			name: "annotation change triggers reconcile",
			mutate: func(newObj *esapi.PushSecret) {
				newObj.Annotations["owner"] = "captain"
			},
			want: true,
		},
		{
			name: "deletion timestamp appearance triggers reconcile",
			mutate: func(newObj *esapi.PushSecret) {
				newObj.DeletionTimestamp = &now
			},
			want: true,
		},
	}

	pred := pushSecretWatchPredicate()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldObj := base()
			newObj := base()
			tt.mutate(newObj)

			got := pred.Update(event.UpdateEvent{
				ObjectOld: oldObj,
				ObjectNew: newObj,
			})
			if got != tt.want {
				t.Errorf("predicate.Update = %v, want %v", got, tt.want)
			}
		})
	}
}
