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

package openbao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const (
	errInvalidCredentials     = "invalid OpenBao credentials: %w"
	errInvalidStore           = "invalid store"
	errInvalidStoreSpec       = "invalid store spec"
	errInvalidStoreProv       = "invalid store provider"
	errInvalidOpenBaoProv     = "invalid OpenBao provider"
	errInvalidAppRoleRef      = "invalid Auth.AppRole.RoleRef: %w"
	errInvalidAppRoleSec      = "invalid Auth.AppRole.SecretRef: %w"
	errInvalidClientCert      = "invalid Auth.Cert.ClientCert: %w"
	errInvalidCertSec         = "invalid Auth.Cert.SecretRef: %w"
	errInvalidJwtSec          = "invalid Auth.Jwt.SecretRef: %w"
	errInvalidJwtK8sSA        = "invalid Auth.Jwt.ServiceAccountRef: %w"
	errInvalidKubeSA          = "invalid Auth.Kubernetes.ServiceAccountRef: %w"
	errInvalidKubeSec         = "invalid Auth.Kubernetes.SecretRef: %w"
	errInvalidLdapSec         = "invalid Auth.Ldap.SecretRef: %w"
	errInvalidTokenRef        = "invalid Auth.TokenSecretRef: %w"
	errInvalidUserPassSec     = "invalid Auth.UserPass.SecretRef: %w"
	errInvalidClientTLSCert   = "invalid ClientTLS.ClientCert: %w"
	errInvalidClientTLSSecret = "invalid ClientTLS.SecretRef: %w"
	errInvalidClientTLS       = "when provided, both ClientTLS.ClientCert and ClientTLS.SecretRef should be provided"
	errCASNotSupportedInKVv1  = "checkAndSet is not supported with OpenBao KV version v1"
)

