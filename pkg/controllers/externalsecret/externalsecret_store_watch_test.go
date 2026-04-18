package externalsecret

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
)

func TestIndexExternalSecretV2StoreRefs(t *testing.T) {
	es := &esv1.ExternalSecret{}
	es.Namespace = "team-a"
	es.Spec.SecretStoreRef = esv1.SecretStoreRef{
		Name: "namespaced-store",
		Kind: esv1.ProviderStoreKindStr,
	}
	es.Spec.Data = []esv1.ExternalSecretData{
		{},
		{
			SourceRef: &esv1.StoreSourceRef{
				SecretStoreRef: esv1.SecretStoreRef{
					Name: "cluster-store",
					Kind: esv1.ClusterProviderStoreKindStr,
				},
			},
		},
	}
	es.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{
		{
			SourceRef: &esv1.StoreGeneratorSourceRef{
				SecretStoreRef: &esv1.SecretStoreRef{
					Name: "secondary-store",
					Kind: esv1.ProviderStoreKindStr,
				},
			},
		},
		{
			SourceRef: &esv1.StoreGeneratorSourceRef{
				SecretStoreRef: &esv1.SecretStoreRef{
					Name: "ignored-v1-store",
					Kind: esv1.SecretStoreKind,
				},
			},
		},
	}

	got := indexExternalSecretV2StoreRefs(es)
	want := map[string]struct{}{
		v2StoreRefIndexKey(esv1.ProviderStoreKindStr, "team-a", "namespaced-store"): {},
		v2StoreRefIndexKey(esv1.ProviderStoreKindStr, "team-a", "secondary-store"):  {},
		v2StoreRefIndexKey(esv1.ClusterProviderStoreKindStr, "", "cluster-store"):    {},
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d index keys, got %d: %#v", len(want), len(got), got)
	}
	for _, key := range got {
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected index key %q in %#v", key, got)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing expected index keys: %#v", want)
	}
}

func TestFindExternalSecretsForV2Store(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv2alpha1.AddToScheme(scheme))

	matching := &esv1.ExternalSecret{}
	matching.Namespace = "team-a"
	matching.Name = "matching"
	matching.Spec.SecretStoreRef = esv1.SecretStoreRef{
		Name: "aws-prod",
		Kind: esv1.ProviderStoreKindStr,
	}

	nonMatching := &esv1.ExternalSecret{}
	nonMatching.Namespace = "team-a"
	nonMatching.Name = "non-matching"
	nonMatching.Spec.SecretStoreRef = esv1.SecretStoreRef{
		Name: "aws-other",
		Kind: esv1.ProviderStoreKindStr,
	}

	cl := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(matching, nonMatching).
		WithIndex(&esv1.ExternalSecret{}, indexESV2StoreRefField, func(obj client.Object) []string {
			return indexExternalSecretV2StoreRefs(obj)
		}).
		Build()

	r := &Reconciler{Client: cl}
	store := &esv2alpha1.ProviderStore{}
	store.Namespace = "team-a"
	store.Name = "aws-prod"

	got := r.findExternalSecretsForV2Store(context.Background(), store)
	want := []reconcile.Request{{
		NamespacedName: client.ObjectKeyFromObject(matching),
	}}

	if len(got) != len(want) {
		t.Fatalf("expected %d requests, got %d: %#v", len(want), len(got), got)
	}
	if got[0] != want[0] {
		t.Fatalf("expected requests %#v, got %#v", want, got)
	}
}

func TestFindExternalSecretsForClusterProviderStore(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv2alpha1.AddToScheme(scheme))

	matching := &esv1.ExternalSecret{}
	matching.Namespace = "team-a"
	matching.Name = "matching"
	matching.Spec.Data = []esv1.ExternalSecretData{
		{
			SourceRef: &esv1.StoreSourceRef{
				SecretStoreRef: esv1.SecretStoreRef{
					Name: "shared",
					Kind: esv1.ClusterProviderStoreKindStr,
				},
			},
		},
	}

	cl := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(matching).
		WithIndex(&esv1.ExternalSecret{}, indexESV2StoreRefField, func(obj client.Object) []string {
			return indexExternalSecretV2StoreRefs(obj)
		}).
		Build()

	r := &Reconciler{Client: cl}
	store := &esv2alpha1.ClusterProviderStore{}
	store.Name = "shared"

	got := r.findExternalSecretsForV2Store(context.Background(), store)
	if len(got) != 1 || got[0].NamespacedName.Name != matching.Name || got[0].NamespacedName.Namespace != matching.Namespace {
		t.Fatalf("unexpected requests: %#v", got)
	}
}
