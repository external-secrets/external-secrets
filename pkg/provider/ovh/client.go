package ovh

import (
	"context"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type client struct{}

var _ esv1.SecretsClient = &client{}

func (c *client) Close(ctx context.Context) error {
	return nil
}
