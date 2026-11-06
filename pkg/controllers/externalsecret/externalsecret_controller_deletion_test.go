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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const (
	deletionNamespace  = "ns"
	deletionESName     = "owner-es"
	deletionTargetName = "the-target"
	deletionStoreName  = "test-store"
)

// esControllerRef builds the controller owner reference an ExternalSecret writes. apiVersion
// and uid are parameters because the rule deliberately ignores both.
func esControllerRef(name, apiVersion string, uid types.UID) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: apiVersion, Kind: esv1.ExtSecretKind,
		Name: name, UID: uid, Controller: new(true),
	}
}

// ownershipCase is shared by the predicate test and the tests that drive the real deletion
// callers, so both agree on what each shape of owner reference means.
type ownershipCase struct {
	name       string
	owners     []metav1.OwnerReference
	authorized bool
}

func ownershipCases() []ownershipCase {
	currentAPIVersion := esv1.SchemeGroupVersion.String()
	return []ownershipCase{
		{
			name:       "this ExternalSecret",
			owners:     []metav1.OwnerReference{esControllerRef(deletionESName, currentAPIVersion, "current-uid")},
			authorized: true,
		},
		{
			// the ExternalSecret was deleted and recreated: same name, new UID
			name:       "this ExternalSecret under an older UID",
			owners:     []metav1.OwnerReference{esControllerRef(deletionESName, currentAPIVersion, "stale-uid")},
			authorized: true,
		},
		{
			name:       "this ExternalSecret through another served API version",
			owners:     []metav1.OwnerReference{esControllerRef(deletionESName, "external-secrets.io/v1beta1", "current-uid")},
			authorized: true,
		},
		{
			name:       "a different ExternalSecret",
			owners:     []metav1.OwnerReference{esControllerRef("other-es", currentAPIVersion, "other-uid")},
			authorized: false,
		},
		{
			name:       "an ExternalSecret kind in a different group",
			owners:     []metav1.OwnerReference{esControllerRef(deletionESName, "other.io/v1", "current-uid")},
			authorized: false,
		},
		{
			name: "a controller of another kind",
			owners: []metav1.OwnerReference{{
				APIVersion: "apps/v1", Kind: "Deployment",
				Name: deletionESName, UID: "dep-uid", Controller: new(true),
			}},
			authorized: false,
		},
		{
			name:       "no owner references at all",
			owners:     nil,
			authorized: false,
		},
		{
			name: "a non-controller reference to this ExternalSecret",
			owners: []metav1.OwnerReference{{
				APIVersion: currentAPIVersion, Kind: esv1.ExtSecretKind,
				Name: deletionESName, UID: "current-uid",
			}},
			authorized: false,
		},
	}
}

func deletionES(policy esv1.ExternalSecretDeletionPolicy, manifest *esv1.ManifestReference) *esv1.ExternalSecret {
	return &esv1.ExternalSecret{
		Name: deletionESName, Namespace: deletionNamespace, UID: "current-uid",
		Spec: esv1.ExternalSecretSpec{
			RefreshInterval: &metav1.Duration{Duration: time.Hour},
			SecretStoreRef:  esv1.SecretStoreRef{Name: deletionStoreName, Kind: esv1.SecretStoreKind},
			Data: []esv1.ExternalSecretData{{
				SecretKey: "key",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "remote-key"},
			}},
			Target: esv1.ExternalSecretTarget{
				Name:           deletionTargetName,
				CreationPolicy: esv1.CreatePolicyOwner,
				DeletionPolicy: policy,
				Manifest:       manifest,
			},
		},
	}
}

func deletionStore() *esv1.SecretStore {
	return &esv1.SecretStore{
		Name: deletionStoreName, Namespace: deletionNamespace,
		Spec: esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{
			AWS: &esv1.AWSProvider{Service: esv1.AWSServiceSecretsManager},
		}},
	}
}

func configMapTarget(owners []metav1.OwnerReference) *unstructured.Unstructured {
	target := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{
			"name": deletionTargetName, "namespace": deletionNamespace, "uid": "target-uid",
		},
		"data": map[string]any{"key": "value"},
	}}
	target.SetOwnerReferences(owners)
	return target
}

func secretTarget(owners []metav1.OwnerReference) *v1.Secret {
	return &v1.Secret{
		Name: deletionTargetName, Namespace: deletionNamespace, UID: "target-uid",
		Labels:          map[string]string{esv1.LabelManaged: esv1.LabelManagedValue},
		OwnerReferences: owners,
		Data:            map[string][]byte{"key": []byte("value")},
	}
}

