//Copyright External Secrets Inc. All Rights Reserved

package delinea

import (
	"github.com/DelineaXPM/dsv-sdk-go/v2/vault"
)

// secretAPI represents the subset of the Delinea DevOps Secrets Vault API
// which is supported by dsv-sdk-go/v2.
// See https://dsv.secretsvaultcloud.com/api for full API documentation.
type secretAPI interface {
	Secret(path string) (*vault.Secret, error)
}
