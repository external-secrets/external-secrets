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
	"sync"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeResolver struct {
	Once    sync.Once
	keyPEM  string
	certPEM string
}

type FakeSecretKeyResolver struct {
	fakeResolver FakeResolver
}

func (fr *FakeSecretKeyResolver) Resolve(_ context.Context, _ kclient.Client, _, _ string, ref *esmeta.SecretKeySelector) (string, error) {
	if ref.Name == "Valid token auth" {
		return "Valid", nil
	}
	if ref.Name == "Valid mtls client certificate" || ref.Name == "Valid mtls client key" {
		var err error
		fr.fakeResolver.Once.Do(func() {
			var privKey *rsa.PrivateKey
			privKey, err = rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return
			}
			fr.fakeResolver.keyPEM = string(pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privKey),
			}))

			template := x509.Certificate{}
			var cert []byte
			cert, err = x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
			if err != nil {
				return
			}
			fr.fakeResolver.certPEM = string(pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: cert,
			}))
		})

		if err != nil {
			return "", err
		}

		if ref.Name == "Valid mtls client certificate" {
			return fr.fakeResolver.certPEM, nil
		}
		return fr.fakeResolver.keyPEM, nil
	}
	return "", nil
}