func TestIsControlledByExternalSecret(t *testing.T) {
	es := deletionES(esv1.DeletionPolicyDelete, nil)

	for _, tt := range ownershipCases() {
		t.Run(tt.name, func(t *testing.T) {
			target := secretTarget(tt.owners)
			assert.Equal(t, tt.authorized, isControlledByExternalSecret(es, target))
		})
	}
}

// TestDeleteGenericResourceAuthorization drives the real deletion caller rather than the
// predicate, so it also proves the target is fetched: the authorizing owner reference only
// exists on the live object, never on one built from the declared name.
func TestDeleteGenericResourceAuthorization(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	manifest := &esv1.ManifestReference{APIVersion: "v1", Kind: "ConfigMap"}

	for _, tt := range ownershipCases() {
		t.Run(tt.name, func(t *testing.T) {
			es := deletionES(esv1.DeletionPolicyDelete, manifest)
			target := configMapTarget(tt.owners)
			r := &Reconciler{
				Client:   fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(target).Build(),
				Scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(10),

				AllowGenericTargets: true,
			}

			err := r.deleteGenericResource(context.Background(), logr.Discard(), es)

			live := &unstructured.Unstructured{}
			live.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
			getErr := r.Get(context.Background(),
				types.NamespacedName{Name: deletionTargetName, Namespace: deletionNamespace}, live)

			if tt.authorized {
				require.NoError(t, err)
				assert.True(t, apierrors.IsNotFound(getErr), "authorized target should be deleted")
				return
			}
			require.ErrorIs(t, err, ErrTargetNotControlled)
			require.NoError(t, getErr, "unauthorized target must be preserved")
			assert.Equal(t, tt.owners, live.GetOwnerReferences(), "ownership must not be rewritten")
		})
	}
}

func TestDeleteGenericResourceMissingAndFailing(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	manifest := &esv1.ManifestReference{APIVersion: "v1", Kind: "ConfigMap"}
	es := deletionES(esv1.DeletionPolicyDelete, manifest)

	t.Run("missing target is a no-op", func(t *testing.T) {
		r := &Reconciler{
			Client:   fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
			Scheme:   scheme.Scheme,
			recorder: record.NewFakeRecorder(10),

			AllowGenericTargets: true,
		}
		require.NoError(t, r.deleteGenericResource(context.Background(), logr.Discard(), es))
	})

	t.Run("a real read failure is still an error", func(t *testing.T) {
		boom := errors.New("api is down")
		r := &Reconciler{
			Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
						return boom
					},
				}).Build(),
			Scheme:   scheme.Scheme,
			recorder: record.NewFakeRecorder(10),

			AllowGenericTargets: true,
		}
		require.ErrorContains(t, r.deleteGenericResource(context.Background(), logr.Discard(), es), "api is down")
	})
}

// TestCleanupManagedSecretsAuthorization covers ExternalSecret finalization. An unauthorized
// target must be left alone without blocking finalization, so the refusal is logged and nil
// is returned rather than an error.
func TestCleanupManagedSecretsAuthorization(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)

	for _, tt := range ownershipCases() {
		t.Run("Secret/"+tt.name, func(t *testing.T) {
			es := deletionES(esv1.DeletionPolicyDelete, nil)
			target := secretTarget(tt.owners)
			r := &Reconciler{
				Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(target).Build(),
				Scheme: scheme.Scheme, recorder: record.NewFakeRecorder(10),
			}

			require.NoError(t, r.cleanupManagedSecrets(context.Background(), logr.Discard(), es))

			live := &v1.Secret{}
			getErr := r.Get(context.Background(),
				types.NamespacedName{Name: deletionTargetName, Namespace: deletionNamespace}, live)
			if tt.authorized {
				assert.True(t, apierrors.IsNotFound(getErr), "authorized Secret should be deleted")
				return
			}
			require.NoError(t, getErr, "unauthorized Secret must be preserved")
			assert.Equal(t, map[string][]byte{"key": []byte("value")}, live.Data)
		})

		t.Run("manifest/"+tt.name, func(t *testing.T) {
			es := deletionES(esv1.DeletionPolicyDelete, &esv1.ManifestReference{APIVersion: "v1", Kind: "ConfigMap"})
			target := configMapTarget(tt.owners)
			r := &Reconciler{
				Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(target).Build(),
				Scheme: scheme.Scheme, recorder: record.NewFakeRecorder(10),

				AllowGenericTargets: true,
			}

			require.NoError(t, r.cleanupManagedSecrets(context.Background(), logr.Discard(), es))

			live := &unstructured.Unstructured{}
			live.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
			getErr := r.Get(context.Background(),
				types.NamespacedName{Name: deletionTargetName, Namespace: deletionNamespace}, live)
			if tt.authorized {
				assert.True(t, apierrors.IsNotFound(getErr), "authorized target should be deleted")
				return
			}
			require.NoError(t, getErr, "unauthorized target must be preserved")
		})
	}
}

