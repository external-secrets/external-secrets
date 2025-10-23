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

package main

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	fakev2alpha1 "github.com/external-secrets/external-secrets/apis/provider/fake/v2alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

// GetSpecMapper returns the spec mapper function for the Fake provider.
// This function converts v2 ProviderReference to v1 SecretStoreSpec.
func GetSpecMapper(kubeClient client.Client) func(*pb.ProviderReference) (*v1.SecretStoreSpec, error) {
	return func(ref *pb.ProviderReference) (*v1.SecretStoreSpec, error) {
		var fakeProvider fakev2alpha1.Fake
		err := kubeClient.Get(context.Background(), client.ObjectKey{
			Namespace: ref.Namespace,
			Name:      ref.Name,
		}, &fakeProvider)
		if err != nil {
			return nil, err
		}
		return &v1.SecretStoreSpec{
			Provider: &v1.SecretStoreProvider{
				Fake: &fakeProvider.Spec,
			},
		}, nil
	}
}