// ValidateStore validates the OpenBao provider configuration in the SecretStore.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, errors.New(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return nil, errors.New(errInvalidStoreProv)
	}
	baoProvider := spc.Provider.OpenBao
	if baoProvider == nil {
		return nil, errors.New(errInvalidOpenBaoProv)
	}
	if baoProvider.Auth != nil {
		if baoProvider.Auth.AppRole != nil {
			// check SecretRef for valid configuration
			if err := esutils.ValidateReferentSecretSelector(store, baoProvider.Auth.AppRole.SecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidAppRoleSec, err)
			}

			// prefer .auth.appRole.roleId, fallback to .auth.appRole.roleRef, give up after that.
			if baoProvider.Auth.AppRole.RoleID == "" { // prevents further RoleID tests if .auth.appRole.roleId is given
				if baoProvider.Auth.AppRole.RoleRef != nil { // check RoleRef for valid configuration
					if err := esutils.ValidateReferentSecretSelector(store, *baoProvider.Auth.AppRole.RoleRef); err != nil {
						return nil, fmt.Errorf(errInvalidAppRoleRef, err)
					}
				} else { // we ran out of ways to get RoleID. return an appropriate error
					return nil, errors.New(errInvalidAppRoleID)
				}
			}
		}
		if baoProvider.Auth.Cert != nil {
			if err := esutils.ValidateReferentSecretSelector(store, baoProvider.Auth.Cert.ClientCert); err != nil {
				return nil, fmt.Errorf(errInvalidClientCert, err)
			}
			if err := esutils.ValidateReferentSecretSelector(store, baoProvider.Auth.Cert.SecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidCertSec, err)
			}
		}
		if baoProvider.Auth.Jwt != nil {
			if baoProvider.Auth.Jwt.SecretRef != nil {
				if err := esutils.ValidateReferentSecretSelector(store, *baoProvider.Auth.Jwt.SecretRef); err != nil {
					return nil, fmt.Errorf(errInvalidJwtSec, err)
				}
			} else if baoProvider.Auth.Jwt.ServiceAccountRef != nil {
				if err := esutils.ValidateReferentServiceAccountSelector(store, *baoProvider.Auth.Jwt.ServiceAccountRef); err != nil {
					return nil, fmt.Errorf(errInvalidJwtK8sSA, err)
				}
			} else {
				return nil, errors.New(errJwtNoTokenSource)
			}
		}
		if baoProvider.Auth.Kubernetes != nil {
			if baoProvider.Auth.Kubernetes.ServiceAccountRef != nil {
				if err := esutils.ValidateReferentServiceAccountSelector(store, *baoProvider.Auth.Kubernetes.ServiceAccountRef); err != nil {
					return nil, fmt.Errorf(errInvalidKubeSA, err)
				}
			}
			if baoProvider.Auth.Kubernetes.SecretRef != nil {
				if err := esutils.ValidateReferentSecretSelector(store, *baoProvider.Auth.Kubernetes.SecretRef); err != nil {
					return nil, fmt.Errorf(errInvalidKubeSec, err)
				}
			}
		}
		if baoProvider.Auth.Ldap != nil {
			if err := esutils.ValidateReferentSecretSelector(store, baoProvider.Auth.Ldap.SecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidLdapSec, err)
			}
		}
		if baoProvider.Auth.UserPass != nil {
			if err := esutils.ValidateReferentSecretSelector(store, baoProvider.Auth.UserPass.SecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidUserPassSec, err)
			}
		}
		if baoProvider.Auth.TokenSecretRef != nil {
			if err := esutils.ValidateReferentSecretSelector(store, *baoProvider.Auth.TokenSecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidTokenRef, err)
			}
		}
		if baoProvider.Auth.Iam != nil {
			if baoProvider.Auth.Iam.JWTAuth != nil {
				if baoProvider.Auth.Iam.JWTAuth.ServiceAccountRef != nil {
					if err := esutils.ValidateReferentServiceAccountSelector(store, *baoProvider.Auth.Iam.JWTAuth.ServiceAccountRef); err != nil {
						return nil, fmt.Errorf(errInvalidTokenRef, err)
					}
				}
			}

			if baoProvider.Auth.Iam.SecretRef != nil {
				if err := esutils.ValidateReferentSecretSelector(store, baoProvider.Auth.Iam.SecretRef.AccessKeyID); err != nil {
					return nil, fmt.Errorf(errInvalidTokenRef, err)
				}
				if err := esutils.ValidateReferentSecretSelector(store, baoProvider.Auth.Iam.SecretRef.SecretAccessKey); err != nil {
					return nil, fmt.Errorf(errInvalidTokenRef, err)
				}
				if baoProvider.Auth.Iam.SecretRef.SessionToken != nil {
					if err := esutils.ValidateReferentSecretSelector(store, *baoProvider.Auth.Iam.SecretRef.SessionToken); err != nil {
						return nil, fmt.Errorf(errInvalidTokenRef, err)
					}
				}
			}
		}
	}
	if baoProvider.ClientTLS.CertSecretRef != nil && baoProvider.ClientTLS.KeySecretRef != nil {
		if err := esutils.ValidateReferentSecretSelector(store, *baoProvider.ClientTLS.CertSecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidClientTLSCert, err)
		}
		if err := esutils.ValidateReferentSecretSelector(store, *baoProvider.ClientTLS.KeySecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidClientTLSSecret, err)
		}
	} else if baoProvider.ClientTLS.CertSecretRef != nil || baoProvider.ClientTLS.KeySecretRef != nil {
		return nil, errors.New(errInvalidClientTLS)
	}

	// Validate CAS configuration
	if baoProvider.CheckAndSet != nil && baoProvider.CheckAndSet.Required {
		if baoProvider.Version == esv1.OpenBaoKVStoreV1 {
			return nil, errors.New(errCASNotSupportedInKVv1)
		}
	}

	return nil, nil
}

func (c *client) Validate() (esv1.ValidationResult, error) {
	// when using referent namespace we can not validate the token
	// because the namespace is not known yet when Validate() is called
	// from the SecretStore controller.
	if c.storeKind == esv1.ClusterSecretStoreKind && isReferentSpec(c.store) {
		return esv1.ValidationResultUnknown, nil
	}
	if c.tokenExpiryTime != nil && c.tokenExpiryTime.After(time.Now()) {
		return esv1.ValidationResultReady, nil
	}
	_, _, err := checkToken(context.Background(), c.token)
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf(errInvalidCredentials, err)
	}
	return esv1.ValidationResultReady, nil
}
