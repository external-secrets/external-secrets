/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vault

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	errInvalidCredentials     = "invalid vault credentials: %w"
	errInvalidStore           = "invalid store"
	errInvalidStoreSpec       = "invalid store spec"
	errInvalidStoreProv       = "invalid store provider"
	errInvalidVaultProv       = "invalid vault provider"
	errInvalidAppRoleRef      = "invalid Auth.AppRole.RoleRef: %w"
	errInvalidAppRoleSec      = "invalid Auth.AppRole.SecretRef: %w"
	errInvalidClientCert      = "invalid Auth.Cert.ClientCert: %w"
	errInvalidCertSec         = "invalid Auth.Cert.SecretRef: %w"
	errInvalidJwtSec          = "invalid Auth.Jwt.SecretRef: %w"
	errInvalidJwtK8sSA        = "invalid Auth.Jwt.KubernetesServiceAccountToken.ServiceAccountRef: %w"
	errInvalidKubeSA          = "invalid Auth.Kubernetes.ServiceAccountRef: %w"
	errInvalidKubeSec         = "invalid Auth.Kubernetes.SecretRef: %w"
	errInvalidLdapSec         = "invalid Auth.Ldap.SecretRef: %w"
	errInvalidTokenRef        = "invalid Auth.TokenSecretRef: %w"
	errInvalidUserPassSec     = "invalid Auth.UserPass.SecretRef: %w"
	errInvalidClientTLSCert   = "invalid ClientTLS.ClientCert: %w"
	errInvalidClientTLSSecret = "invalid ClientTLS.SecretRef: %w"
	errInvalidClientTLS       = "when provided, both ClientTLS.ClientCert and ClientTLS.SecretRef should be provided"
)

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, fmt.Errorf(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, fmt.Errorf(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return nil, fmt.Errorf(errInvalidStoreProv)
	}
	vaultProvider := spc.Provider.Vault
	if vaultProvider == nil {
		return nil, fmt.Errorf(errInvalidVaultProv)
	}
	if vaultProvider.Auth.AppRole != nil {
		// check SecretRef for valid configuration
		if err := utils.ValidateReferentSecretSelector(store, vaultProvider.Auth.AppRole.SecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidAppRoleSec, err)
		}

		// prefer .auth.appRole.roleId, fallback to .auth.appRole.roleRef, give up after that.
		if vaultProvider.Auth.AppRole.RoleID == "" { // prevents further RoleID tests if .auth.appRole.roleId is given
			if vaultProvider.Auth.AppRole.RoleRef != nil { // check RoleRef for valid configuration
				if err := utils.ValidateReferentSecretSelector(store, *vaultProvider.Auth.AppRole.RoleRef); err != nil {
					return nil, fmt.Errorf(errInvalidAppRoleRef, err)
				}
			} else { // we ran out of ways to get RoleID. return an appropriate error
				return nil, fmt.Errorf(errInvalidAppRoleID)
			}
		}
	}
	if vaultProvider.Auth.Cert != nil {
		if err := utils.ValidateReferentSecretSelector(store, vaultProvider.Auth.Cert.ClientCert); err != nil {
			return nil, fmt.Errorf(errInvalidClientCert, err)
		}
		if err := utils.ValidateReferentSecretSelector(store, vaultProvider.Auth.Cert.SecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidCertSec, err)
		}
	}
	if vaultProvider.Auth.Jwt != nil {
		if vaultProvider.Auth.Jwt.SecretRef != nil {
			if err := utils.ValidateReferentSecretSelector(store, *vaultProvider.Auth.Jwt.SecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidJwtSec, err)
			}
		} else if vaultProvider.Auth.Jwt.KubernetesServiceAccountToken != nil {
			if err := utils.ValidateReferentServiceAccountSelector(store, vaultProvider.Auth.Jwt.KubernetesServiceAccountToken.ServiceAccountRef); err != nil {
				return nil, fmt.Errorf(errInvalidJwtK8sSA, err)
			}
		} else {
			return nil, fmt.Errorf(errJwtNoTokenSource)
		}
	}
	if vaultProvider.Auth.Kubernetes != nil {
		if vaultProvider.Auth.Kubernetes.ServiceAccountRef != nil {
			if err := utils.ValidateReferentServiceAccountSelector(store, *vaultProvider.Auth.Kubernetes.ServiceAccountRef); err != nil {
				return nil, fmt.Errorf(errInvalidKubeSA, err)
			}
		}
		if vaultProvider.Auth.Kubernetes.SecretRef != nil {
			if err := utils.ValidateReferentSecretSelector(store, *vaultProvider.Auth.Kubernetes.SecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidKubeSec, err)
			}
		}
	}
	if vaultProvider.Auth.Ldap != nil {
		if err := utils.ValidateReferentSecretSelector(store, vaultProvider.Auth.Ldap.SecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidLdapSec, err)
		}
	}
	if vaultProvider.Auth.UserPass != nil {
		if err := utils.ValidateReferentSecretSelector(store, vaultProvider.Auth.UserPass.SecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidUserPassSec, err)
		}
	}
	if vaultProvider.Auth.TokenSecretRef != nil {
		if err := utils.ValidateReferentSecretSelector(store, *vaultProvider.Auth.TokenSecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidTokenRef, err)
		}
	}
	if vaultProvider.Auth.Iam != nil {
		if vaultProvider.Auth.Iam.JWTAuth != nil {
			if vaultProvider.Auth.Iam.JWTAuth.ServiceAccountRef != nil {
				if err := utils.ValidateReferentServiceAccountSelector(store, *vaultProvider.Auth.Iam.JWTAuth.ServiceAccountRef); err != nil {
					return nil, fmt.Errorf(errInvalidTokenRef, err)
				}
			}
		}

		if vaultProvider.Auth.Iam.SecretRef != nil {
			if err := utils.ValidateReferentSecretSelector(store, vaultProvider.Auth.Iam.SecretRef.AccessKeyID); err != nil {
				return nil, fmt.Errorf(errInvalidTokenRef, err)
			}
			if err := utils.ValidateReferentSecretSelector(store, vaultProvider.Auth.Iam.SecretRef.SecretAccessKey); err != nil {
				return nil, fmt.Errorf(errInvalidTokenRef, err)
			}
			if vaultProvider.Auth.Iam.SecretRef.SessionToken != nil {
				if err := utils.ValidateReferentSecretSelector(store, *vaultProvider.Auth.Iam.SecretRef.SessionToken); err != nil {
					return nil, fmt.Errorf(errInvalidTokenRef, err)
				}
			}
		}
	}
	if vaultProvider.ClientTLS.CertSecretRef != nil && vaultProvider.ClientTLS.KeySecretRef != nil {
		if err := utils.ValidateReferentSecretSelector(store, *vaultProvider.ClientTLS.CertSecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidClientTLSCert, err)
		}
		if err := utils.ValidateReferentSecretSelector(store, *vaultProvider.ClientTLS.KeySecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidClientTLSSecret, err)
		}
	} else if vaultProvider.ClientTLS.CertSecretRef != nil || vaultProvider.ClientTLS.KeySecretRef != nil {
		return nil, errors.New(errInvalidClientTLS)
	}
	return nil, nil
}

func (c *client) Validate() (esv1beta1.ValidationResult, error) {
	// when using referent namespace we can not validate the token
	// because the namespace is not known yet when Validate() is called
	// from the SecretStore controller.
	if c.storeKind == esv1beta1.ClusterSecretStoreKind && isReferentSpec(c.store) {
		return esv1beta1.ValidationResultUnknown, nil
	}
	_, err := checkToken(context.Background(), c.token)
	if err != nil {
		return esv1beta1.ValidationResultError, fmt.Errorf(errInvalidCredentials, err)
	}
	return esv1beta1.ValidationResultReady, nil
}
