/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package common

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"crypto/rand"
	"crypto/x509"
	"encoding/pem"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	p12 "software.sslmate.com/src/go-pkcs12"
)

// This case creates multiple secrets with simple key/value pairs and syncs them using multiple .Spec.Data blocks.
// Not supported by: vault.
func SimpleDataSync(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync simple secrets from .Data[]", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "other")
		secretValue := "bar"
		tc.Secrets = map[string]string{
			secretKey1: secretValue,
			secretKey2: secretValue,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte(secretValue),
				secretKey2: []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: secretKey1,
				},
			},
			{
				SecretKey: secretKey2,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: secretKey2,
				},
			},
		}
	}
}

// This case creates multiple secrets with json values and syncs them using multiple .Spec.Data blocks.
// The data is extracted from the JSON key using ref.Property.
func JSONDataWithProperty(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync multiple secrets from .Data[]", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "two")
		secretValue1 := "{\"foo1\":\"foo1-val\",\"bar1\":\"bar1-val\"}"
		secretValue2 := "{\"foo2\":\"foo2-val\",\"bar2\":\"bar2-val\"}"
		tc.Secrets = map[string]string{
			secretKey1: secretValue1,
			secretKey2: secretValue2,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte("foo1-val"),
				secretKey2: []byte("bar2-val"),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "foo1",
				},
			},
			{
				SecretKey: secretKey2,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey2,
					Property: "bar2",
				},
			},
		}
	}
}

// This case creates multiple secrets with json values and renders a template.
// The data is extracted from the JSON key using ref.Property.
func JSONDataWithTemplate(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync json secrets with template", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "other")
		secretValue1 := "{\"foo1\":\"foo1-val\",\"bar1\":\"bar1-val\"}"
		secretValue2 := "{\"foo2\":\"foo2-val\",\"bar2\":\"bar2-val\"}"
		tc.Secrets = map[string]string{
			secretKey1: secretValue1,
			secretKey2: secretValue2,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"example": "annotation",
				},
				Labels: map[string]string{
					"example": "label",
				},
			},
			Data: map[string][]byte{
				"my-data": []byte(`executed: foo1-val|bar2-val`),
			},
		}
		tc.ExternalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Metadata: esv1alpha1.ExternalSecretTemplateMetadata{
				Annotations: map[string]string{
					"example": "annotation",
				},
				Labels: map[string]string{
					"example": "label",
				},
			},
			Data: map[string]string{
				"my-data": "executed: {{ .one | toString }}|{{ .two | toString }}",
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: "one",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "foo1",
				},
			},
			{
				SecretKey: "two",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey2,
					Property: "bar2",
				},
			},
		}
	}
}

// This case creates one secret with json values and syncs them using a single .Spec.DataFrom block.
func JSONDataFromSync(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync secrets with dataFrom", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		targetSecretKey1 := "name"
		targetSecretValue1 := "great-name"
		targetSecretKey2 := "surname"
		targetSecretValue2 := "great-surname"
		secretValue := fmt.Sprintf("{ \"%s\": \"%s\", \"%s\": \"%s\" }", targetSecretKey1, targetSecretValue1, targetSecretKey2, targetSecretValue2)
		tc.Secrets = map[string]string{
			secretKey1: secretValue,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				targetSecretKey1: []byte(targetSecretValue1),
				targetSecretKey2: []byte(targetSecretValue2),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esv1alpha1.ExternalSecretDataRemoteRef{
			{
				Key: secretKey1,
			},
		}
	}
}

// This case creates a secret with a nested json value. It is synced into two secrets.
// The values from the nested data are extracted using gjson.
// not supported by: vault.
func NestedJSONWithGJSON(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync nested json secrets and get inner keys", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		targetSecretKey1 := "firstname"
		targetSecretValue1 := "Tom"
		targetSecretKey2 := "first_friend"
		targetSecretValue2 := "Roger"
		secretValue := fmt.Sprintf(
			`{
				"name": {"first": "%s", "last": "Anderson"},
				"friends":
				[
					{"first": "Dale", "last": "Murphy"},
					{"first": "%s", "last": "Craig"},
					{"first": "Jane", "last": "Murphy"}
				]
			}`, targetSecretValue1, targetSecretValue2)
		tc.Secrets = map[string]string{
			secretKey1: secretValue,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				targetSecretKey1: []byte(targetSecretValue1),
				targetSecretKey2: []byte(targetSecretValue2),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: targetSecretKey1,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "name.first",
				},
			},
			{
				SecretKey: targetSecretKey2,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "friends.1.first",
				},
			},
		}
	}
}

