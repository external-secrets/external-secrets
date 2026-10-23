/*
Copyright © The ESO Authors

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

package pushsecret

import (
	"errors"
	"strings"
	"testing"

	"k8s.io/client-go/tools/record"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctrlutil "github.com/external-secrets/external-secrets/pkg/controllers/util"
)

const pushSecretSentinel = "sentinel-secret-value"

func TestMarkAsFailedKeepsUnsafeErrorsOutOfStatusAndEvents(t *testing.T) {
	recorder := record.NewFakeRecorder(1)
	r := &Reconciler{recorder: recorder}
	ps := &esapi.PushSecret{}
	syncState := esapi.SyncedPushSecretsMap{"SecretStore/example": {}}

	r.markAsFailed(errFailedSetSecret, errors.New("provider returned "+pushSecretSentinel), ps, syncState)

	cond := GetPushSecretCondition(ps.Status.Conditions, esapi.PushSecretReady)
	if cond == nil {
		t.Fatal("no Ready condition was set")
	}
	if strings.Contains(cond.Message, pushSecretSentinel) {
		t.Fatalf("condition message leaked provider text: %q", cond.Message)
	}
	if cond.Message != errFailedSetSecret {
		t.Errorf("condition message = %q, want %q", cond.Message, errFailedSetSecret)
	}

	event := <-recorder.Events
	if strings.Contains(event, pushSecretSentinel) {
		t.Fatalf("event leaked provider text: %q", event)
	}
	if !strings.Contains(event, errFailedSetSecret) {
		t.Errorf("event %q lost the generic failure message", event)
	}

	if _, ok := ps.Status.SyncedPushSecrets["SecretStore/example"]; !ok {
		t.Error("failure status did not preserve the synchronized secret state")
	}
}

func TestMarkAsFailedPublishesExplicitlySafeErrors(t *testing.T) {
	recorder := record.NewFakeRecorder(1)
	r := &Reconciler{recorder: recorder}
	ps := &esapi.PushSecret{}

	r.markAsFailed("invalid PushSecret", ctrlutil.Safe(errors.New("missing store reference")), ps, nil)

	cond := GetPushSecretCondition(ps.Status.Conditions, esapi.PushSecretReady)
	if cond == nil {
		t.Fatal("no Ready condition was set")
	}
	if !strings.Contains(cond.Message, "missing store reference") {
		t.Errorf("condition message %q dropped the safe detail", cond.Message)
	}

	event := <-recorder.Events
	if !strings.Contains(event, "missing store reference") {
		t.Errorf("event %q dropped the safe detail", event)
	}
}
