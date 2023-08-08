package conjur

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/cyberark/conjur-api-go/conjurapi"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/golang-jwt/jwt/v5"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"time"
)

const JwtLifespan = 600     // 10 minutes
const JwtRefreshBuffer = 30 // 30 seconds

// getJWTToken retrieves a JWT token either using the TokenRequest API for a specified service account, or from a jwt stored in a k8s secret.
func (p *Provider) getJWTToken(ctx context.Context, conjurJWTConfig *esv1beta1.ConjurJWT) (string, error) {
	if conjurJWTConfig.ServiceAccountRef != nil {
		// Should work for Kubernetes >=v1.22: fetch token via TokenRequest API
		jwtToken, err := p.getJwtFromServiceAccountTokenRequest(ctx, *conjurJWTConfig.ServiceAccountRef, nil, JwtLifespan)
		if err != nil {
			return "", err
		}
		return jwtToken, nil
	} else if conjurJWTConfig.SecretRef != nil {
		tokenRef := conjurJWTConfig.SecretRef
		if tokenRef.Key == "" {
			tokenRef = conjurJWTConfig.SecretRef.DeepCopy()
			tokenRef.Key = "token"
		}
		jwtToken, err := p.secretKeyRef(ctx, tokenRef)
		if err != nil {
			return "", err
		}
		return jwtToken, nil
	}
	return "", fmt.Errorf("missing ServiceAccountRef or SecretRef")
}

// getJwtFromServiceAccountTokenRequest uses the TokenRequest API to get a JWT token for the given service account.
func (p *Provider) getJwtFromServiceAccountTokenRequest(ctx context.Context, serviceAccountRef esmeta.ServiceAccountSelector, additionalAud []string, expirationSeconds int64) (string, error) {
	audiences := serviceAccountRef.Audiences
	if len(additionalAud) > 0 {
		audiences = append(audiences, additionalAud...)
	}
	tokenRequest := &authenticationv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.namespace,
		},
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}
	if (p.StoreKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		tokenRequest.Namespace = *serviceAccountRef.Namespace
	}
	tokenResponse, err := p.corev1.ServiceAccounts(tokenRequest.Namespace).CreateToken(ctx, serviceAccountRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf(errGetKubeSATokenRequest, serviceAccountRef.Name, err)
	}
	return tokenResponse.Status.Token, nil
}

// newClientFromJwt creates a new Conjur client using the given JWT Auth Config.
func (p *Provider) newClientFromJwt(ctx context.Context, config conjurapi.Config, jwtAuth *esv1beta1.ConjurJWT) (Client, error) {
	jwtToken, getJWTError := p.getJWTToken(ctx, jwtAuth)
	if getJWTError != nil {
		return nil, getJWTError
	}

	expirationTime, err := determineExpirationTime(jwtToken)

	if err != nil {
		// Catch bad JWT token, or tokens that have no expiration time
		return nil, err
	}

	client, err := p.clientApi.NewClientFromJWT(config, jwtToken, jwtAuth.ServiceId)

	// Only update the renewClientAfter if we successfully create a new client
	p.renewClientAfter = expirationTime.Add(time.Duration(-JwtRefreshBuffer) * time.Second)

	return client, nil
}

func determineExpirationTime(jwtToken string) (*jwt.NumericDate, error) {
	parsedToken, _, err := new(jwt.Parser).ParseUnverified(jwtToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf(errFailedToParseJWTToken, err)
	}
	expirationTime, err := parsedToken.Claims.GetExpirationTime()
	if expirationTime == nil || err != nil {
		return nil, fmt.Errorf(errFailedToDetermineJWTTokenExpiration)
	}
	return expirationTime, nil
}

// newHTTPSClient creates a new HTTPS client with the given cert
func newHTTPSClient(cert []byte) (*http.Client, error) {
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(cert)
	if !ok {
		return nil, fmt.Errorf("can't append Conjur SSL cert")
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}
	return &http.Client{Transport: tr, Timeout: time.Second * 10}, nil
}
