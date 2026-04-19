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
	"encoding/json"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

func buildCompatibilityStore(store esv1.GenericStore) (*pb.CompatibilityStore, error) {
	specJSON, err := json.Marshal(store.GetSpec())
	if err != nil {
		return nil, err
	}

	return &pb.CompatibilityStore{
		StoreName:       store.GetName(),
		StoreNamespace:  store.GetNamespace(),
		StoreKind:       store.GetKind(),
		StoreUid:        string(store.GetUID()),
		StoreGeneration: store.GetGeneration(),
		StoreSpecJson:   specJSON,
	}, nil
}
