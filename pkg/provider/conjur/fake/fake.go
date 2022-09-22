package fake

import (
	"errors"
)

type ConjurMockClient struct {
}

func (mc *ConjurMockClient) RetrieveSecret(secret string) (result []byte, err error) {
	if secret == "error" {
		err = errors.New("error")
		return nil, err
	}
	return []byte("secret"), nil
}
