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

package aws

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type recordingSecretStoreProvider struct {
	created []string
	deleted []string
}

func (p *recordingSecretStoreProvider) CreateSecret(key string, _ framework.SecretEntry) {
	p.created = append(p.created, key)
}

func (p *recordingSecretStoreProvider) DeleteSecret(key string) {
	p.deleted = append(p.deleted, key)
}

func TestVersionedParameterV2RegistersCleanupWithoutDeletingDuringSetup(t *testing.T) {
	fakeProvider := &recordingSecretStoreProvider{}

	f := &framework.Framework{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
		},
	}
	_, tweak := versionedParameterV2(fakeProvider)(f)
	tc := &framework.TestCase{
		ExternalSecret: &esapi.ExternalSecret{},
	}

	tweak(tc)

	if got, want := len(fakeProvider.created), 5; got != want {
		t.Fatalf("expected %d created versions, got %d", want, got)
	}
	if got := len(fakeProvider.deleted); got != 0 {
		t.Fatalf("expected no deletes during setup, got %d", got)
	}
	if tc.Cleanup == nil {
		t.Fatalf("expected cleanup callback to be registered")
	}

	tc.Cleanup()

	if got, want := len(fakeProvider.deleted), 1; got != want {
		t.Fatalf("expected %d delete after cleanup, got %d", want, got)
	}
}
