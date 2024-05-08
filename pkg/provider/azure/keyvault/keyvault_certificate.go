package keyvault

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

// ClientInMemoryCertificateConfig struct includes a Certificate field to hold the certificate data as a byte slice.
type ClientInMemoryCertificateConfig struct {
	ClientID    string
	Certificate []byte // Certificate data as a byte slice
	TenantID    string
	AuxTenants  []string
	AADEndpoint string
	Resource    string
}

func NewClientInMemoryCertificateConfig(clientID string, certificate []byte, tenantID string) ClientInMemoryCertificateConfig {
	return ClientInMemoryCertificateConfig{
		ClientID:    clientID,
		Certificate: certificate,
		TenantID:    tenantID,
		Resource:    azure.PublicCloud.ResourceManagerEndpoint,
		AADEndpoint: azure.PublicCloud.ActiveDirectoryEndpoint,
	}
}

// ServicePrincipalToken creates a adal.ServicePrincipalToken from client certificate using the certificate byte slice
func (ccc ClientInMemoryCertificateConfig) ServicePrincipalToken() (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(ccc.AADEndpoint, ccc.TenantID)
	if err != nil {
		return nil, err
	}
	// Use the byte slice directly instead of reading from a file
	certificate, rsaPrivateKey, err := loadCertificateFromBytes(ccc.Certificate)

	if err != nil {
		return nil, fmt.Errorf("failed to decode certificate: %v", err)
	}
	return adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, ccc.ClientID, certificate, rsaPrivateKey, ccc.Resource)
}

func loadCertificateFromBytes(certificateBytes []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	var cert *x509.Certificate
	var privateKey *rsa.PrivateKey
	var err error

	// Extract certificate and private key
	for {
		block, rest := pem.Decode(certificateBytes)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse PEM certificate: %w", err)
			}

		} else {
			privateKey, err = parsePrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to extract private key from PEM certificate: %w", err)
			}
		}
		certificateBytes = rest
	}

	return cert, privateKey, nil
}

func parsePrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey:
			return key, nil
		default:
			return nil, fmt.Errorf("found unknown private key type in PKCS#8 wrapping")
		}
	}
	return nil, fmt.Errorf("failed to parse private key")
}

// Implementation of the AuthorizerConfig interface
func (ccc ClientInMemoryCertificateConfig) Authorizer() (autorest.Authorizer, error) {
	spToken, err := ccc.ServicePrincipalToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth token from certificate auth: %v", err)
	}
	return autorest.NewBearerAuthorizer(spToken), nil
}
