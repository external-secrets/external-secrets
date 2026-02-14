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

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type FakeSecretKeyResolver struct{}

func (fr *FakeSecretKeyResolver) Resolve(_ context.Context, _ kclient.Client, _, _ string, ref *esmeta.SecretKeySelector) (string, error) {
	switch ref.Name {
	case "Valid token auth":
		return "Valid", nil
	case "Valid mtls client certificate":
		const mockClientCertPEM = `-----BEGIN CERTIFICATE-----
MIICDDCCAXUCFBLEQBCxspRPCp8BfOtrifCv1B3SMA0GCSqGSIb3DQEBCwUAMEUx
CzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRl
cm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMjUxMjE3MTQ1NTQwWhcNMjYxMjE3MTQ1
NTQwWjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UE
CgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIGfMA0GCSqGSIb3DQEBAQUAA4GN
ADCBiQKBgQC2YZzXoQ4pHjVAHSJzs1g+J6LBkeBA5bRPEL3BZoPtxX0GhXgfc37c
FDpWH9DRfkcndwO29yh5Rrjdf24UES25HkTPrrGc6CICEsxHWvm00kgMU32SqVhD
dO3pwkEcbLzxNcu0xcfQO767lwT8j5BpESGTLmey1t1aHrHgTZ8DowIDAQABMA0G
CSqGSIb3DQEBCwUAA4GBAAV0XtV9GG8tk2Fz1Fy4hztyU17ZZccx3bYgPUrLo6b5
YFO8LRrvmLICMJwgeiy2VBDb5WAP34C4yN0jv5OQaI45bHMffud8ADkBSBM9RAvb
HMzKCq4wjntZHFhsu9u2OPOoU/Rey7EQhnsnO0w2oAbnjaqamAKL4uRuZQQHPtjo
-----END CERTIFICATE-----`
		return mockClientCertPEM, nil
	case "Valid mtls client key":
		const mockClientKeyPEM = `-----BEGIN PRIVATE KEY-----
MIICeAIBADANBgkqhkiG9w0BAQEFAASCAmIwggJeAgEAAoGBALZhnNehDikeNUAd
InOzWD4nosGR4EDltE8QvcFmg+3FfQaFeB9zftwUOlYf0NF+Ryd3A7b3KHlGuN1/
bhQRLbkeRM+usZzoIgISzEda+bTSSAxTfZKpWEN07enCQRxsvPE1y7TFx9A7vruX
BPyPkGkRIZMuZ7LW3VoeseBNnwOjAgMBAAECgYEAiw64GIzbECzRKzZLm24mHRX5
eZ+xHapGpXY9SGXSt4s5faxsX4afNkxSAnK1s9WViRisg2fFu1pZ/8B2fORwOAe8
VHAvRqsBTLZUKGR3Pm7S0zNGPcYw6X4HJi7cPDpdOUUBUy8Zg+dRcqMlHx4vaBmE
o0HqADbRjNiVmAebMoECQQDp2v2BwwVr68ugwqdb0HxihK862esPAWE69tg3D/iF
WLo7BdMVxMb/CBPVfE6tw+z8T0MzeRSYY7V2X5lccFfjAkEAx6bPv+brAHTcPlxc
T63AntlTm4yun+JfwTqjE+bajrJcRm8ij2Y15EFDWASoo7K0EqqAWbRUw2ReTNw1
2vTxQQJAFuP/sobzbefry7WiCiOzOTWBrYINNy/MY6gr69/dVLglqodcbSIQ1H/m
6Ru829d0yBG+Iziz4mLILWkYKus4PwJBAJ4kNHS17TkcV4QR1pDKeUOZs08HnR5J
yj0dPCU8e6wB/XNQ/lgFxvQ4+aXTctzPZTFP2oCzhVyLuOI6n3IDCMECQQCErbRe
lJZuvpUXsinpM3EfgB1NXmqTx3U4BTOJbNQqeXou7J/XbwO3TV37ARsP/iTqoh5S
oT11tLXIyFX0l2Ul
-----END PRIVATE KEY-----`
		return mockClientKeyPEM, nil
	default:
		return "", nil
	}
}
