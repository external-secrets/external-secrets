//Copyright External Secrets Inc. All Rights Reserved

package v1

// A reference to a specific 'key' within a Secret resource,
// In some instances, `key` is a required field.
type SecretKeySelector struct {
	// The name of the Secret resource being referred to.
	Name string `json:"name,omitempty"`
	// Namespace of the resource being referred to. Ignored if referent is not cluster-scoped. cluster-scoped defaults
	// to the namespace of the referent.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
	// The key of the entry in the Secret resource's `data` field to be used. Some instances of this field may be
	// defaulted, in others it may be required.
	// +optional
	Key string `json:"key,omitempty"`
}

// A reference to a ServiceAccount resource.
type ServiceAccountSelector struct {
	// The name of the ServiceAccount resource being referred to.
	Name string `json:"name"`
	// Namespace of the resource being referred to. Ignored if referent is not cluster-scoped. cluster-scoped defaults
	// to the namespace of the referent.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
	// Audience specifies the `aud` claim for the service account token
	// If the service account uses a well-known annotation for e.g. IRSA or GCP Workload Identity
	// then this audiences will be appended to the list
	// +optional
	Audiences []string `json:"audiences,omitempty"`
}
