//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "external-secrets.io"
	Version = "v1beta1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
)

// ExternalSecret type metadata.
var (
	ExtSecretKind             = reflect.TypeOf(ExternalSecret{}).Name()
	ExtSecretGroupKind        = schema.GroupKind{Group: Group, Kind: ExtSecretKind}.String()
	ExtSecretKindAPIVersion   = ExtSecretKind + "." + SchemeGroupVersion.String()
	ExtSecretGroupVersionKind = SchemeGroupVersion.WithKind(ExtSecretKind)
)

// ClusterExternalSecret type metadata.
var (
	ClusterExtSecretKind             = reflect.TypeOf(ClusterExternalSecret{}).Name()
	ClusterExtSecretGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterExtSecretKind}.String()
	ClusterExtSecretKindAPIVersion   = ClusterExtSecretKind + "." + SchemeGroupVersion.String()
	ClusterExtSecretGroupVersionKind = SchemeGroupVersion.WithKind(ClusterExtSecretKind)
)

// SecretStore type metadata.
var (
	SecretStoreKind             = reflect.TypeOf(SecretStore{}).Name()
	SecretStoreGroupKind        = schema.GroupKind{Group: Group, Kind: SecretStoreKind}.String()
	SecretStoreKindAPIVersion   = SecretStoreKind + "." + SchemeGroupVersion.String()
	SecretStoreGroupVersionKind = SchemeGroupVersion.WithKind(SecretStoreKind)
)

// ClusterSecretStore type metadata.
var (
	ClusterSecretStoreKind             = reflect.TypeOf(ClusterSecretStore{}).Name()
	ClusterSecretStoreGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterSecretStoreKind}.String()
	ClusterSecretStoreKindAPIVersion   = ClusterSecretStoreKind + "." + SchemeGroupVersion.String()
	ClusterSecretStoreGroupVersionKind = SchemeGroupVersion.WithKind(ClusterSecretStoreKind)
)

func init() {
	SchemeBuilder.Register(&ExternalSecret{}, &ExternalSecretList{})
	SchemeBuilder.Register(&ClusterExternalSecret{}, &ClusterExternalSecretList{})
	SchemeBuilder.Register(&SecretStore{}, &SecretStoreList{})
	SchemeBuilder.Register(&ClusterSecretStore{}, &ClusterSecretStoreList{})
}
