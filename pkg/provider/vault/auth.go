package vault

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"
	authkubernetes "github.com/hashicorp/vault/api/auth/kubernetes"
	authldap "github.com/hashicorp/vault/api/auth/ldap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"os"
	"strings"
)

func (v *client) setAuth(ctx context.Context, cfg *vault.Config) error {
	tokenExists, err := setSecretKeyToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setAppRoleToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setKubernetesAuthToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setLdapAuthToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setJwtAuthToken(ctx, v)
	if tokenExists {
		return err
	}

	tokenExists, err = setCertAuthToken(ctx, v, cfg)
	if tokenExists {
		return err
	}

	return errors.New(errAuthFormat)
}

func setAppRoleToken(ctx context.Context, v *client) (bool, error) {
	tokenRef := v.store.Auth.TokenSecretRef
	if tokenRef != nil {
		token, err := v.secretKeyRef(ctx, tokenRef)
		if err != nil {
			return true, err
		}
		v.client.SetToken(token)
		return true, nil
	}
	return false, nil
}

func setSecretKeyToken(ctx context.Context, v *client) (bool, error) {
	appRole := v.store.Auth.AppRole
	if appRole != nil {
		err := v.requestTokenWithAppRoleRef(ctx, appRole)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setKubernetesAuthToken(ctx context.Context, v *client) (bool, error) {
	kubernetesAuth := v.store.Auth.Kubernetes
	if kubernetesAuth != nil {
		err := v.requestTokenWithKubernetesAuth(ctx, kubernetesAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setLdapAuthToken(ctx context.Context, v *client) (bool, error) {
	ldapAuth := v.store.Auth.Ldap
	if ldapAuth != nil {
		err := v.requestTokenWithLdapAuth(ctx, ldapAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setJwtAuthToken(ctx context.Context, v *client) (bool, error) {
	jwtAuth := v.store.Auth.Jwt
	if jwtAuth != nil {
		err := v.requestTokenWithJwtAuth(ctx, jwtAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func setCertAuthToken(ctx context.Context, v *client, cfg *vault.Config) (bool, error) {
	certAuth := v.store.Auth.Cert
	if certAuth != nil {
		err := v.requestTokenWithCertAuth(ctx, certAuth, cfg)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (v *client) requestTokenWithAppRoleRef(ctx context.Context, appRole *esv1beta1.VaultAppRole) error {
	roleID := strings.TrimSpace(appRole.RoleID)

	secretID, err := v.secretKeyRef(ctx, &appRole.SecretRef)
	if err != nil {
		return err
	}
	secret := approle.SecretID{FromString: secretID}
	appRoleClient, err := approle.NewAppRoleAuth(roleID, &secret, approle.WithMountPath(appRole.Path))
	if err != nil {
		return err
	}
	_, err = v.auth.Login(ctx, appRoleClient)
	if err != nil {
		return err
	}
	return nil
}

func (v *client) requestTokenWithKubernetesAuth(ctx context.Context, kubernetesAuth *esv1beta1.VaultKubernetesAuth) error {
	jwtString, err := getJwtString(ctx, v, kubernetesAuth)
	if err != nil {
		return err
	}
	k, err := authkubernetes.NewKubernetesAuth(kubernetesAuth.Role, authkubernetes.WithServiceAccountToken(jwtString), authkubernetes.WithMountPath(kubernetesAuth.Path))
	if err != nil {
		return err
	}
	_, err = v.auth.Login(ctx, k)
	if err != nil {
		return err
	}
	return nil
}

func getJwtString(ctx context.Context, v *client, kubernetesAuth *esv1beta1.VaultKubernetesAuth) (string, error) {
	if kubernetesAuth.ServiceAccountRef != nil {
		jwt, err := v.secretKeyRefForServiceAccount(ctx, kubernetesAuth.ServiceAccountRef)
		if err != nil {
			return "", err
		}
		return jwt, nil
	} else if kubernetesAuth.SecretRef != nil {
		tokenRef := kubernetesAuth.SecretRef
		if tokenRef.Key == "" {
			tokenRef = kubernetesAuth.SecretRef.DeepCopy()
			tokenRef.Key = "token"
		}
		jwt, err := v.secretKeyRef(ctx, tokenRef)
		if err != nil {
			return "", err
		}
		return jwt, nil
	} else {
		// Kubernetes authentication is specified, but without a referenced
		// Kubernetes secret. We check if the file path for in-cluster service account
		// exists and attempt to use the token for Vault Kubernetes auth.
		if _, err := os.Stat(serviceAccTokenPath); err != nil {
			return "", fmt.Errorf(errServiceAccount, err)
		}
		jwtByte, err := os.ReadFile(serviceAccTokenPath)
		if err != nil {
			return "", fmt.Errorf(errServiceAccount, err)
		}
		return string(jwtByte), nil
	}
}

func (v *client) requestTokenWithLdapAuth(ctx context.Context, ldapAuth *esv1beta1.VaultLdapAuth) error {
	username := strings.TrimSpace(ldapAuth.Username)

	password, err := v.secretKeyRef(ctx, &ldapAuth.SecretRef)
	if err != nil {
		return err
	}
	pass := authldap.Password{FromString: password}
	l, err := authldap.NewLDAPAuth(username, &pass, authldap.WithMountPath(ldapAuth.Path))
	if err != nil {
		return err
	}
	_, err = v.auth.Login(ctx, l)
	if err != nil {
		return err
	}
	return nil
}

func (v *client) requestTokenWithJwtAuth(ctx context.Context, jwtAuth *esv1beta1.VaultJwtAuth) error {
	role := strings.TrimSpace(jwtAuth.Role)
	var jwt string
	var err error
	if jwtAuth.SecretRef != nil {
		jwt, err = v.secretKeyRef(ctx, jwtAuth.SecretRef)
	} else if k8sServiceAccountToken := jwtAuth.KubernetesServiceAccountToken; k8sServiceAccountToken != nil {
		audiences := k8sServiceAccountToken.Audiences
		if audiences == nil {
			audiences = &[]string{"vault"}
		}
		expirationSeconds := k8sServiceAccountToken.ExpirationSeconds
		if expirationSeconds == nil {
			tmp := int64(600)
			expirationSeconds = &tmp
		}
		jwt, err = v.serviceAccountToken(ctx, k8sServiceAccountToken.ServiceAccountRef, *audiences, *expirationSeconds)
	} else {
		err = fmt.Errorf(errJwtNoTokenSource)
	}
	if err != nil {
		return err
	}

	parameters := map[string]interface{}{
		"role": role,
		"jwt":  jwt,
	}
	url := strings.Join([]string{"auth", jwtAuth.Path, "login"}, "/")
	vaultResult, err := v.logical.WriteWithContext(ctx, url, parameters)
	if err != nil {
		return err
	}

	token, err := vaultResult.TokenID()
	if err != nil {
		return fmt.Errorf(errVaultToken, err)
	}
	v.client.SetToken(token)
	return nil
}

func (v *client) requestTokenWithCertAuth(ctx context.Context, certAuth *esv1beta1.VaultCertAuth, cfg *vault.Config) error {
	clientKey, err := v.secretKeyRef(ctx, &certAuth.SecretRef)
	if err != nil {
		return err
	}

	clientCert, err := v.secretKeyRef(ctx, &certAuth.ClientCert)
	if err != nil {
		return err
	}

	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return fmt.Errorf(errClientTLSAuth, err)
	}

	if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	url := strings.Join([]string{"auth", "cert", "login"}, "/")
	vaultResult, err := v.logical.WriteWithContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf(errVaultRequest, err)
	}
	token, err := vaultResult.TokenID()
	if err != nil {
		return fmt.Errorf(errVaultToken, err)
	}
	v.client.SetToken(token)
	return nil
}

func (v *client) secretKeyRefForServiceAccount(ctx context.Context, serviceAccountRef *esmeta.ServiceAccountSelector) (string, error) {
	serviceAccount := &corev1.ServiceAccount{}
	ref := types.NamespacedName{
		Namespace: v.namespace,
		Name:      serviceAccountRef.Name,
	}
	if (v.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		ref.Namespace = *serviceAccountRef.Namespace
	}
	err := v.kube.Get(ctx, ref, serviceAccount)
	if err != nil {
		return "", fmt.Errorf(errGetKubeSA, ref.Name, err)
	}
	if len(serviceAccount.Secrets) == 0 {
		return "", fmt.Errorf(errGetKubeSASecrets, ref.Name)
	}
	for _, tokenRef := range serviceAccount.Secrets {
		retval, err := v.secretKeyRef(ctx, &esmeta.SecretKeySelector{
			Name:      tokenRef.Name,
			Namespace: &ref.Namespace,
			Key:       "token",
		})

		if err != nil {
			continue
		}

		return retval, nil
	}
	return "", fmt.Errorf(errGetKubeSANoToken, ref.Name)
}
