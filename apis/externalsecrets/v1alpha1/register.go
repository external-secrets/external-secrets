/*
Copyright © 2025 ESO Maintainer Team

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

package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "external-secrets.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

var (
	// PushSecretKind is the kind name used for PushSecret resources.
	PushSecretKind = reflect.TypeOf(PushSecret{}).Name()
	// PushSecretGroupKind is the group/kind used for PushSecret resources.
	PushSecretGroupKind = schema.GroupKind{Group: Group, Kind: PushSecretKind}.String()
	// PushSecretKindAPIVersion is the kind/apiVersion used for PushSecret resources.
	PushSecretKindAPIVersion = PushSecretKind + "." + SchemeGroupVersion.String()
	// PushSecretGroupVersionKind is the GroupVersionKind for PushSecret resources.
	PushSecretGroupVersionKind = SchemeGroupVersion.WithKind(PushSecretKind)
)

var (
	// ClusterPushSecretKind is the kind name used for ClusterPushSecret resources.
	ClusterPushSecretKind = reflect.TypeOf(ClusterPushSecret{}).Name()
	// ClusterPushSecretGroupKind is the group/kind used for ClusterPushSecret resources.
	ClusterPushSecretGroupKind = schema.GroupKind{Group: Group, Kind: ClusterPushSecretKind}.String()
	// ClusterPushSecretKindAPIVersion is the kind/apiVersion used for ClusterPushSecret resources.
	ClusterPushSecretKindAPIVersion = ClusterPushSecretKind + "." + SchemeGroupVersion.String()
	// ClusterPushSecretGroupVersionKind is the GroupVersionKind for ClusterPushSecret resources.
	ClusterPushSecretGroupVersionKind = SchemeGroupVersion.WithKind(ClusterPushSecretKind)
)

func init() {
	SchemeBuilder.Register(&PushSecret{}, &PushSecretList{})
	SchemeBuilder.Register(&ClusterPushSecret{}, &ClusterPushSecretList{})
}
