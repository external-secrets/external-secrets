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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	ctrlutil "github.com/external-secrets/external-secrets/pkg/controllers/util"
)

// secretValue is the shape that leaked through a PushSecret condition in
// external-secrets#5884: json.Unmarshal echoes the offending value.
const secretValue = "8019210420527506405"

func markAsFailedFixture() (*Reconciler, *esv1.ExternalSecret, prometheus.Counter) {
	r := &Reconciler{recorder: record.NewFakeRecorder(10)}
	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{Name: "es", Namespace: "default"},
	}
	return r, es, prometheus.NewCounter(prometheus.CounterOpts{Name: "test_sync_errors"})
}

func readyCondition(t *testing.T, es *esv1.ExternalSecret) *esv1.ExternalSecretStatusCondition {
	t.Helper()
	cond := esv1.GetExternalSecretCondition(es.Status, esv1.ExternalSecretReady)
	if cond == nil {
		t.Fatal("no Ready condition was set")
	}
	return cond
}

func TestMarkAsFailedKeepsUnsafeErrorsOutOfCondition(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "provider unmarshal error",
			err:  fmt.Errorf("json: cannot unmarshal number %s into Go value of type float64", secretValue),
		},
		{
			name: "template render error",
			err:  fmt.Errorf(errApplyTemplate, fmt.Errorf("executing template: bad value %s", secretValue)),
		},
		{
			name: "provider error wrapping a marked error",
			err:  fmt.Errorf("provider said %s: %w", secretValue, ctrlutil.Safe(errors.New("connection refused"))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, es, counter := markAsFailedFixture()

			r.markAsFailed(msgErrorGetSecretData, tt.err, es, counter, esv1.ConditionReasonSecretSyncedError)

			cond := readyCondition(t, es)
			if strings.Contains(cond.Message, secretValue) {
				t.Fatalf("condition message leaked provider text: %q", cond.Message)
			}
			if cond.Status != v1.ConditionFalse {
				t.Errorf("status = %v, want False", cond.Status)
			}
		})
	}
}

func TestMarkAsFailedDetailsSafeErrors(t *testing.T) {
	r, es, counter := markAsFailedFixture()
	err := ctrlutil.Safe(fmt.Errorf(errUpdate, "my-secret", ErrSecretImmutable))

	r.markAsFailed(msgErrorUpdateImmutable, err, es, counter, esv1.ConditionReasonSecretImmutable)

	cond := readyCondition(t, es)
	if cond.Reason != esv1.ConditionReasonSecretImmutable {
		t.Errorf("reason = %q, want %q", cond.Reason, esv1.ConditionReasonSecretImmutable)
	}
	if !strings.HasPrefix(cond.Message, msgErrorUpdateImmutable) {
		t.Errorf("message %q lost the base text", cond.Message)
	}
	if !strings.Contains(cond.Message, "my-secret") {
		t.Errorf("message %q dropped the safe detail", cond.Message)
	}
}

// A long safe error must not grow the condition message without bound: there is
// no maxLength on the Message field in the CRDs.
func TestMarkAsFailedCapsMessageLength(t *testing.T) {
	r, es, counter := markAsFailedFixture()
	err := ctrlutil.Safe(errors.New(strings.Repeat("a", 4096)))

	r.markAsFailed(msgErrorUpdateSecret, err, es, counter, esv1.ConditionReasonSecretSyncedError)

	cond := readyCondition(t, es)
	limit := len(msgErrorUpdateSecret) + ctrlutil.MaxConditionMessageLength + len(": ...")
	if len(cond.Message) > limit {
		t.Errorf("message length = %d, want at most %d", len(cond.Message), limit)
	}
}