// TestCleanupManagedSecretsPropagatesAPIFailure separates a refusal from a real failure: only
// the latter may be returned, because a returned error retries finalization.
func TestCleanupManagedSecretsPropagatesAPIFailure(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	boom := errors.New("api is down")
	r := &Reconciler{
		Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return boom
				},
			}).Build(),
		Scheme: scheme.Scheme, recorder: record.NewFakeRecorder(10),
	}

	err := r.cleanupManagedSecrets(context.Background(), logr.Discard(), deletionES(esv1.DeletionPolicyDelete, nil))
	require.ErrorContains(t, err, "api is down")
}

// TestProviderEmptyDeletionAuthorization covers the other deletion trigger: the provider
// returned no data and deletionPolicy=Delete. Both target lanes must check ownership before
// deleting, and neither may adopt the target first to make itself the owner.
func TestProviderEmptyDeletionAuthorization(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeProvider.Reset()
	defer fakeProvider.Reset()
	fakeProvider.WithGetSecret(nil, esv1.NoSecretErr)

	owned := []metav1.OwnerReference{esControllerRef(deletionESName, esv1.SchemeGroupVersion.String(), "stale-uid")}
	foreign := []metav1.OwnerReference{esControllerRef("other-es", esv1.SchemeGroupVersion.String(), "other-uid")}

	// Any adoption write before the delete would mean ownership was manufactured rather than
	// verified, so Update and Patch on the target are made to fail loudly.
	refuseTargetWrites := interceptor.Funcs{
		Update: func(_ context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if obj.GetName() == deletionTargetName {
				return errors.New("unexpected adoption Update on the target")
			}
			return c.Update(context.Background(), obj, opts...)
		},
		Patch: func(_ context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			if obj.GetName() == deletionTargetName {
				return errors.New("unexpected adoption Patch on the target")
			}
			return c.Patch(context.Background(), obj, patch, opts...)
		},
	}

	t.Run("Secret lane", func(t *testing.T) {
		for _, tt := range []struct {
			name     string
			owners   []metav1.OwnerReference
			wantKept bool
		}{
			{"owned under a stale ES UID is deleted", owned, false},
			{"owned by another ExternalSecret is kept", foreign, true},
			{"unowned is kept", nil, true},
		} {
			t.Run(tt.name, func(t *testing.T) {
				es := deletionES(esv1.DeletionPolicyDelete, nil)
				fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(es, deletionStore(), secretTarget(tt.owners)).
					WithStatusSubresource(&esv1.ExternalSecret{}).
					WithInterceptorFuncs(refuseTargetWrites).Build()
				r := &Reconciler{
					Client: fakeClient, SecretClient: fakeClient, Scheme: scheme.Scheme,
					Log: logr.Discard(), recorder: record.NewFakeRecorder(100),
					RequeueInterval: time.Hour,
				}

				_, err := r.Reconcile(context.Background(), ctrl.Request{
					Name: deletionESName, Namespace: deletionNamespace,
				})
				require.NoError(t, err)

				live := &v1.Secret{}
				getErr := r.Get(context.Background(),
					types.NamespacedName{Name: deletionTargetName, Namespace: deletionNamespace}, live)
				if tt.wantKept {
					require.NoError(t, getErr, "target must be preserved")
					assert.Equal(t, tt.owners, live.OwnerReferences, "ownership must not be rewritten")
					return
				}
				assert.True(t, apierrors.IsNotFound(getErr), "owned target should be deleted")
			})
		}
	})

	t.Run("manifest lane", func(t *testing.T) {
		for _, tt := range []struct {
			name     string
			owners   []metav1.OwnerReference
			wantKept bool
		}{
			{"owned under a stale ES UID is deleted", owned, false},
			{"owned by another ExternalSecret is kept", foreign, true},
			{"unowned is kept", nil, true},
		} {
			t.Run(tt.name, func(t *testing.T) {
				es := deletionES(esv1.DeletionPolicyDelete, &esv1.ManifestReference{APIVersion: "v1", Kind: "ConfigMap"})
				fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(deletionStore(), configMapTarget(tt.owners)).
					WithInterceptorFuncs(refuseTargetWrites).Build()
				r := &Reconciler{
					Client: fakeClient, Scheme: scheme.Scheme,
					Log: logr.Discard(), recorder: record.NewFakeRecorder(100),
					AllowGenericTargets: true, informerManager: noopInformerManager{},
					RequeueInterval: time.Hour,
				}

				resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(
					map[string]string{"name": es.Name, "namespace": es.Namespace})
				_, err := r.reconcileGenericTarget(context.Background(), es, logr.Discard(), time.Now(),
					resourceLabels, esmetrics.GetCounterVec(esmetrics.SyncCallsErrorKey))
				require.NoError(t, err)

				live := &unstructured.Unstructured{}
				live.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
				getErr := r.Get(context.Background(),
					types.NamespacedName{Name: deletionTargetName, Namespace: deletionNamespace}, live)
				if tt.wantKept {
					require.NoError(t, getErr, "target must be preserved")
					assert.Equal(t, tt.owners, live.GetOwnerReferences(), "ownership must not be rewritten")
					return
				}
				assert.True(t, apierrors.IsNotFound(getErr), "owned target should be deleted")
			})
		}
	})
}

