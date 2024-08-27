package previder

import (
	"errors"
	"github.com/previder/vault-cli/pkg"
	"github.com/previder/vault-cli/pkg/model"
)

type PreviderVaultFakeClient struct {
	pkg.PreviderVaultClient
}

var (
	secrets = map[string]string{"secret1": "secret1content", "secret2": "secret2content"}
)

func (v *PreviderVaultFakeClient) DecryptSecret(id string) (*model.SecretDecrypt, error) {
	for k, v := range secrets {
		if k == id {
			return &model.SecretDecrypt{Secret: v}, nil
		}
	}
	return nil, errors.New("404 not found")
}

func (v *PreviderVaultFakeClient) GetSecrets() ([]model.Secret, error) {
	var secretList []model.Secret
	for k, _ := range secrets {
		secretList = append(secretList, model.Secret{Description: k})
	}
	return secretList, nil
}
