//Copyright External Secrets Inc. All Rights Reserved

package secretserver

import (
	"github.com/DelineaXPM/tss-sdk-go/v2/server"
)

// secretAPI represents the subset of the Secret Server API
// which is supported by tss-sdk-go/v2.
type secretAPI interface {
	Secret(id int) (*server.Secret, error)
	Secrets(searchText, field string) ([]server.Secret, error)
}
