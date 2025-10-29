package ovh

import esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

func (c *client) Validate() (esv1.ValidationResult, error) {
	return 1, nil
}
