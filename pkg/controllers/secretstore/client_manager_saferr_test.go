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

package secretstore

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	ctrlutil "github.com/external-secrets/external-secrets/pkg/controllers/util"
)

// providerLeak stands in for a provider error carrying secret material.
const providerLeak = "auth failed for token AKIAIOSFODNN7EXAMPLE"

func newSafeErrManager(t *testing.T) (*Manager, client.Client) {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))

	kube := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	return &Manager{
		log:       logr.Discard(),
		client:    kube,
		clientMap: make(map[clientKey]*clientVal),
	}, kube
}

// A store naming no provider fails before any provider code runs, so the reason
// is reportable and the message may be surfaced in the store status.
func TestGetFromStoreProviderResolutionIsSafe(t *testing.T) {
	mgr, _ := newSafeErrManager(t)

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "no-provider", Namespace: "default"},
		Spec:       esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{}},
	}

	_, err := mgr.GetFromStore(context.Background(), store, "default")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProviderResolution), "want ErrProviderResolution, got %v", err)
	assert.Contains(t, ctrlutil.SafeMessage(err), "could not resolve store provider")
}

// A client constructor failure is provider code, so nothing may be published.
func TestGetFromStoreClientErrorIsNotSafe(t *testing.T) {
	mgr, _ := newSafeErrManager(t)

	fakeProvider := &WrapProvider{
		newClientFunc: func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error) {
			return nil, errors.New(providerLeak)
		},
	}
	esv1.ForceRegister(fakeProvider, &esv1.SecretStoreProvider{
		AWS: &esv1.AWSProvider{},
	}, esv1.MaintenanceStatusMaintained)

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-store", Namespace: "default"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{AWS: &esv1.AWSProvider{}},
		},
	}

	_, err := mgr.GetFromStore(context.Background(), store, "default")
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrProviderResolution))
	assert.Empty(t, ctrlutil.SafeMessage(err), "provider error must not be publishable")
}
