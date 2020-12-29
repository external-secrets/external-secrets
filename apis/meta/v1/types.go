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

package v1

// A reference to an object in the same namespace as the referent.
type LocalObjectReference struct {
	// Name of the resource being referred to.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name"`
}

// A reference to a specific 'key' within a Secret resource,
// In some instances, `key` is a required field.
type SecretKeySelector struct {
	// The name of the Secret resource being referred to.
	LocalObjectReference `json:",inline"`
	// Namespace of the resource being referred to. Ignored if referent is not cluster-scoped. cluster-scoped defaults
	// to the namespace of the referent.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
	// The key of the entry in the Secret resource's `data` field to be used. Some instances of this field may be
	// defaulted, in others it may be required.
	// +optional
	Key string `json:"key,omitempty"`
}
