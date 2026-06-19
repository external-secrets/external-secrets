package auth

import (
	"github.com/openbao/openbao/api/v2"
)

type AuthMethodFactory interface {
	UserPass(username, password, mount string) (api.AuthMethod, error)
	AppRole(id, secret, mount string) (api.AuthMethod, error)
}
