//go:build akeyless || all_providers

package akeyless

import (
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/provider"
)

func init() {
	esv1.Register(NewProvider(), ProviderSpec(), MaintenanceStatus())
	provider.Register("akeyless", Metadata())
}

func Metadata() provider.Metadata {
	return provider.Metadata{
		Stability: provider.StabilityStable,
		Capabilities: []provider.Capability{
			{
				Name: provider.CapabilityGetSecret,
			},
		},
		Comment: "TODO Akeyless metadata",
	}
}
