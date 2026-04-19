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
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
)

const indexESV2StoreRefField = ".spec.v2StoreRefs"

func v2StoreRefIndexKey(kind, namespace, name string) string {
	if kind == esv1.ClusterProviderStoreKindStr {
		return fmt.Sprintf("%s/%s", kind, name)
	}
	return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
}

func indexExternalSecretV2StoreRefs(obj client.Object) []string {
	es, ok := obj.(*esv1.ExternalSecret)
	if !ok {
		return nil
	}

	keys := make(map[string]struct{})
	addExternalSecretV2StoreRefIndexKey(keys, es.Namespace, es.Spec.SecretStoreRef)

	for i := range es.Spec.Data {
		if es.Spec.Data[i].SourceRef == nil {
			continue
		}
		addExternalSecretV2StoreRefIndexKey(keys, es.Namespace, es.Spec.Data[i].SourceRef.SecretStoreRef)
	}

	for i := range es.Spec.DataFrom {
		if es.Spec.DataFrom[i].SourceRef == nil || es.Spec.DataFrom[i].SourceRef.SecretStoreRef == nil {
			continue
		}
		addExternalSecretV2StoreRefIndexKey(keys, es.Namespace, *es.Spec.DataFrom[i].SourceRef.SecretStoreRef)
	}

	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	return out
}

func addExternalSecretV2StoreRefIndexKey(keys map[string]struct{}, namespace string, ref esv1.SecretStoreRef) {
	switch ref.Kind {
	case esv1.ProviderStoreKindStr:
		keys[v2StoreRefIndexKey(esv1.ProviderStoreKindStr, namespace, ref.Name)] = struct{}{}
	case esv1.ClusterProviderStoreKindStr:
		keys[v2StoreRefIndexKey(esv1.ClusterProviderStoreKindStr, "", ref.Name)] = struct{}{}
	}
}

func (r *Reconciler) findExternalSecretsForV2Store(ctx context.Context, obj client.Object) []reconcile.Request {
	var (
		key         string
		listOptions []client.ListOption
	)

	switch store := obj.(type) {
	case *esv2alpha1.ProviderStore:
		key = v2StoreRefIndexKey(esv1.ProviderStoreKindStr, store.Namespace, store.Name)
		listOptions = append(listOptions, client.InNamespace(store.Namespace))
	case *esv2alpha1.ClusterProviderStore:
		key = v2StoreRefIndexKey(esv1.ClusterProviderStoreKindStr, "", store.Name)
	default:
		return nil
	}

	listOptions = append(listOptions, client.MatchingFields{indexESV2StoreRefField: key})

	var externalSecrets esv1.ExternalSecretList
	if err := r.List(ctx, &externalSecrets, listOptions...); err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0, len(externalSecrets.Items))
	for i := range externalSecrets.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&externalSecrets.Items[i]),
		})
	}
	return requests
}