// P12Cert case creates a secret with a p12 cert containing a privkey and cert bundled together.
// It uses templating to generate a k8s secret of type tls with pem values
func P12Cert(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync p12 configurated cert secrets with template", func(tc *framework.TestCase) {
		cloudSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, "p12-cert-example")
		certPEM := `-----BEGIN CERTIFICATE-----
MIIFQjCCBCqgAwIBAgISBHszg5W2maz/7CIxGrf7mqukMA0GCSqGSIb3DQEBCwUA
MDIxCzAJBgNVBAYTAlVTMRYwFAYDVQQKEw1MZXQncyBFbmNyeXB0MQswCQYDVQQD
EwJSMzAeFw0yMTA3MjQxMjQyMzNaFw0yMTEwMjIxMjQyMzFaMCgxJjAkBgNVBAMT
HXRlbXBvcmFyeS5leHRlcm5hbC1zZWNyZXRzLmlvMIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAyRROdZskA8qnGnoMgQ5Ry5MVY/lgo3HzlhKq02u23J2w
14w+LiEU2hcSJKYv5OXysbfq7M52u2zXYZXs6krkQZlYNpFw7peZ0JtUbVkSpST/
X4b1GJKDSkRs7fTi+v+pb9OT9rTbtd8jfGe/YCe5rjXEm/ih2DgS13737lKCD5n6
3QUOG7CR+SKFeRXOGkncqJHAyRkpNfAmS8m1C+ucodfjSFoqAwwVGx7eyEktG4s/
JbwLEb03hGrP15vnnOgxQmiAzWskxhMyHX6vmA71Oq4F3RVsuD3CEjKzgJ2+ghk3
BIY3DZSfSReWSMYM573YFglENi+qJK012XnFmZcevwIDAQABo4ICWjCCAlYwDgYD
VR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNV
HRMBAf8EAjAAMB0GA1UdDgQWBBRvn1wGi46XcyhRIIxJkSSUoCyoNzAfBgNVHSME
GDAWgBQULrMXt1hWy65QCUDmH6+dixTCxjBVBggrBgEFBQcBAQRJMEcwIQYIKwYB
BQUHMAGGFWh0dHA6Ly9yMy5vLmxlbmNyLm9yZzAiBggrBgEFBQcwAoYWaHR0cDov
L3IzLmkubGVuY3Iub3JnLzAoBgNVHREEITAfgh10ZW1wb3JhcnkuZXh0ZXJuYWwt
c2VjcmV0cy5pbzBMBgNVHSAERTBDMAgGBmeBDAECATA3BgsrBgEEAYLfEwEBATAo
MCYGCCsGAQUFBwIBFhpodHRwOi8vY3BzLmxldHNlbmNyeXB0Lm9yZzCCAQYGCisG
AQQB1nkCBAIEgfcEgfQA8gB3APZclC/RdzAiFFQYCDCUVo7jTRMZM7/fDC8gC8xO
8WTjAAABetjA0asAAAQDAEgwRgIhAPYbBNim7q3P0qmD9IrAx1E1fEClYpoLrAVs
4LGBkQobAiEA+IaTPWs9eHmqtCwar96PNxE0Iucak0DYkgfcWJT5gfYAdwBvU3as
MfAxGdiZAKRRFf93FRwR2QLBACkGjbIImjfZEwAAAXrYwNJTAAAEAwBIMEYCIQDY
xWJKFljK1AW2z/uVsU7TwcAAcIqUf5/nhS04JAwpfwIhANDTvwvcRvPebU7fv6dq
lNH1g2Oyv/4Vm7W+Vrc5cFD0MA0GCSqGSIb3DQEBCwUAA4IBAQAR29s3pDGZbNPN
5K+Zqg9UDT8s+P0fb9r97T7hWEFkiUtG4bz7QvGzSoDXhD/DZkdjLmkX7+bLiE3L
hRSSYe+Am+Bw5soyzefX2FHAUeOLeK0mJhOrdiKqrW4nnvOOJWLkcWS799kW2z7j
2MgUWTOz/xXGUOWHt1KjyoM31G3shoAIB9lg3lHbuVIyDd3yyUpjt0zevVdYrO9G
CgI2mJfv26EiddBvgudzN+R5Ayis9czaFHu8gpplaf9DahaKs1Uys6lg0HnzRn3l
XMYitHfpGhc+DTTiTWMQ13J0b1j4yv8A7ZaG2366aa28oSTD6eQFhmVCBwa54j++
IOwzHn5R
-----END CERTIFICATE-----
`
		privkeyPEM := `-----BEGIN PRIVATE KEY-----
MIIEwAIBADANBgkqhkiG9w0BAQEFAASCBKowggSmAgEAAoIBAQDJFE51myQDyqca
egyBDlHLkxVj+WCjcfOWEqrTa7bcnbDXjD4uIRTaFxIkpi/k5fKxt+rszna7bNdh
lezqSuRBmVg2kXDul5nQm1RtWRKlJP9fhvUYkoNKRGzt9OL6/6lv05P2tNu13yN8
Z79gJ7muNcSb+KHYOBLXfvfuUoIPmfrdBQ4bsJH5IoV5Fc4aSdyokcDJGSk18CZL
ybUL65yh1+NIWioDDBUbHt7ISS0biz8lvAsRvTeEas/Xm+ec6DFCaIDNayTGEzId
fq+YDvU6rgXdFWy4PcISMrOAnb6CGTcEhjcNlJ9JF5ZIxgznvdgWCUQ2L6okrTXZ
ecWZlx6/AgMBAAECggEBAI9sDX5zFuAhdsk6zppqtUrn8TTq1dQe3ihnzjKYvMhl
LZLA9EUA0ZexJv6/DqBMp6u9TDJ2HVgYDRQM1PxUSLTFhJb/bDayKUMS18ha5SKn
3gKsBzvsnPqnDa84oYF4Q8mAdyRb4e66ZtxAP8985kLtFPxO/llzvXS5mmwBq8Ul
wlLOg5xAXubm3vgLyFm2GW9qI6ZvY9mmh1mv5ZLP8/8hikRjwJijnX3dyqqIAYnc
DHjJYy2I1VxGJybqVQRquG++Tl4qLXbOUZ/lhKe62ARx/MBR9lEst5TURc9N7U3D
Mgsu7FcFwqjVkig3P0XiNRWwCu0HrYee5rLXmtDnF9kCgYEA69+OuJM/RIsrLQQd
1alppgT+SFyaJM3X1MJD3yxW6Vqqvkhqe7+XCWnmVYcpHPcilWmZnnQ3PiWqPJ8A
3mIMp+Xg0ddFQXb3n7z4D0Mg4IPzvSKnlieTT1rDhhHRv/xArw1UBkF6kqcnZizZ
FcWcOIt/dYodTWZzPJtLtf7QW0sCgYEA2jy0vJ5rg0/CSinkccreegC6gbbd+oE9
uR/aGeu1XmnULoYYMMy7BLqd8/OiXvujbgUSUWnzbEclR88dPDkiRxDL7mYiaCn+
l9jPuVB1W5x6irJdG/7lpSnLuijpkzey177ZKrlfGsOjtVZsc1ytnqTCWsF1r9eY
yXCSvkJQjd0CgYEA5+vl0hh+MfBA4L9WcnpkNehc+luK+LspB7qHr81SG5qZngVo
JgspAAmPf/Mo+qEI8S5m7MVKeCHitD6HRSHVXdUK7GklYIwQSJEuuxr/HaLAquyD
KYH6NyGAdLfarFHka/rH7mq9kasnczCPtveZdoO7LKBD1ZHxptrvY6CLz+cCgYEA
yEq2xfXPTrDA7DgOhbFfBjHs+mfOyr4a2/Czxt5hkskmB5ziTsdXTTvJA8Ay4WGp
2Kum6DmJQ3L4cDNR7ZeyMe7ke2QZZ+hC1TITU0zYqL+wZ+LTOYJzWWZGqBAsbwTL
it6JiYCgHHw5n5A18Jq6bcNg7NJpJH2GqDo9M4jBTbECgYEAlMuvNExEXGVzWrGF
NXHpAev64RJ2jTq59jtmxWrNvzeWJREOWd/Nt+0t+bE0sHMfgaMrhNFWiR8oesrF
Jdx0ECYawviQoreDAyIXV6HouoeRbDtLZ9AJvxMoIjGcjAR2FQHc3yx4h/lf3Tfx
x6HaRh+EUwU51von6M9lEF9/p5Q=
-----END PRIVATE KEY-----
`
		blockCert, _ := pem.Decode([]byte(certPEM))
		cert, _ := x509.ParseCertificate(blockCert.Bytes)
		blockPrivKey, _ := pem.Decode([]byte(privkeyPEM))
		privkey, _ := x509.ParsePKCS8PrivateKey(blockPrivKey.Bytes)
		emptyCACerts := []*x509.Certificate{}
		p12Cert, _ := p12.Encode(rand.Reader, privkey, cert, emptyCACerts, "")

		tc.Secrets = map[string]string{
			cloudSecretName: string(p12Cert),
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.crt": []byte(certPEM),
				"tls.key": []byte(privkeyPEM),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: cloudSecretName,
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Type: v1.SecretTypeTLS,
			Data: map[string]string{
				"tls.crt": "{{ .mysecret | pkcs12cert | pemCertificate }}",
				"tls.key": "{{ .mysecret | pkcs12key | pemPrivateKey }}",
			},
		}
	}
}
