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
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
)

// Create a function that will sign requests, given the signing certificate, optional certificate chain, and the private key.
func CreateSignFunction(privateKey crypto.PrivateKey, certificate x509.Certificate, certificateChain []x509.Certificate) func(*request.Request) {
	v4x509 := RolesAnywhereSigner{privateKey, certificate, certificateChain}
	return func(r *request.Request) {
		v4x509.SignWithCurrTime(r)
	}
}

type RolesAnywhereSigner struct {
	PrivateKey       crypto.PrivateKey
	Certificate      x509.Certificate
	CertificateChain []x509.Certificate
}

// Define constants used in signing.
const (
	AWS4X509RSASHA256   = "AWS4-X509-RSA-SHA256"
	AWS4X509ECDSASHA256 = "AWS4-X509-ECDSA-SHA256"
	timeFormat          = "20060102T150405Z"
	shortTimeFormat     = "20060102"
	XAMZDate            = "X-Amz-Date"
	XAMZX509            = "X-Amz-X509"
	XAMZX509Chain       = "X-Amz-X509-Chain"
	XAMZContentSHA256   = "X-Amz-Content-Sha256"
	authorization       = "Authorization"
	host                = "Host"
	emptyStringSHA256   = `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
)

// Sign the request using the current time
func (v4x509 RolesAnywhereSigner) SignWithCurrTime(req *request.Request) error {
	// Find the signing algorithm
	var signingAlgorithm string
	_, isRsaKey := v4x509.PrivateKey.(rsa.PrivateKey)
	if isRsaKey {
		signingAlgorithm = AWS4X509RSASHA256
	}
	_, isEcKey := v4x509.PrivateKey.(ecdsa.PrivateKey)
	if isEcKey {
		signingAlgorithm = AWS4X509ECDSASHA256
	}
	if signingAlgorithm == "" {
		return errors.New("unsupported algorithm")
	}

	region := req.ClientInfo.SigningRegion
	if region == "" {
		region = aws.StringValue(req.Config.Region)
	}

	name := req.ClientInfo.SigningName
	if name == "" {
		name = req.ClientInfo.ServiceName
	}

	signerParams := SignerParams{time.Now(), region, name, signingAlgorithm}

	// Set headers that are necessary for signing
	req.HTTPRequest.Header.Set(host, req.HTTPRequest.URL.Host)
	req.HTTPRequest.Header.Set(XAMZDate, signerParams.GetFormattedSigningDateTime())
	req.HTTPRequest.Header.Set(XAMZX509, certificateToString(v4x509.Certificate))
	if v4x509.CertificateChain != nil {
		req.HTTPRequest.Header.Set(XAMZX509Chain, certificateChainToString(v4x509.CertificateChain))
	}

	contentSha256 := calculateContentHash(req.HTTPRequest, req.Body)
	if req.HTTPRequest.Header.Get(XAMZContentSHA256) == "required" {
		req.HTTPRequest.Header.Set(XAMZContentSHA256, contentSha256)
	}

	canonicalRequest, signedHeadersString := createCanonicalRequest(req.HTTPRequest, req.Body, contentSha256)

	stringToSign := CreateStringToSign(canonicalRequest, signerParams)

	signingResult, _ := signPayload([]byte(stringToSign), SigningOpts{v4x509.PrivateKey, crypto.SHA256})

	req.HTTPRequest.Header.Set(authorization, BuildAuthorizationHeader(req.HTTPRequest, req.Body, signedHeadersString, signingResult.Signature, v4x509.Certificate, signerParams))
	req.SignedHeaderVals = req.HTTPRequest.Header
	return nil
}

// Builds the complete authorization header
func BuildAuthorizationHeader(request *http.Request, body io.ReadSeeker, signedHeadersString string, signature string, certificate x509.Certificate, signerParams SignerParams) string {
	signingCredentials := certificate.SerialNumber.String() + "/" + signerParams.GetScope()
	credential := "Credential=" + signingCredentials
	signerHeaders := "SignedHeaders=" + signedHeadersString
	signatureHeader := "Signature=" + signature

	var authHeaderStringBuilder strings.Builder
	authHeaderStringBuilder.WriteString(signerParams.SigningAlgorithm)
	authHeaderStringBuilder.WriteString(" ")
	authHeaderStringBuilder.WriteString(credential)
	authHeaderStringBuilder.WriteString(", ")
	authHeaderStringBuilder.WriteString(signerHeaders)
	authHeaderStringBuilder.WriteString(", ")
	authHeaderStringBuilder.WriteString(signatureHeader)
	authHeaderString := authHeaderStringBuilder.String()
	return authHeaderString
}

type SigningOpts struct {
	// Private key to use for the signing operation.
	PrivateKey crypto.PrivateKey
	// Digest to use in the signing operation. For example, SHA256
	Digest crypto.Hash
}

// Container for data returned after performing a signing operation.
type SigningResult struct {
	// Signature encoded in hex.
	Signature string `json:"signature"`
}

// Sign the provided payload with the specified options.
func signPayload(payload []byte, opts SigningOpts) (SigningResult, error) {
	var hash []byte
	switch opts.Digest {
	case crypto.SHA256:
		sum := sha256.Sum256(payload)
		hash = sum[:]
	case crypto.SHA384:
		sum := sha512.Sum384(payload)
		hash = sum[:]
	case crypto.SHA512:
		sum := sha512.Sum512(payload)
		hash = sum[:]
	default:
		return SigningResult{}, errors.New("unsupported digest")
	}

	ecdsaPrivateKey, ok := opts.PrivateKey.(ecdsa.PrivateKey)
	if ok {
		sig, err := ecdsa.SignASN1(rand.Reader, &ecdsaPrivateKey, hash[:])
		if err == nil {
			return SigningResult{hex.EncodeToString(sig)}, nil
		}
	}

	rsaPrivateKey, ok := opts.PrivateKey.(rsa.PrivateKey)
	if ok {
		sig, err := rsa.SignPKCS1v15(rand.Reader, &rsaPrivateKey, opts.Digest, hash[:])
		if err == nil {
			return SigningResult{hex.EncodeToString(sig)}, nil
		}
	}

	return SigningResult{}, errors.New("unsupported algorithm")
}

// Create the string to sign.
func CreateStringToSign(canonicalRequest string, signerParams SignerParams) string {
	var stringToSignStrBuilder strings.Builder
	stringToSignStrBuilder.WriteString(signerParams.SigningAlgorithm)
	stringToSignStrBuilder.WriteString("\n")
	stringToSignStrBuilder.WriteString(signerParams.GetFormattedSigningDateTime())
	stringToSignStrBuilder.WriteString("\n")
	stringToSignStrBuilder.WriteString(signerParams.GetScope())
	stringToSignStrBuilder.WriteString("\n")
	stringToSignStrBuilder.WriteString(canonicalRequest)
	stringToSign := stringToSignStrBuilder.String()
	return stringToSign
}

// Create the canonical request.
func createCanonicalRequest(r *http.Request, body io.ReadSeeker, contentSha256 string) (string, string) {
	var canonicalRequestStrBuilder strings.Builder
	canonicalHeaderString, signedHeadersString := createCanonicalHeaderString(r)
	canonicalRequestStrBuilder.WriteString("POST")
	canonicalRequestStrBuilder.WriteString("\n")
	canonicalRequestStrBuilder.WriteString("/sessions")
	canonicalRequestStrBuilder.WriteString("\n")
	canonicalRequestStrBuilder.WriteString(createCanonicalQueryString(r, body))
	canonicalRequestStrBuilder.WriteString("\n")
	canonicalRequestStrBuilder.WriteString(canonicalHeaderString)
	canonicalRequestStrBuilder.WriteString("\n\n")
	canonicalRequestStrBuilder.WriteString(signedHeadersString)
	canonicalRequestStrBuilder.WriteString("\n")
	canonicalRequestStrBuilder.WriteString(contentSha256)
	canonicalRequestString := canonicalRequestStrBuilder.String()
	canonicalRequestStringHashBytes := sha256.Sum256([]byte(canonicalRequestString))
	return hex.EncodeToString(canonicalRequestStringHashBytes[:]), signedHeadersString
}

// Create the canonical query string.
func createCanonicalQueryString(r *http.Request, body io.ReadSeeker) string {
	rawQuery := strings.Replace(r.URL.Query().Encode(), "+", "%20", -1)
	return rawQuery
}

// Headers that aren't included in calculating the signature
var ignoredHeaderKeys = map[string]bool{
	"Authorization":   true,
	"User-Agent":      true,
	"X-Amzn-Trace-Id": true,
}

// Create the canonical header string.
func createCanonicalHeaderString(r *http.Request) (string, string) {
	var headers []string
	signedHeaderVals := make(http.Header)
	for k, v := range r.Header {
		canonicalKey := http.CanonicalHeaderKey(k)
		if ignoredHeaderKeys[canonicalKey] {
			continue
		}

		lowerCaseKey := strings.ToLower(k)
		if _, ok := signedHeaderVals[lowerCaseKey]; ok {
			// include additional values
			signedHeaderVals[lowerCaseKey] = append(signedHeaderVals[lowerCaseKey], v...)
			continue
		}

		headers = append(headers, lowerCaseKey)
		signedHeaderVals[lowerCaseKey] = v
	}
	sort.Strings(headers)

	headerValues := make([]string, len(headers))
	for i, k := range headers {
		headerValues[i] = k + ":" + strings.Join(signedHeaderVals[k], ",")
	}
	stripExcessSpaces(headerValues)
	return strings.Join(headerValues, "\n"), strings.Join(headers, ";")
}

const doubleSpace = "  "

// stripExcessSpaces will rewrite the passed in slice's string values to not
// contain muliple side-by-side spaces.
func stripExcessSpaces(vals []string) {
	var j, k, l, m, spaces int
	for i, str := range vals {
		// Trim trailing spaces
		for j = len(str) - 1; j >= 0 && str[j] == ' '; j-- {
		}

		// Trim leading spaces
		for k = 0; k < j && str[k] == ' '; k++ {
		}
		str = str[k : j+1]

		// Strip multiple spaces.
		j = strings.Index(str, doubleSpace)
		if j < 0 {
			vals[i] = str
			continue
		}

		buf := []byte(str)
		for k, m, l = j, j, len(buf); k < l; k++ {
			if buf[k] == ' ' {
				if spaces == 0 {
					// First space.
					buf[m] = buf[k]
					m++
				}
				spaces++
			} else {
				// End of multiple spaces.
				spaces = 0
				buf[m] = buf[k]
				m++
			}
		}

		vals[i] = string(buf[:m])
	}
}

// Calculate the hash of the request body
func calculateContentHash(r *http.Request, body io.ReadSeeker) string {
	hash := r.Header.Get(XAMZContentSHA256)

	if hash == "" {
		if body == nil {
			hash = emptyStringSHA256
		} else {
			hash = hex.EncodeToString(makeSha256Reader(body))
		}
	}

	return hash
}

// Find the SHA256 hash of the provided request body as a io.ReadSeeker
func makeSha256Reader(reader io.ReadSeeker) []byte {
	hash := sha256.New()
	start, _ := reader.Seek(0, 1)
	defer reader.Seek(start, 0)

	io.Copy(hash, reader)
	return hash.Sum(nil)
}

// Convert certificate to string, so that it can be present in the HTTP request header
func certificateToString(certificate x509.Certificate) string {
	return base64.StdEncoding.EncodeToString(certificate.Raw)
}

// Convert certificate chain to string, so that it can be pressent in the HTTP request header
func certificateChainToString(certificateChain []x509.Certificate) string {
	var x509ChainString strings.Builder
	for i, certificate := range certificateChain {
		x509ChainString.WriteString(certificateToString(certificate))
		if i != len(certificateChain)-1 {
			x509ChainString.WriteString(",")
		}
	}
	return x509ChainString.String()
}

type SignerParams struct {
	OverriddenDate   time.Time
	RegionName       string
	ServiceName      string
	SigningAlgorithm string
}

// Obtain the date-time, formatted as specified by SigV4
func (signerParams *SignerParams) GetFormattedSigningDateTime() string {
	return signerParams.OverriddenDate.UTC().Format(timeFormat)
}

// Obtain the short date-time, formatted as specified by SigV4
func (signerParams *SignerParams) GetFormattedShortSigningDateTime() string {
	return signerParams.OverriddenDate.UTC().Format(shortTimeFormat)
}

// Obtain the scope as part of the SigV4-X509 signature
func (signerParams *SignerParams) GetScope() string {
	var scopeStringBuilder strings.Builder
	scopeStringBuilder.WriteString(signerParams.GetFormattedShortSigningDateTime())
	scopeStringBuilder.WriteString("/")
	scopeStringBuilder.WriteString(signerParams.RegionName)
	scopeStringBuilder.WriteString("/")
	scopeStringBuilder.WriteString(signerParams.ServiceName)
	scopeStringBuilder.WriteString("/")
	scopeStringBuilder.WriteString("aws4_request")
	return scopeStringBuilder.String()
}
