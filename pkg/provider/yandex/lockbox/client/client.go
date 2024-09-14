//Copyright External Secrets Inc. All Rights Reserved

package client

import (
	"context"

	api "github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
)

// Requests the payload of the given secret from Lockbox.
type LockboxClient interface {
	GetPayloadEntries(ctx context.Context, iamToken, secretID, versionID string) ([]*api.Payload_Entry, error)
}
