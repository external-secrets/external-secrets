//Copyright External Secrets Inc. All Rights Reserved

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
	AddToScheme   = SchemeBuilder.AddToScheme
)

// ExternalSecret type metadata.
var (
	ExtSecretKind             = reflect.TypeOf(ExternalSecret{}).Name()
	ExtSecretGroupKind        = schema.GroupKind{Group: Group, Kind: ExtSecretKind}.String()
	ExtSecretKindAPIVersion   = ExtSecretKind + "." + SchemeGroupVersion.String()
	ExtSecretGroupVersionKind = SchemeGroupVersion.WithKind(ExtSecretKind)
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

var (
	PushSecretKind             = reflect.TypeOf(PushSecret{}).Name()
	PushSecretGroupKind        = schema.GroupKind{Group: Group, Kind: PushSecretKind}.String()
	PushSecretKindAPIVersion   = PushSecretKind + "." + SchemeGroupVersion.String()
	PushSecretGroupVersionKind = SchemeGroupVersion.WithKind(PushSecretKind)
)

func init() {
	SchemeBuilder.Register(&ExternalSecret{}, &ExternalSecretList{})
	SchemeBuilder.Register(&SecretStore{}, &SecretStoreList{})
	SchemeBuilder.Register(&ClusterSecretStore{}, &ClusterSecretStoreList{})
	SchemeBuilder.Register(&PushSecret{}, &PushSecretList{})
}
