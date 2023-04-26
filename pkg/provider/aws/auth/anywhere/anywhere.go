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
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/rolesanywhere-credential-helper/rolesanywhere"
)

// Options
type Options struct {
	PrivateKey     string
	Certificate    string
	ProfileArn     string
	TrustAnchorArn string
	RoleArn        string
	Region         string
	Endpoint       string
}

func Generate(opts Options) (*credentials.Credentials, error) {
	pk, err := readPrivateKeyData(opts.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unabl to read private key: %w", err)
	}
	certData, err := readCertificateData(opts.Certificate)
	if err != nil {
		return nil, err
	}
	certificateDerData, err := base64.StdEncoding.DecodeString(certData.CertificateData)
	if err != nil {
		return nil, err
	}
	certificate, err := x509.ParseCertificate([]byte(certificateDerData))
	if err != nil {
		return nil, err
	}
	var certificateChain []x509.Certificate

	config := aws.NewConfig().WithRegion(opts.Region)
	if opts.Endpoint != "" {
		config.WithEndpoint(opts.Endpoint)
	}
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	rolesAnywhereClient := rolesanywhere.New(sess, config)
	rolesAnywhereClient.Handlers.Build.RemoveByName("core.SDKVersionUserAgentHandler")
	rolesAnywhereClient.Handlers.Build.PushBackNamed(request.NamedHandler{Name: "v4x509.CredHelperUserAgentHandler", Fn: request.MakeAddToUserAgentHandler("eso-aws-auth", "v1", runtime.Version(), runtime.GOOS, runtime.GOARCH)})
	rolesAnywhereClient.Handlers.Sign.Clear()
	rolesAnywhereClient.Handlers.Sign.PushBackNamed(request.NamedHandler{Name: "v4x509.SignRequestHandler", Fn: CreateSignFunction(pk, *certificate, certificateChain)})

	output, err := rolesAnywhereClient.CreateSession(&rolesanywhere.CreateSessionInput{
		Cert:           &opts.Certificate,
		ProfileArn:     &opts.ProfileArn,
		TrustAnchorArn: &opts.TrustAnchorArn,
		RoleArn:        &opts.RoleArn,
	})
	if err != nil {
		return nil, err
	}
	return credentials.NewStaticCredentials(*output.CredentialSet[0].Credentials.AccessKeyId, *output.CredentialSet[0].Credentials.SecretAccessKey, *output.CredentialSet[0].Credentials.SessionToken), nil
}
