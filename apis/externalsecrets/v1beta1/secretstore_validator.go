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

package v1beta1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.CustomValidator = &GenericStoreValidator{}

const (
	errInvalidStore = "invalid store"
)

type GenericStoreValidator struct{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	st, ok := obj.(GenericStore)
	if !ok {
		return fmt.Errorf(errInvalidStore)
	}
	return validateStore(st)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	st, ok := newObj.(GenericStore)
	if !ok {
		return fmt.Errorf(errInvalidStore)
	}
	return validateStore(st)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func validateStore(store GenericStore) error {
	provider, err := GetProvider(store)
	if err != nil {
		return err
	}
	return provider.ValidateStore(store)
}
