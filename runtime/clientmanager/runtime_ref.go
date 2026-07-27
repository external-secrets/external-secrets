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

package clientmanager

import (
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

func buildProviderReference(store esv1.GenericStore, sourceNamespace string) (*pb.ProviderReference, error) {
	spec := store.GetSpec()
	if spec == nil || spec.ProviderRef == nil {
		return nil, fmt.Errorf("%s spec.providerRef is required when spec.runtimeRef is set", store.GetKind())
	}

	namespace, err := resolveProviderRefNamespace(store, sourceNamespace)
	if err != nil {
		return nil, err
	}

	return &pb.ProviderReference{
		ApiVersion:   spec.ProviderRef.APIVersion,
		Kind:         spec.ProviderRef.Kind,
		Name:         spec.ProviderRef.Name,
		Namespace:    namespace,
		StoreRefKind: store.GetKind(),
	}, nil
}

func resolveProviderRefNamespace(store esv1.GenericStore, sourceNamespace string) (string, error) {
	ref := store.GetSpec().ProviderRef
	if store.GetKind() == esv1.SecretStoreKind {
		if ref.Namespace == "" {
			return store.GetNamespace(), nil
		}
		return ref.Namespace, nil
	}

	if ref.Namespace != "" {
		return ref.Namespace, nil
	}
	if sourceNamespace == "" {
		return "", fmt.Errorf("%s spec.providerRef.namespace requires a caller namespace", store.GetKind())
	}
	return sourceNamespace, nil
}
