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

package reporter

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

var (
	RecorderInstance record.EventRecorder
	ReporterInstance Reporter
)

func init() {
	eventBroadcaster := record.NewBroadcaster()
	RecorderInstance = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "ESO Recorder"})
	ReporterInstance = *NewReporter(RecorderInstance)
}

const (
	readyMessage = "External provider authenticated successfully"
)

type Reporter struct {
	recorder record.EventRecorder
}

func NewReporter(recorder record.EventRecorder) *Reporter {
	return &Reporter{
		recorder: recorder,
	}
}

// Failed marks a generic SecretStore Auth attempt as terminally failed and sends a corresponding event.
func (r *Reporter) Failed(store esv1alpha1.GenericStore, err error, reason, message string) {
	storeKind := store.GetObjectKind().GroupVersionKind().Kind
	if storeKind == esv1alpha1.ClusterSecretStoreKind {
		ClusterStore := store.(*esv1alpha1.ClusterSecretStore)
		if ClusterStore.Status.FailureTime == nil {
			nowTime := metav1.NewTime(time.Now())
			ClusterStore.Status.FailureTime = &nowTime
		}
		message = fmt.Sprintf("%s: %v", message, err)
		r.recorder.Event(ClusterStore, corev1.EventTypeWarning, reason, message)
	} else {
		SecretStore := store.(*esv1alpha1.SecretStore)
		if SecretStore.Status.FailureTime == nil {
			nowTime := metav1.NewTime(time.Now())
			SecretStore.Status.FailureTime = &nowTime
		}
		message = fmt.Sprintf("%s: %v", message, err)
		r.recorder.Event(SecretStore, corev1.EventTypeWarning, reason, message)
	}
}

// Ready marks a generic SecretStore as Ready and sends a corresponding event.
func (r *Reporter) Ready(store esv1alpha1.GenericStore) {
	storeKind := store.GetObjectKind().GroupVersionKind().Kind
	if storeKind == esv1alpha1.ClusterSecretStoreKind {
		ClusterStore := store.(*esv1alpha1.ClusterSecretStore)
		r.recorder.Event(ClusterStore, corev1.EventTypeNormal, "Store Authenticated", readyMessage)
	} else {
		SecretStore := store.(*esv1alpha1.SecretStore)
		r.recorder.Event(SecretStore, corev1.EventTypeNormal, "Store Authenticated", readyMessage)
	}
}