// TestDeletePinsTargetUID asserts every deletion path pins its delete to the UID whose
// ownership it just checked. Whether a real API server honors that pin is proven by the
// envtest spec "refuses to delete a target that was replaced after the ownership check"; what
// this covers is the wiring, which that spec cannot reach for the Secret paths. It matters
// because the fake client ignores client.Preconditions, so a path that silently stopped
// passing one would leave every other test in this file green.
func TestDeletePinsTargetUID(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	owned := []metav1.OwnerReference{
		esControllerRef(deletionESName, esv1.SchemeGroupVersion.String(), "stale-uid"),
	}

	// capturePreconditions records the options production code passes for the target, then
	// lets the delete proceed so the path under test behaves normally.
	capturePreconditions := func(captured **metav1.Preconditions, name string) interceptor.Funcs {
		return interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				if obj.GetName() == name {
					var applied client.DeleteOptions
					for _, opt := range opts {
						opt.ApplyToDelete(&applied)
					}
					*captured = applied.Preconditions
				}
				return c.Delete(ctx, obj, opts...)
			},
		}
	}

	assertPinned := func(t *testing.T, captured *metav1.Preconditions) {
		t.Helper()
		require.NotNil(t, captured, "delete carried no preconditions, so the target UID is unpinned")
		require.NotNil(t, captured.UID, "delete carried preconditions without a UID")
		assert.Equal(t, types.UID("target-uid"), *captured.UID, "pinned to the wrong UID")
	}

	t.Run("manifest provider-empty path", func(t *testing.T) {
		var captured *metav1.Preconditions
		es := deletionES(esv1.DeletionPolicyDelete, &esv1.ManifestReference{APIVersion: "v1", Kind: "ConfigMap"})
		r := &Reconciler{
			Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
				WithObjects(configMapTarget(owned)).
				WithInterceptorFuncs(capturePreconditions(&captured, deletionTargetName)).Build(),
			Scheme: scheme.Scheme, recorder: record.NewFakeRecorder(10),

			AllowGenericTargets: true,
		}

		require.NoError(t, r.deleteGenericResource(context.Background(), logr.Discard(), es))
		assertPinned(t, captured)
	})

	t.Run("Secret finalization cleanup", func(t *testing.T) {
		var captured *metav1.Preconditions
		es := deletionES(esv1.DeletionPolicyDelete, nil)
		r := &Reconciler{
			Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
				WithObjects(secretTarget(owned)).
				WithInterceptorFuncs(capturePreconditions(&captured, deletionTargetName)).Build(),
			Scheme: scheme.Scheme, recorder: record.NewFakeRecorder(10),
		}

		require.NoError(t, r.cleanupManagedSecrets(context.Background(), logr.Discard(), es))
		assertPinned(t, captured)
	})

	t.Run("Secret provider-empty path", func(t *testing.T) {
		fakeProvider.Reset()
		defer fakeProvider.Reset()
		fakeProvider.WithGetSecret(nil, esv1.NoSecretErr)

		var captured *metav1.Preconditions
		es := deletionES(esv1.DeletionPolicyDelete, nil)
		fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
			WithObjects(es, deletionStore(), secretTarget(owned)).
			WithStatusSubresource(&esv1.ExternalSecret{}).
			WithInterceptorFuncs(capturePreconditions(&captured, deletionTargetName)).Build()
		r := &Reconciler{
			Client: fakeClient, SecretClient: fakeClient, Scheme: scheme.Scheme,
			Log: logr.Discard(), recorder: record.NewFakeRecorder(100),
			RequeueInterval: time.Hour,
		}

		_, err := r.Reconcile(context.Background(), ctrl.Request{
			Name: deletionESName, Namespace: deletionNamespace,
		})
		require.NoError(t, err)
		assertPinned(t, captured)
	})
}

