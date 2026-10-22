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

package template

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"reflect"
	"testing"

	gopkcs12 "software.sslmate.com/src/go-pkcs12"
)


func TestPemToPkcs12(t *testing.T) {
	// Mock the random generator, to guarantee to always generate the same output
	gopkcs12.Modern = gopkcs12.Modern.WithRand(MockRandomReader{})

	out, err := pemToPkcs12Pass(certData, keyData, "password")
	if err != nil {
		t.Errorf("pemToPkcs12Pass() got error '%v', expected none", err)
		return
	}

	key, cert, cas, err := decodePFX(out)
	if err != nil || cert == nil || len(cas) != 0 || key == nil {
		t.Fatalf("decoded PFX: key=%v cert=%v cas=%d err=%v", key != nil, cert != nil, len(cas), err)
	}
	assertCertificate(t, cert, certData)
	assertKey(t, key, keyData)
}

func TestFullPemToPkcs12(t *testing.T) {
	// Mock the random generator, to guarantee to always generate the same output
	gopkcs12.Modern = gopkcs12.Modern.WithRand(MockRandomReader{})

	out, err := fullPemToPkcs12Pass(certData+"\n"+otherCert, keyData, "password")
	if err != nil {
		t.Errorf("pemToPkcs12Pass() got error '%v', expected none", err)
		return
	}

	key, cert, cas, err := decodePFX(out)
	if err != nil || cert == nil || len(cas) != 1 || key == nil {
		t.Fatalf("decoded PFX: key=%v cert=%v cas=%d err=%v", key != nil, cert != nil, len(cas), err)
	}
	assertCertificate(t, cert, certData)
	assertCertificate(t, cas[0], otherCert)
	assertKey(t, key, keyData)
}

func TestPemTruststoreToPKCS12(t *testing.T) {
	// Mock the random generator, to guarantee to always generate the same output
	gopkcs12.Modern = gopkcs12.Modern.WithRand(MockRandomReader{})

	out, err := pemTruststoreToPKCS12Pass(certData+"\n"+otherCert, "password")
	if err != nil {
		t.Errorf("pemTruststoreToPKCS12Pass() got error '%v', expected none", err)
		return
	}

	data, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		t.Fatal(err)
	}
	certs, err := gopkcs12.DecodeTrustStore(data, "password")
	if err != nil || len(certs) != 2 {
		t.Fatalf("decoded truststore: certs=%d err=%v", len(certs), err)
	}
	assertCertificate(t, certs[0], certData)
	assertCertificate(t, certs[1], otherCert)
}

// Compare decoded contents rather than serialized bytes: go-pkcs12 v0.7.4
// adds the RFC 8018-required NULL PRF parameter, changing the PFX encoding.
func decodePFX(encoded string) (any, *x509.Certificate, []*x509.Certificate, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, nil, nil, err
	}
	return gopkcs12.DecodeChain(data, "password")
}

func assertCertificate(t *testing.T, got *x509.Certificate, encoded string) {
	t.Helper()
	block, _ := pem.Decode([]byte(encoded))
	if block == nil || !reflect.DeepEqual(got.Raw, block.Bytes) {
		t.Fatal("certificate does not match input")
	}
}

func assertKey(t *testing.T, got any, encoded string) {
	t.Helper()
	block, _ := pem.Decode([]byte(encoded))
	if block == nil {
		t.Fatal("invalid input key")
	}
	original, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		original, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	if err != nil || !reflect.DeepEqual(got, original) {
		t.Fatalf("private key does not match input: %v", err)
	}
}

type MockRandomReader struct{}

func (r MockRandomReader) Read(p []byte) (int, error) {
	copy(p, `"0123456789"`)
	return len(p), nil
}
