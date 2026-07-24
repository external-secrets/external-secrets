/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package auth

import (
	"github.com/openbao/openbao/api/auth/approle/v2"
	"github.com/openbao/openbao/api/auth/kubernetes/v2"
	"github.com/openbao/openbao/api/auth/userpass/v2"
	"github.com/openbao/openbao/api/v2"
)

type authMethodFactory struct{}

// AppRole implements [Factory].
func (authMethodFactory) AppRole(id, secret, mount string) (api.AuthMethod, error) {
	return approle.NewAppRoleAuth(id, &approle.SecretID{
		FromString: secret,
	}, approle.WithMountPath(mount))
}

// UserPass implements [Factory].
func (authMethodFactory) UserPass(username, password, mount string) (api.AuthMethod, error) {
	return userpass.NewUserpassAuth(username, &userpass.Password{
		FromString: password,
	}, userpass.WithMountPath(mount))
}

// Kubernetes implements [Factory].
func (authMethodFactory) Kubernetes(role, jwt, mount string) (api.AuthMethod, error) {
	return kubernetes.NewKubernetesAuth(role, kubernetes.WithServiceAccountToken(jwt), kubernetes.WithMountPath(mount))
}

// DefaultAuthMethodFactory implements [Factory].
var DefaultAuthMethodFactory Factory = authMethodFactory{}