// TestDeleteOrphanedSecretsAuthorization covers the third deletion trigger: creationPolicy
// =Owner cleaning up a Secret left behind when spec.target.name changed. The owner label
// selects candidates, but it is an ordinary label anyone able to write a Secret can set, so
// the controller owner reference is what has to authorize the delete.
func TestDeleteOrphanedSecretsAuthorization(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)

	const orphanName = "the-previous-target"
	ownerLabel := esutils.ObjectHash(deletionNamespace + "/" + deletionESName)

	// labeled builds a Secret carrying this ExternalSecret's owner label, which is all the
	// list selector looks at, plus whatever owner references the case is about.
	labeled := func(name string, owners []metav1.OwnerReference) *v1.Secret {
		return &v1.Secret{
			Name: name, Namespace: deletionNamespace, UID: types.UID(name + "-uid"),
			Labels: map[string]string{
				esv1.LabelManaged: esv1.LabelManagedValue,
				esv1.LabelOwner:   ownerLabel,
			},
			OwnerReferences: owners,
			Data:            map[string][]byte{"key": []byte("value")},
		}
	}

	for _, tt := range ownershipCases() {
		t.Run(tt.name, func(t *testing.T) {
			es := deletionES(esv1.DeletionPolicyDelete, nil)
			r := &Reconciler{
				Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(labeled(orphanName, tt.owners), labeled(deletionTargetName, tt.owners)).Build(),
				Scheme: scheme.Scheme, recorder: record.NewFakeRecorder(10),
			}

			require.NoError(t, r.deleteOrphanedSecrets(context.Background(), logr.Discard(), es, deletionTargetName))

			orphan := &v1.Secret{}
			orphanErr := r.Get(context.Background(),
				types.NamespacedName{Name: orphanName, Namespace: deletionNamespace}, orphan)

			// the current target is never an orphan, whatever its ownership
			current := &v1.Secret{}
			require.NoError(t, r.Get(context.Background(),
				types.NamespacedName{Name: deletionTargetName, Namespace: deletionNamespace}, current),
				"the current target must never be deleted as an orphan")

			if tt.authorized {
				assert.True(t, apierrors.IsNotFound(orphanErr), "an owned orphan should be cleaned up")
				return
			}
			require.NoError(t, orphanErr, "a Secret this ExternalSecret does not control must be preserved")
			assert.Equal(t, map[string][]byte{"key": []byte("value")}, orphan.Data)
		})
	}
}

// TestDeleteOrphanedSecretsPinsUID extends the wiring guarantee to the orphan cleanup path,
// which reaches the same shared helper.
func TestDeleteOrphanedSecretsPinsUID(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)

	const orphanName = "the-previous-target"
	var captured *metav1.Preconditions

	orphan := &v1.Secret{
		Name: orphanName, Namespace: deletionNamespace, UID: types.UID(orphanName + "-uid"),
		Labels: map[string]string{
			esv1.LabelOwner: esutils.ObjectHash(deletionNamespace + "/" + deletionESName),
		},
		OwnerReferences: []metav1.OwnerReference{
			esControllerRef(deletionESName, esv1.SchemeGroupVersion.String(), "stale-uid"),
		},
	}

	r := &Reconciler{
		Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(orphan).
			WithInterceptorFuncs(interceptor.Funcs{
				Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if obj.GetName() == orphanName {
						var applied client.DeleteOptions
						for _, opt := range opts {
							opt.ApplyToDelete(&applied)
						}
						captured = applied.Preconditions
					}
					return c.Delete(ctx, obj, opts...)
				},
			}).Build(),
		Scheme: scheme.Scheme, recorder: record.NewFakeRecorder(10),
	}

	es := deletionES(esv1.DeletionPolicyDelete, nil)
	require.NoError(t, r.deleteOrphanedSecrets(context.Background(), logr.Discard(), es, deletionTargetName))

	require.NotNil(t, captured, "orphan delete carried no preconditions, so the UID is unpinned")
	require.NotNil(t, captured.UID, "orphan delete carried preconditions without a UID")
	assert.Equal(t, types.UID(orphanName+"-uid"), *captured.UID)
}
