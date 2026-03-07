/*
Copyright © 2025 ESO Maintainer Team

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

package fake

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type FakeSecretKeyResolver struct {
	privateKey *rsa.PrivateKey
}

func (fr *FakeSecretKeyResolver) Resolve(_ context.Context, _ kclient.Client, _, _ string, ref esmeta.SecretKeySelector) (string, error) {
	switch ref.Name {
	case "Valid token auth":
		return "Valid", nil
	case "Valid mtls client certificate":
		return fr.generateFakeCert()
	case "Valid mtls client key":
		return fr.generateFakeKey()
	default:
		return "", nil
	}
}

func (fr *FakeSecretKeyResolver) generateFakeKey() (string, error) {
	var err error

	fr.privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	if err = fr.privateKey.Validate(); err != nil {
		return "", err
	}

	privateKeyDER := x509.MarshalPKCS1PrivateKey(fr.privateKey)
	pemBlock := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	}
	privateKeyPEM := pem.EncodeToMemory(&pemBlock)

	return string(privateKeyPEM), nil
}

func (fr *FakeSecretKeyResolver) generateFakeCert() (string, error) {
	template := x509.Certificate{}
	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, &fr.privateKey.PublicKey, fr.privateKey)
	if err != nil {
		return "", err
	}

	pemBlock := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}
	certPEM := pem.EncodeToMemory(&pemBlock)

	return string(certPEM), nil
}
