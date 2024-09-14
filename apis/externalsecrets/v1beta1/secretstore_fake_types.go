//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

// FakeProvider configures a fake provider that returns static values.
type FakeProvider struct {
	Data []FakeProviderData `json:"data"`
}

type FakeProviderData struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
	// Deprecated: ValueMap is deprecated and is intended to be removed in the future, use the `value` field instead.
	ValueMap map[string]string `json:"valueMap,omitempty"`
	Version  string            `json:"version,omitempty"`
}
