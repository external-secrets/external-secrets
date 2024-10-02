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

package template

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
)

const (
	errJunk = "error filtering pem: found junk"

	certTypeLeaf         = "leaf"
	certTypeIntermediate = "intermediate"
	certTypeRoot         = "root"
)

func filterPEM(pemType, input string) (string, error) {
	data := []byte(input)
	var blocks []byte
	var block *pem.Block
	var rest []byte
	for {
		block, rest = pem.Decode(data)
		data = rest

		if block == nil {
			break
		}
		if !strings.EqualFold(block.Type, pemType) {
			continue
		}

		var buf bytes.Buffer
		err := pem.Encode(&buf, block)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, buf.Bytes()...)
	}

	if len(blocks) == 0 && len(rest) != 0 {
		return "", errors.New(errJunk)
	}

	return string(blocks), nil
}

func filterCertChain(certType, input string) (string, error) {
	ordered, err := fetchX509CertChains([]byte(input))
	if err != nil {
		return "", err
	}

	switch certType {
	case certTypeLeaf:
		cert := ordered[0]
		if cert.AuthorityKeyId != nil && !bytes.Equal(cert.AuthorityKeyId, cert.SubjectKeyId) {
			return pemEncode(ordered[0].Raw, pemTypeCertificate)
		}
	case certTypeIntermediate:
		if len(ordered) < 2 {
			return "", nil
		}
		var pemData []byte
		for _, cert := range ordered[1:] {
			if isRootCertificate(cert) {
				break
			}
			b := &pem.Block{
				Type:  pemTypeCertificate,
				Bytes: cert.Raw,
			}
			pemData = append(pemData, pem.EncodeToMemory(b)...)
		}
		return string(pemData), nil
	case certTypeRoot:
		cert := ordered[len(ordered)-1]
		if isRootCertificate(cert) {
			return pemEncode(cert.Raw, pemTypeCertificate)
		}
	}

	return "", nil
}

func isRootCertificate(cert *x509.Certificate) bool {
	return cert.AuthorityKeyId == nil || bytes.Equal(cert.AuthorityKeyId, cert.SubjectKeyId)
}

func pemEncode(thing []byte, kind string) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: kind, Bytes: thing})
	return buf.String(), err
}
