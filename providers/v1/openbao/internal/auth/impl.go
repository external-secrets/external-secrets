package auth

import (
	"github.com/openbao/openbao/api/auth/approle/v2"
	"github.com/openbao/openbao/api/auth/userpass/v2"
	"github.com/openbao/openbao/api/v2"
)

type authMethodFactory struct{}

func (authMethodFactory) AppRole(id, secret, mount string) (api.AuthMethod, error) {
	return approle.NewAppRoleAuth(id, &approle.SecretID{
		FromString: secret,
	}, approle.WithMountPath(mount))
}

func (authMethodFactory) UserPass(username, password, mount string) (api.AuthMethod, error) {
	return userpass.NewUserpassAuth(username, &userpass.Password{
		FromString: password,
	}, userpass.WithMountPath(mount))
}

var DefaultAuthMethodFactory AuthMethodFactory = authMethodFactory{}
