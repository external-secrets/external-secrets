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

package anywhere

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
)

// Container for certificate data returned to the SDK as JSON.
type CertificateData struct {
	// Type for the key contained in the certificate.
	// Passed back to the `sign-string` command
	KeyType string `json:"keyType"`
	// Certificate, as base64-encoded DER; used in the `x-amz-x509`
	// header in the API request.
	CertificateData string `json:"certificateData"`
	// Serial number of the certificate. Used in the credential
	// field of the Authorization header
	SerialNumber string `json:"serialNumber"`
	// Supported signing algorithms based on the KeyType
	Algorithms []string `json:"supportedAlgorithms"`
}

// Load the certificate referenced by `certificateId` and extract
// details required by the SDK to construct the StringToSign.
func readCertificateData(certificateID string) (CertificateData, error) {
	block, err := parseDERFromPEM(certificateID, "CERTIFICATE")
	if err != nil {
		return CertificateData{}, errors.New("could not parse PEM data")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CertificateData{}, errors.New("could not parse certificate")
	}

	//extract serial number
	serialNumber := cert.SerialNumber.String()

	//encode certificate
	encodedDer, _ := encodeDer(block.Bytes)

	//extract key type
	var keyType string
	switch cert.PublicKeyAlgorithm {
	case x509.RSA:
		keyType = "RSA"
	case x509.ECDSA:
		keyType = "EC"
	default:
		keyType = ""
	}

	supportedAlgorithms := []string{
		fmt.Sprintf("%sSHA256", keyType),
		fmt.Sprintf("%sSHA384", keyType),
		fmt.Sprintf("%sSHA512", keyType),
	}

	return CertificateData{keyType, encodedDer, serialNumber, supportedAlgorithms}, nil
}

func encodeDer(der []byte) (string, error) {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write(der)
	encoder.Close()
	return buf.String(), nil
}
