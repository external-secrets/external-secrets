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

package keyvault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"encoding/base64"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"k8s.io/utils/pointer"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fake "github.com/external-secrets/external-secrets/pkg/provider/azure/keyvault/fake"
	utils "github.com/external-secrets/external-secrets/pkg/utils"
)

type secretManagerTestCase struct {
	mockClient      *fake.AzureMockClient
	secretName      string
	secretVersion   string
	serviceURL      string
	ref             *esv1beta1.ExternalSecretDataRemoteRef
	refFind         *esv1beta1.ExternalSecretFind
	apiErr          error
	setErr          error
	pushRef         esv1beta1.PushRemoteRef
	secretOutput    keyvault.SecretBundle
	setSecretOutput keyvault.SecretBundle
	keyOutput       keyvault.KeyBundle
	createKeyOutput keyvault.KeyBundle
	certOutput      keyvault.CertificateBundle
	importOutput    keyvault.CertificateBundle
	listOutput      keyvault.SecretListResultIterator
	expectError     string
	setValue        []byte
	expectedSecret  string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidSecretManagerTestCase() *secretManagerTestCase {
	secretString := "Hello World!"
	smtc := secretManagerTestCase{
		mockClient:     &fake.AzureMockClient{},
		secretName:     "MySecret",
		secretVersion:  "",
		ref:            makeValidRef(),
		refFind:        makeValidFind(),
		secretOutput:   keyvault.SecretBundle{Value: &secretString},
		serviceURL:     "",
		apiErr:         nil,
		expectError:    "",
		expectedSecret: secretString,
		expectedData:   map[string][]byte{},
	}

	smtc.mockClient.WithValue(smtc.serviceURL, smtc.secretName, smtc.secretVersion, smtc.secretOutput, smtc.apiErr)

	return &smtc
}

func makeValidSecretManagerTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}

	smtc.mockClient.WithValue(smtc.serviceURL, smtc.secretName, smtc.secretVersion, smtc.secretOutput, smtc.apiErr)
	smtc.mockClient.WithKey(smtc.serviceURL, smtc.secretName, smtc.secretVersion, smtc.keyOutput, smtc.apiErr)
	smtc.mockClient.WithCertificate(smtc.serviceURL, smtc.secretName, smtc.secretVersion, smtc.certOutput, smtc.apiErr)
	smtc.mockClient.WithList(smtc.serviceURL, smtc.listOutput, smtc.apiErr)
	smtc.mockClient.WithImportCertificate(smtc.importOutput, smtc.setErr)
	smtc.mockClient.WithImportKey(smtc.createKeyOutput, smtc.setErr)
	smtc.mockClient.WithSetSecret(smtc.setSecretOutput, smtc.setErr)

	return smtc
}

const (
	jwkPubRSA            = `{"kid":"ex","kty":"RSA","key_ops":["sign","verify","wrapKey","unwrapKey","encrypt","decrypt"],"n":"p2VQo8qCfWAZmdWBVaYuYb-a-tWWm78K6Sr9poCvNcmv8rUPSLACxitQWR8gZaSH1DklVkqz-Ed8Cdlf8lkDg4Ex5tkB64jRdC1Uvn4CDpOH6cp-N2s8hTFLqy9_YaDmyQS7HiqthOi9oVjil1VMeWfaAbClGtFt6UnKD0Vb_DvLoWYQSqlhgBArFJi966b4E1pOq5Ad02K8pHBDThlIIx7unibLehhDU6q3DCwNH_OOLx6bgNtmvGYJDd1cywpkLQ3YzNCUPWnfMBJRP3iQP_WI21uP6cvo0DqBPBM4wvVzHbCT0vnIflwkbgEWkq1FprqAitZlop9KjLqzjp9vyQ","e":"AQAB"}`
	jwkPubEC             = `{"kid":"https://example.vault.azure.net/keys/ec-p-521/e3d0e9c179b54988860c69c6ae172c65","kty":"EC","key_ops":["sign","verify"],"crv":"P-521","x":"AedOAtb7H7Oz1C_cPKI_R4CN_eai5nteY6KFW07FOoaqgQfVCSkQDK22fCOiMT_28c8LZYJRsiIFz_IIbQUW7bXj","y":"AOnchHnmBphIWXvanmMAmcCDkaED6ycW8GsAl9fQ43BMVZTqcTkJYn6vGnhn7MObizmkNSmgZYTwG-vZkIg03HHs"}`
	jsonTestString       = `{"Name": "External", "LastName": "Secret", "Address": { "Street": "Myroad st.", "CP": "J4K4T4" } }`
	jsonSingleTestString = `{"Name": "External", "LastName": "Secret" }`
	jsonTagTestString    = `{"tagname":"tagvalue","tagname2":"tagvalue2"}`
	keyName              = "key/keyname"
	certName             = "cert/certname"
	secretString         = "changedvalue"
	unexpectedError      = "[%d] unexpected error: %s, expected: '%s'"
	unexpectedSecretData = "[%d] unexpected secret data: expected %#v, got %#v"
	errorNoTag           = "tag something does not exist"
	something            = "something"
	tagname              = "tagname"
	tagname2             = "tagname2"
	tagvalue             = "tagvalue"
	tagvalue2            = "tagvalue2"
	secretName           = "example-1"
	testsecret           = "test-secret"
	fakeURL              = "noop"
	errStore             = "Azure.ValidateStore() error = %v, wantErr %v"
)

func getTagMap() map[string]*string {
	tag1 := "tagname"
	tag2 := "tagname2"
	value1 := "tagvalue"
	value2 := "tagvalue2"
	tagMap := make(map[string]*string)
	tagMap[tag1] = &value1
	tagMap[tag2] = &value2
	return tagMap
}

func newKVJWK(b []byte) *keyvault.JSONWebKey {
	var key keyvault.JSONWebKey
	err := json.Unmarshal(b, &key)
	if err != nil {
		panic(err)
	}
	return &key
}

type fakeRef struct {
	key string
}

func (f fakeRef) GetRemoteKey() string {
	return f.key
}

func TestAzureKeyVaultSetSecret(t *testing.T) {
	p12Cert, _ := base64.StdEncoding.DecodeString("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")
	goodKey, _ := base64.StdEncoding.DecodeString("LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUpRZ0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQ1N3d2dna29BZ0VBQW9JQ0FRQ1pITzRvNkpteU9aZGYKQXQ3RFdqR2tHdzdENVVIU1BHZXQyTjg2cnBGWXcrZThnL3dSeDBnZDBzRk9pelBBREdjcnpmdWE5Z3ZFcDRWcwpXb2FHbmN3UXhqdnMrZ1orWmQ2UkVPNHRLNzRURmYxaWZibmowUHE2OENlQlFpaG8xbDNwM2UwQy8yemVJMjNiCnZWRHZlMm13VXE5aDY4UTFFUmdWMU1LaWJHU1Naak5DQzdkRGFQWmpKazViMFlWVFdxREViemREVnh2ZVVMNVIKcUZnL0RKQTMzVnE2VFQzQ2U5RjBIcEorb3graSs4cUxmWU5qZExSUDZlbEtLTU5naVhhNTFvdnQ5MjF4UkVGdgpYRXYvTUtqWTlhNkppNndIRSs0NmdvbFY4V2puK2xMRkRKVHh6WEFEN2p2NzVzaHY0WEczdFlaQ2J4cTMzZ2JtCm96c0VQZ3lTRGtCMm5zc0tIUEFhSVNPaWpjNDhiSXhwbDVocFJPWUZFblJDWnhablhQNjdLZVF1VWZXQkpoVWcKYWltc0JRK3p6cFB6ZjVUbjRnVExkWll2NU41V1V2djJJdUF5Qktha0ZhR1ZYTzFpZ2FDeVQvUTNBcEE2ZGx4Sgo1VW44SzY4dS9KSGFmWWZ5engwVnVoZk5zbmtiWkxWSEZsR2Rxd3JrU0tCWSs1eS9WWlpkeC9hSHNWWndVN3ZECmNlaGxlWlFNNGV2cm5tMUY3dk5xSHBUK3BHSnpNVWVUNGZMVFpabTBra1Y3ZXl5RGRMMDFEWXRXQk1TM2NEb1EKdU5vWElBMCtDeFZPOHcxcC9wbXF2UFQ3cmpad2pwYkVMUkp3MWs4R3ozU2FKb2VqaFBzWC9xNzNGWWdBc09PRApwTXJuK3ZpU2U0ZnJmR0VmZlEvYXJUVE5qK1BUb3dJREFRQUJBb0lDQUM3ek1CUmJQc1huNHdLL1hvK0ltTEE1Cm04MTEvemo0VE5LQ0xmRlFsa0VoMFcxOUMwNW9UVFRYNjI2cVFMUWpHWC9WS2RIYW9NRXNuVDBjaFNQQ1AxRGwKZUhxeU1FdVI4UzJLZzM1V2EzSnV5OFBueVppUi9GQldVOGJQQXBVakpxa1A1QjJITlZyb2drZGZSZklwWmI4cgptNXZyTDc4Vi9zeXk4UHZkUVBtalhSUmpnMDZvWU9VR1dnRE52cFJRdGZ1R0h1d0hTZ1JodmZwTUpNTXdsd2lLClY4Zkk1NmM3VUg3SzRTRHo1RCtWOWdYUDl2b0lUMEl4OTlkRnFLTnhnM1o0MDIrazcycE1BOFNpQ0t1M3dBN0gKUnozbUZsb1ZRbmV1ajI1TEdHQUo0bGVLQkNJaFhMZlgxWXpvdDQyWEU4ZkJZZW45SjdRNTRPUFlLY0NqUmpjSgp1M2NkamtIbmFWVFc1dDdLTDFuYVAxRmF0S0ZxSjY1V1Y0c3pxWDhPVkpzbWhLalNsNUhqTk1VeERuaFUraWRTCmsxaGNaa00zOWd2RGR1ekRHeHF0L2hHMWNJS3VtamxZb01WNDV4VWFoVHdhTjZnamlrTUxNdFgrb2c0MVAxU3cKa09hZTZ4enJFQmU1eXhqSnVDWFJzK2FFOXZhTmpIWmpnSTNKREJ0enNjeCtvRFZBMXoxWVBpR2t1NXBNYmxYUQpFMWlRQnlJOVRjeHMrazN0NWdIQ0d3Z2lOcXVnOVZJaXY1cTQ2R2VGRVdnQS8wZ2hEZ0hIRnNRSDJ4VEpGU2d6ClluTkRVNlZtQ1RYZEQ0QU5jS085Z0loQzdxYk9iazlUeS9zZkZIQjBrYUdCVjFFZGZ3a0R4LytYdXRacHNUN3IKdkl6SUVDd2JPTEUzZCtLb1grUUJBb0lCQVFESG9SVU42U1VmQ3I4Z2FsOFM3UDhscU1kZnhsQVNRcWRqOHY2WAp3V1o1MFJKVE9TRmxqN3dlb2FnTStDT3pEUHpoU3pMbE4vcVdnK2h1RFJGcXBWb08xTmlrZVdvZEVwajFyZG5qCmlLeFlEVUJKNjFCMk5GT3R6Qm9CZUgyOFpDR3dRUW93clZSNUh5dUlqOTRhTzBiRlNUWEJTdWx2d3NQeDZhR2cKaTV2Q0VITHB6ODZKV1BzcjYwSmxVSDk2Z2U3NXJNZEFuRTJ1UE5JVlRnR2grMHpOenZ2a21yZHRYRVR4QXpFZwo5d0RaNVFZTUNYTGVjV0RxaWtmQUpoaUFJTjdVWEtvajN0b1ZMMzh6Sm95WmNWT3ZLaVRIQXY1MCtyNGhVTzhiCjJmL1J2VllKMngybnJuSVR4L0s2Y2N3UUttb1dFNmJRdmg4SXJGTEI3aWN2cVJzUEFvSUJBUURFV1VGemRyRHgKN2w4VGg2bVV5ZlBIWWtOUU0vdDBqM3l3RDROQ2JuSlEvZGd2OGNqMVhGWTNkOWptdWZreGtrQ01WVC8rcVNrOQp1cm1JVVJDeGo5ZDJZcUtMYXZVcUVFWCtNVStIZ0VDOW4yTHluN0xXdVNyK2dFWVdkNllXUVNSVXpoS0xaN2RUCnliTnhmcnNtczNFSVJEZTkwcFV4ZGJ0eWpJSTlZd1NaRDdMUHVOQmc1cWNaTW1xWG9vSnQxdnJld1JINncwam8KM1pxTWMrVGFtNGxYc0xmU0pqTlAzd2IzZEE0ZDFvWWFIb29WWTVyK0dER1F5YnVKYllQZSt6d01NTkJhZ2dTVQpCL3J5NlBldVBTWVJnby9kTlR2TERDamJjbytXdFpncjRJaWxCVmpCbmwycEhzakVHYjZDV2Q2bXZCdlk3SWM5ClM3cXJLUGQrWE00dEFvSUJBR08wRkN2cWNkdmJKakl1Ym1XcGNKV0NnbkZYUHM2Zjg3Sjd2cVJVdDdYSHNmdFcKNFZNMFFxU1o0TEQ1amZyelZhbkFRUjh5b2psaWtFZkd4eGdZbGE0cXFEa2RXdDVDVjVyOHhZSmExSmoxcFZKRgo4TjNZcktKMCtkZ2FNZEpSd0hHalNrK2RnajhzVGpYYWhQZGMrNisxTE4vcFprV25aTzRCM2ZPdFJwSGFYVXBoCnU2bmxneTBnUnYwTEEyQlFYT2JlWUhYb212T1c5T1luRzdHbkxXanRJK205VERlV2llaEZ5OWZIQmVuTjlRTTIKQk9VTWczY2dzVTFLdVpuazBPWUhrZ0p3WDBPTmdWNHV0ckk4WTZ0c3hRbVFlVDQ3clpJK05lNFhKeW0rQXFiUgpoVEltY2x0bTFkaEExY2FOS0liMk1hNjRCZy95NFRKeW02ZTJNZ2tDZ2dFQkFKTGt5NmljVllqSjh1dGpoU1ZCCmFWWHpWN1M3RHhhRytwdWxIMmdseFBSKzFLd1owV1J1N2ptVk9mcHppOURnUDlZOU9TRkdZUXBEbGVZNzc2ZEgKbThSL3ltZFBYNWRXa1dhNGNXMUlNQ2N0QlJQTEVqcStVVUlScVYzSnFjSGdmbFBMeitmbmNpb0hMbTVzaDR0TwpsL085Ulk2SDZ3SVR1R2JjWTl1VkpxMTBKeXhzY2NqdEJubzlVNjJaOE1aSUhXdGxPaFJHNFZjRjQwZk10Snd2CjNMSjBEVEgxVGxJazRzdGlVZVZVeHdMbmNocktaL3hORVZmbTlKeStCL2hjTVBKVjJxcTd0cjBnczBmanJ0ajEKK25NRElLbzMxMEh6R09ZRWNSUXBTMjBZRUdLVSsyL3ZFTmNqcHNPL0Z0M2lha2FIV0xZVFRxSTI4N0oxZGFOZAp2d2tDZ2dFQUNqWTJIc0ErSlQvWlU1Q0k1NlFRNmlMTkdJeFNUYkxUMGJNbGNWTDJraGFFNTRMVGtld0I5enFTCk5xNVFacUhxbGk2anZiKzM4Q1FPUWxPWmd6clVtZlhIemNWQ1FwMUk1RjRmSGkyWUVVa3FJL2dWdlVGMUxCNUUKZE1KR1FZa3Jick83Qjc0eE50RUV3Mmh3UFUwcTRmby92eFZXV0pFdTNoMGpSL0llMDA3UGtPZ0p1K1R5ZWZBNwpQVkM4OFlQbmsyZ3ArUFpRdDljanhOL0V4enRweDZ4cUJzT0MvQWZIYU5BdFA0azM5MVc5NjN3eHVwbUE5SkdiCk4yM0NCRmVIZDJmTUViTWJuWDk1Q1NYNjNJVWNaNVRhZTdwQS9OZ094YkdzaGRSMHdFZldTMGNyT1VTdGt6aE0KT3lCekNZSk53d3Bld3cyOFpIMGgybHh6VVRHWStRPT0KLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo=")
	goodSecret := "old"
	typeNotSupported := func(smtc *secretManagerTestCase) {
		smtc.pushRef = fakeRef{
			key: "badtype/secret",
		}
		smtc.expectError = "secret type badtype not supported"
	}
	secretSuccess := func(smtc *secretManagerTestCase) {
		smtc.setValue = []byte("secret")
		smtc.pushRef = fakeRef{
			key: secretName,
		}
		smtc.secretOutput = keyvault.SecretBundle{
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("external-secrets"),
			},
			Value: &goodSecret,
		}
	}
	secretNoChange := func(smtc *secretManagerTestCase) {
		smtc.setValue = []byte(goodSecret)
		smtc.pushRef = fakeRef{
			key: secretName,
		}
		smtc.secretOutput = keyvault.SecretBundle{
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("external-secrets"),
			},
			Value: &goodSecret,
		}
	}
	secretWrongTags := func(smtc *secretManagerTestCase) {
		smtc.setValue = []byte(goodSecret)
		smtc.pushRef = fakeRef{
			key: secretName,
		}
		smtc.secretOutput = keyvault.SecretBundle{
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("nope"),
			},
			Value: &goodSecret,
		}
		smtc.expectError = "secret example-1 is not managed by external-secrets"
	}
	secretNoTags := func(smtc *secretManagerTestCase) {
		smtc.setValue = []byte(goodSecret)
		smtc.pushRef = fakeRef{
			key: secretName,
		}
		smtc.secretOutput = keyvault.SecretBundle{
			Tags:  map[string]*string{},
			Value: &goodSecret,
		}
		smtc.expectError = "secret example-1 is not managed by external-secrets"
	}
	secretNotFound := func(smtc *secretManagerTestCase) {
		smtc.setValue = []byte(goodSecret)
		smtc.pushRef = fakeRef{
			key: secretName,
		}
		smtc.apiErr = autorest.DetailedError{StatusCode: 404, Method: "GET", Message: "Not Found"}
	}
	failedGetSecret := func(smtc *secretManagerTestCase) {
		smtc.setValue = []byte(goodSecret)
		smtc.pushRef = fakeRef{
			key: secretName,
		}
		smtc.apiErr = autorest.DetailedError{StatusCode: 403, Method: "GET", Message: "Forbidden"}
		smtc.expectError = "could not get secret example-1: #GET: Forbidden: StatusCode=403"
	}
	failedNotParseableError := func(smtc *secretManagerTestCase) {
		smtc.setValue = []byte(goodSecret)
		smtc.pushRef = fakeRef{
			key: secretName,
		}
		smtc.apiErr = fmt.Errorf("Crash!")
		smtc.expectError = "could not get secret example-1: could not parse error: Crash!"
	}
	failedSetSecret := func(smtc *secretManagerTestCase) {
		smtc.setValue = []byte(goodSecret)
		smtc.pushRef = fakeRef{
			key: secretName,
		}
		smtc.apiErr = autorest.DetailedError{StatusCode: 404, Method: "GET", Message: "Not Found"}
		smtc.setErr = autorest.DetailedError{StatusCode: 403, Method: "POST", Message: "Forbidden"}
		smtc.expectError = "could not set secret example-1: #POST: Forbidden: StatusCode=403"
	}
	keySuccess := func(smtc *secretManagerTestCase) {
		smtc.setValue = goodKey
		smtc.pushRef = fakeRef{
			key: keyName,
		}
		smtc.keyOutput = keyvault.KeyBundle{
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("external-secrets"),
			},
			Key: &keyvault.JSONWebKey{},
		}
	}
	noTags := func(smtc *secretManagerTestCase) {
		smtc.setValue = goodKey
		smtc.pushRef = fakeRef{
			key: keyName,
		}
		smtc.keyOutput = keyvault.KeyBundle{
			Tags: map[string]*string{},
			Key:  &keyvault.JSONWebKey{},
		}
		smtc.expectError = "key not managed by external-secrets"
	}
	wrongTags := func(smtc *secretManagerTestCase) {
		smtc.setValue = goodKey
		smtc.pushRef = fakeRef{
			key: keyName,
		}
		smtc.keyOutput = keyvault.KeyBundle{
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("internal-secrets"),
			},
			Key: &keyvault.JSONWebKey{},
		}
		smtc.expectError = "key not managed by external-secrets"
	}
	invalidKey := func(smtc *secretManagerTestCase) {
		invalid := []byte("nope")
		smtc.setValue = invalid
		smtc.pushRef = fakeRef{
			key: keyName,
		}
		smtc.expectError = "could not load private key keyname: format not compatible with PCKS8"
	}
	errorGetKey := func(smtc *secretManagerTestCase) {
		smtc.setValue = goodKey
		smtc.pushRef = fakeRef{
			key: keyName,
		}
		smtc.apiErr = autorest.DetailedError{StatusCode: 403, Method: "GET", Message: "Forbidden"}
		smtc.expectError = "could not get key keyname: #GET: Forbidden: StatusCode=403"
	}
	keyNotFound := func(smtc *secretManagerTestCase) {
		smtc.setValue = goodKey
		smtc.pushRef = fakeRef{
			key: keyName,
		}
		smtc.apiErr = autorest.DetailedError{StatusCode: 404, Method: "GET", Message: "Not Found"}
		smtc.expectError = ""
	}
	importKeyFailed := func(smtc *secretManagerTestCase) {
		smtc.setValue = goodKey
		smtc.pushRef = fakeRef{
			key: keyName,
		}
		smtc.apiErr = autorest.DetailedError{StatusCode: 404, Method: "GET", Message: "Not Found"}
		smtc.setErr = autorest.DetailedError{StatusCode: 403, Method: "POST", Message: "Forbidden"}
		smtc.expectError = "could not import key keyname: #POST: Forbidden: StatusCode=403"
	}
	certP12Success := func(smtc *secretManagerTestCase) {
		smtc.setValue = p12Cert
		smtc.pushRef = fakeRef{
			key: certName,
		}
		smtc.certOutput = keyvault.CertificateBundle{
			X509Thumbprint: pointer.StringPtr("123"),
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("external-secrets"),
			},
		}
	}
	certPEMSuccess := func(smtc *secretManagerTestCase) {
		pemCert, _ := base64.StdEncoding.DecodeString("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZwekNDQTQrZ0F3SUJBZ0lVTUhhVDZtZG8vd2Urbit0NFB2R0JZaUdDSXE0d0RRWUpLb1pJaHZjTkFRRUwKQlFBd1l6RUxNQWtHQTFVRUJoTUNRVlV4RXpBUkJnTlZCQWdNQ2xOdmJXVXRVM1JoZEdVeElUQWZCZ05WQkFvTQpHRWx1ZEdWeWJtVjBJRmRwWkdkcGRITWdVSFI1SUV4MFpERWNNQm9HQTFVRUF3d1RZVzV2ZEdobGNpMW1iMjh0ClltRnlMbU52YlRBZUZ3MHlNakEyTURreE56UTFNelphRncweU16QTJNRGt4TnpRMU16WmFNR014Q3pBSkJnTlYKQkFZVEFrRlZNUk13RVFZRFZRUUlEQXBUYjIxbExWTjBZWFJsTVNFd0h3WURWUVFLREJoSmJuUmxjbTVsZENCWAphV1JuYVhSeklGQjBlU0JNZEdReEhEQWFCZ05WQkFNTUUyRnViM1JvWlhJdFptOXZMV0poY2k1amIyMHdnZ0lpCk1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQ0R3QXdnZ0lLQW9JQ0FRQ1pITzRvNkpteU9aZGZBdDdEV2pHa0d3N0QKNVVIU1BHZXQyTjg2cnBGWXcrZThnL3dSeDBnZDBzRk9pelBBREdjcnpmdWE5Z3ZFcDRWc1dvYUduY3dReGp2cworZ1orWmQ2UkVPNHRLNzRURmYxaWZibmowUHE2OENlQlFpaG8xbDNwM2UwQy8yemVJMjNidlZEdmUybXdVcTloCjY4UTFFUmdWMU1LaWJHU1Naak5DQzdkRGFQWmpKazViMFlWVFdxREViemREVnh2ZVVMNVJxRmcvREpBMzNWcTYKVFQzQ2U5RjBIcEorb3graSs4cUxmWU5qZExSUDZlbEtLTU5naVhhNTFvdnQ5MjF4UkVGdlhFdi9NS2pZOWE2SgppNndIRSs0NmdvbFY4V2puK2xMRkRKVHh6WEFEN2p2NzVzaHY0WEczdFlaQ2J4cTMzZ2Jtb3pzRVBneVNEa0IyCm5zc0tIUEFhSVNPaWpjNDhiSXhwbDVocFJPWUZFblJDWnhablhQNjdLZVF1VWZXQkpoVWdhaW1zQlErenpwUHoKZjVUbjRnVExkWll2NU41V1V2djJJdUF5Qktha0ZhR1ZYTzFpZ2FDeVQvUTNBcEE2ZGx4SjVVbjhLNjh1L0pIYQpmWWZ5engwVnVoZk5zbmtiWkxWSEZsR2Rxd3JrU0tCWSs1eS9WWlpkeC9hSHNWWndVN3ZEY2VobGVaUU00ZXZyCm5tMUY3dk5xSHBUK3BHSnpNVWVUNGZMVFpabTBra1Y3ZXl5RGRMMDFEWXRXQk1TM2NEb1F1Tm9YSUEwK0N4Vk8KOHcxcC9wbXF2UFQ3cmpad2pwYkVMUkp3MWs4R3ozU2FKb2VqaFBzWC9xNzNGWWdBc09PRHBNcm4rdmlTZTRmcgpmR0VmZlEvYXJUVE5qK1BUb3dJREFRQUJvMU13VVRBZEJnTlZIUTRFRmdRVWJPQk14azJ5UkNkR1N4eEZGMzBUCkZORFhHS3N3SHdZRFZSMGpCQmd3Rm9BVWJPQk14azJ5UkNkR1N4eEZGMzBURk5EWEdLc3dEd1lEVlIwVEFRSC8KQkFVd0F3RUIvekFOQmdrcWhraUc5dzBCQVFzRkFBT0NBZ0VBQXdudUtxOThOQ2hUMlUzU2RSNEFVem1MTjFCVwowNHIwMTA3TjlKdW9LbzJycjhoZ21mRmd0MDgrdFNDYzR5ajZSNStyY1hudXpqeEZLaWJVYnFncFpvd0pSSGEyCjF0NUJicEwxeWcybGZyZnhIb3YvRjh0VnNTbUE4d3loNlVpV1J3RTlrdlBXUm5LblR1a3Y1enpzcVNsTlNpbG0KNDl6UTdTV05sK0lBRnkvc3dacnRKUTEwVlQ5czRuUGVHM29XUU1vdE9QUCtsbFNpeW5LTFpxUTRnU0tSaTNmZQpQTGlXcHQ5WGZYb0dVQ0VqN3E1cGhibExQZ2RLVUNyaEdQMW4yalltWHNjV0xNeWtBbmEyMGNobHJxVlluQ2E4CkpVcDRMZnRGRHA4OVlUb1hPRkhuRm1uTkN2Y0lyRGZGeURmaGw0VU1GcEswT1VLcVRUeFdhSzl1cU9JcGFySXMKS1l3c3ArZkxlV0xiUTZrR2Ztbk81aURSZCtvT2hyTllvb1RaVks5ZlFSNXJEMmU0QitlYTByelFGWEFBVWpKNQpPWGFieGJEclErT01landjNEhxcXN4enRKZ0QyYVAyZUsyL0w1UFdQdWcwRSsxZzhBQlpmVmJvaC9NM01IZ2J6ClBnYVRxZ3V6R0Zka0czRVh1K09oR2JVMC8rNzdWTW5aaTJJUVpuL2F3R1VhN1grTVAwQkR2alZZNWtWcE1aMWgKYzJDbERqZ3hOc0xHdGlrTzRjV2I1c1FSUjJHWU0zZE1rNTBWUWN0SjVScXNSczZwT0NYRFhFM1JlVlFqNGhOQgplV3ZhRFdRMktteU9haTU1ZGJEcmxKK251ODNPbUNwNTlSelA1azU4WmFEWG5sQzM4VXdUdDBxMUQ3K3pGMHRzCjFHOTMydUVCSFdZSHVPQT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=")
		smtc.setValue = pemCert
		smtc.pushRef = fakeRef{
			key: certName,
		}
		smtc.certOutput = keyvault.CertificateBundle{
			X509Thumbprint: pointer.StringPtr("123"),
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("external-secrets"),
			},
		}
	}

	certDERSuccess := func(smtc *secretManagerTestCase) {
		derCert, _ := base64.StdEncoding.DecodeString("MIIFpzCCA4+gAwIBAgIUMHaT6mdo/we+n+t4PvGBYiGCIq4wDQYJKoZIhvcNAQELBQAwYzELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDEcMBoGA1UEAwwTYW5vdGhlci1mb28tYmFyLmNvbTAeFw0yMjA2MDkxNzQ1MzZaFw0yMzA2MDkxNzQ1MzZaMGMxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQxHDAaBgNVBAMME2Fub3RoZXItZm9vLWJhci5jb20wggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCZHO4o6JmyOZdfAt7DWjGkGw7D5UHSPGet2N86rpFYw+e8g/wRx0gd0sFOizPADGcrzfua9gvEp4VsWoaGncwQxjvs+gZ+Zd6REO4tK74TFf1ifbnj0Pq68CeBQiho1l3p3e0C/2zeI23bvVDve2mwUq9h68Q1ERgV1MKibGSSZjNCC7dDaPZjJk5b0YVTWqDEbzdDVxveUL5RqFg/DJA33Vq6TT3Ce9F0HpJ+ox+i+8qLfYNjdLRP6elKKMNgiXa51ovt921xREFvXEv/MKjY9a6Ji6wHE+46golV8Wjn+lLFDJTxzXAD7jv75shv4XG3tYZCbxq33gbmozsEPgySDkB2nssKHPAaISOijc48bIxpl5hpROYFEnRCZxZnXP67KeQuUfWBJhUgaimsBQ+zzpPzf5Tn4gTLdZYv5N5WUvv2IuAyBKakFaGVXO1igaCyT/Q3ApA6dlxJ5Un8K68u/JHafYfyzx0VuhfNsnkbZLVHFlGdqwrkSKBY+5y/VZZdx/aHsVZwU7vDcehleZQM4evrnm1F7vNqHpT+pGJzMUeT4fLTZZm0kkV7eyyDdL01DYtWBMS3cDoQuNoXIA0+CxVO8w1p/pmqvPT7rjZwjpbELRJw1k8Gz3SaJoejhPsX/q73FYgAsOODpMrn+viSe4frfGEffQ/arTTNj+PTowIDAQABo1MwUTAdBgNVHQ4EFgQUbOBMxk2yRCdGSxxFF30TFNDXGKswHwYDVR0jBBgwFoAUbOBMxk2yRCdGSxxFF30TFNDXGKswDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAgEAAwnuKq98NChT2U3SdR4AUzmLN1BW04r0107N9JuoKo2rr8hgmfFgt08+tSCc4yj6R5+rcXnuzjxFKibUbqgpZowJRHa21t5BbpL1yg2lfrfxHov/F8tVsSmA8wyh6UiWRwE9kvPWRnKnTukv5zzsqSlNSilm49zQ7SWNl+IAFy/swZrtJQ10VT9s4nPeG3oWQMotOPP+llSiynKLZqQ4gSKRi3fePLiWpt9XfXoGUCEj7q5phblLPgdKUCrhGP1n2jYmXscWLMykAna20chlrqVYnCa8JUp4LftFDp89YToXOFHnFmnNCvcIrDfFyDfhl4UMFpK0OUKqTTxWaK9uqOIparIsKYwsp+fLeWLbQ6kGfmnO5iDRd+oOhrNYooTZVK9fQR5rD2e4B+ea0rzQFXAAUjJ5OXabxbDrQ+OMejwc4HqqsxztJgD2aP2eK2/L5PWPug0E+1g8ABZfVboh/M3MHgbzPgaTqguzGFdkG3EXu+OhGbU0/+77VMnZi2IQZn/awGUa7X+MP0BDvjVY5kVpMZ1hc2ClDjgxNsLGtikO4cWb5sQRR2GYM3dMk50VQctJ5RqsRs6pOCXDXE3ReVQj4hNBeWvaDWQ2KmyOai55dbDrlJ+nu83OmCp59RzP5k58ZaDXnlC38UwTt0q1D7+zF0ts1G932uEBHWYHuOA=")
		smtc.setValue = derCert
		smtc.pushRef = fakeRef{
			key: certName,
		}
		smtc.certOutput = keyvault.CertificateBundle{
			X509Thumbprint: pointer.StringPtr("123"),
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("external-secrets"),
			},
		}
	}

	certImportCertificateError := func(smtc *secretManagerTestCase) {
		smtc.setErr = errors.New("error")
		smtc.setValue = p12Cert
		smtc.pushRef = fakeRef{
			key: certName,
		}
		smtc.certOutput = keyvault.CertificateBundle{
			X509Thumbprint: pointer.StringPtr("123"),
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("external-secrets"),
			},
		}
		smtc.expectError = "could not import certificate certname: error"
	}

	certFingerprintMatches := func(smtc *secretManagerTestCase) {
		smtc.setErr = errors.New("error")
		smtc.setValue = p12Cert
		smtc.pushRef = fakeRef{
			key: certName,
		}
		smtc.certOutput = keyvault.CertificateBundle{
			X509Thumbprint: pointer.StringPtr("ilZxcWgYWs3WodyrZQAAsliFtB4"),
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("external-secrets"),
			},
		}
	}

	certNotManagedByES := func(smtc *secretManagerTestCase) {
		smtc.setValue = p12Cert
		smtc.pushRef = fakeRef{
			key: certName,
		}
		smtc.certOutput = keyvault.CertificateBundle{
			X509Thumbprint: pointer.StringPtr("123"),
			Tags: map[string]*string{
				"managed-by": pointer.StringPtr("foobar"),
			},
		}
		smtc.expectError = "certificate not managed by external-secrets"
	}

	certNoManagerTags := func(smtc *secretManagerTestCase) {
		smtc.setValue = p12Cert
		smtc.pushRef = fakeRef{
			key: certName,
		}
		smtc.certOutput = keyvault.CertificateBundle{
			X509Thumbprint: pointer.StringPtr("123"),
		}
		smtc.expectError = "certificate not managed by external-secrets"
	}

	certNotACertificate := func(smtc *secretManagerTestCase) {
		smtc.setValue = []byte("foobar")
		smtc.pushRef = fakeRef{
			key: certName,
		}
		smtc.certOutput = keyvault.CertificateBundle{
			X509Thumbprint: pointer.StringPtr("123"),
		}
		smtc.expectError = "value from secret is not a valid certificate: x509: malformed certificate"
	}

	certNoPermissions := func(smtc *secretManagerTestCase) {
		smtc.apiErr = autorest.DetailedError{
			StatusCode: 403,
			Method:     "GET",
			Message:    "Insufficient Permissions",
		}
		smtc.setValue = p12Cert
		smtc.pushRef = fakeRef{
			key: certName,
		}
		smtc.certOutput = keyvault.CertificateBundle{
			X509Thumbprint: pointer.StringPtr("123"),
		}
		smtc.expectError = "could not get certificate from keyvault: #GET: Insufficient Permissions: StatusCode=403"
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(certP12Success),
		makeValidSecretManagerTestCaseCustom(certPEMSuccess),
		makeValidSecretManagerTestCaseCustom(certDERSuccess),
		makeValidSecretManagerTestCaseCustom(certImportCertificateError),
		makeValidSecretManagerTestCaseCustom(certFingerprintMatches),
		makeValidSecretManagerTestCaseCustom(certNotManagedByES),
		makeValidSecretManagerTestCaseCustom(certNoManagerTags),
		makeValidSecretManagerTestCaseCustom(certNotACertificate),
		makeValidSecretManagerTestCaseCustom(certNoPermissions),
		makeValidSecretManagerTestCaseCustom(keySuccess),
		makeValidSecretManagerTestCaseCustom(invalidKey),
		makeValidSecretManagerTestCaseCustom(errorGetKey),
		makeValidSecretManagerTestCaseCustom(keyNotFound),
		makeValidSecretManagerTestCaseCustom(importKeyFailed),
		makeValidSecretManagerTestCaseCustom(noTags),
		makeValidSecretManagerTestCaseCustom(wrongTags),
		makeValidSecretManagerTestCaseCustom(secretSuccess),
		makeValidSecretManagerTestCaseCustom(secretNoChange),
		makeValidSecretManagerTestCaseCustom(secretWrongTags),
		makeValidSecretManagerTestCaseCustom(secretNoTags),
		makeValidSecretManagerTestCaseCustom(secretNotFound),
		makeValidSecretManagerTestCaseCustom(failedGetSecret),
		makeValidSecretManagerTestCaseCustom(failedNotParseableError),
		makeValidSecretManagerTestCaseCustom(failedSetSecret),
		makeValidSecretManagerTestCaseCustom(typeNotSupported),
	}

	sm := Azure{
		provider: &esv1beta1.AzureKVProvider{VaultURL: pointer.StringPtr(fakeURL)},
	}
	for k, v := range successCases {
		sm.baseClient = v.mockClient
		err := sm.SetSecret(context.Background(), v.setValue, v.pushRef)
		if !utils.ErrorContains(err, v.expectError) {
			if err == nil {
				t.Errorf("[%d] unexpected error: <nil>, expected: '%s'", k, v.expectError)
			} else {
				t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
			}
		}
	}

}

// test the sm<->azurekv interface
// make sure correct values are passed and errors are handled accordingly.
func TestAzureKeyVaultSecretManagerGetSecret(t *testing.T) {
	secretString := "changedvalue"
	secretCertificate := "certificate_value"
	tagMap := getTagMap()

	// good case
	setSecretString := func(smtc *secretManagerTestCase) {
		smtc.expectedSecret = secretString
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString,
		}
	}

	setSecretStringWithVersion := func(smtc *secretManagerTestCase) {
		smtc.expectedSecret = secretString
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString,
		}
		smtc.ref.Version = "v1"
		smtc.secretVersion = smtc.ref.Version
	}

	setSecretWithProperty := func(smtc *secretManagerTestCase) {
		jsonString := jsonTestString
		smtc.expectedSecret = "External"
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &jsonString,
		}
		smtc.ref.Property = "Name"
	}

	badSecretWithProperty := func(smtc *secretManagerTestCase) {
		jsonString := jsonTestString
		smtc.expectedSecret = ""
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &jsonString,
		}
		smtc.ref.Property = "Age"
		smtc.expectError = fmt.Sprintf("property %s does not exist in key %s", smtc.ref.Property, smtc.ref.Key)
		smtc.apiErr = errors.New(smtc.expectError)
	}

	// // good case: key set
	setPubRSAKey := func(smtc *secretManagerTestCase) {
		smtc.secretName = keyName
		smtc.expectedSecret = jwkPubRSA
		smtc.keyOutput = keyvault.KeyBundle{
			Key: newKVJWK([]byte(jwkPubRSA)),
		}
		smtc.ref.Key = smtc.secretName
	}

	// // good case: key set
	setPubECKey := func(smtc *secretManagerTestCase) {
		smtc.secretName = keyName
		smtc.expectedSecret = jwkPubEC
		smtc.keyOutput = keyvault.KeyBundle{
			Key: newKVJWK([]byte(jwkPubEC)),
		}
		smtc.ref.Key = smtc.secretName
	}

	// // good case: key set
	setCertificate := func(smtc *secretManagerTestCase) {
		byteArrString := []byte(secretCertificate)
		smtc.secretName = certName
		smtc.expectedSecret = secretCertificate
		smtc.certOutput = keyvault.CertificateBundle{
			Cer: &byteArrString,
		}
		smtc.ref.Key = smtc.secretName
	}

	badSecretType := func(smtc *secretManagerTestCase) {
		smtc.secretName = "name"
		smtc.expectedSecret = ""
		smtc.expectError = fmt.Sprintf("unknown Azure Keyvault object Type for %s", smtc.secretName)
		smtc.ref.Key = fmt.Sprintf("dummy/%s", smtc.secretName)
	}

	setSecretWithTag := func(smtc *secretManagerTestCase) {
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.ref.Property = tagname
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString, Tags: tagMap,
		}
		smtc.expectedSecret = tagvalue
	}

	badSecretWithTag := func(smtc *secretManagerTestCase) {
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.ref.Property = something
		smtc.expectedSecret = ""
		smtc.expectError = errorNoTag
		smtc.apiErr = errors.New(smtc.expectError)
	}

	setSecretWithNoSpecificTag := func(smtc *secretManagerTestCase) {
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString, Tags: tagMap,
		}
		smtc.expectedSecret = jsonTagTestString
	}

	setSecretWithNoTags := func(smtc *secretManagerTestCase) {
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.secretOutput = keyvault.SecretBundle{}
		smtc.expectedSecret = "{}"
	}

	setCertWithTag := func(smtc *secretManagerTestCase) {
		byteArrString := []byte(secretCertificate)
		smtc.secretName = certName
		smtc.certOutput = keyvault.CertificateBundle{
			Cer: &byteArrString, Tags: tagMap,
		}
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.ref.Property = tagname
		smtc.expectedSecret = tagvalue
		smtc.ref.Key = smtc.secretName
	}

	badCertWithTag := func(smtc *secretManagerTestCase) {
		byteArrString := []byte(secretCertificate)
		smtc.secretName = certName
		smtc.ref.Key = smtc.secretName
		smtc.certOutput = keyvault.CertificateBundle{
			Cer: &byteArrString,
		}
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.ref.Property = something
		smtc.expectedSecret = ""
		smtc.expectError = errorNoTag
		smtc.apiErr = errors.New(smtc.expectError)
	}

	setCertWithNoSpecificTag := func(smtc *secretManagerTestCase) {
		byteArrString := []byte(secretCertificate)
		smtc.secretName = certName
		smtc.ref.Key = smtc.secretName
		smtc.certOutput = keyvault.CertificateBundle{
			Cer: &byteArrString, Tags: tagMap,
		}
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.expectedSecret = jsonTagTestString
	}

	setCertWithNoTags := func(smtc *secretManagerTestCase) {
		byteArrString := []byte(secretCertificate)
		smtc.secretName = certName
		smtc.ref.Key = smtc.secretName
		smtc.certOutput = keyvault.CertificateBundle{
			Cer: &byteArrString,
		}
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.expectedSecret = "{}"
	}

	setKeyWithTag := func(smtc *secretManagerTestCase) {
		smtc.secretName = keyName
		smtc.keyOutput = keyvault.KeyBundle{
			Key: newKVJWK([]byte(jwkPubRSA)), Tags: tagMap,
		}
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.ref.Property = tagname
		smtc.expectedSecret = tagvalue
		smtc.ref.Key = smtc.secretName
	}

	badKeyWithTag := func(smtc *secretManagerTestCase) {
		smtc.secretName = keyName
		smtc.ref.Key = smtc.secretName
		smtc.keyOutput = keyvault.KeyBundle{
			Key: newKVJWK([]byte(jwkPubRSA)), Tags: tagMap,
		}
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.ref.Property = something
		smtc.expectedSecret = ""
		smtc.expectError = errorNoTag
		smtc.apiErr = errors.New(smtc.expectError)
	}

	setKeyWithNoSpecificTag := func(smtc *secretManagerTestCase) {
		smtc.secretName = keyName
		smtc.ref.Key = smtc.secretName
		smtc.keyOutput = keyvault.KeyBundle{
			Key: newKVJWK([]byte(jwkPubRSA)), Tags: tagMap,
		}
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.expectedSecret = jsonTagTestString
	}

	setKeyWithNoTags := func(smtc *secretManagerTestCase) {
		smtc.secretName = keyName
		smtc.ref.Key = smtc.secretName
		smtc.keyOutput = keyvault.KeyBundle{
			Key: newKVJWK([]byte(jwkPubRSA)),
		}
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.expectedSecret = "{}"
	}

	badPropertyTag := func(smtc *secretManagerTestCase) {
		smtc.ref.Property = tagname
		smtc.expectedSecret = ""
		smtc.expectError = "property tagname does not exist in key test-secret"
		smtc.apiErr = errors.New(smtc.expectError)
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCase(),
		makeValidSecretManagerTestCaseCustom(setSecretString),
		makeValidSecretManagerTestCaseCustom(setSecretStringWithVersion),
		makeValidSecretManagerTestCaseCustom(setSecretWithProperty),
		makeValidSecretManagerTestCaseCustom(badSecretWithProperty),
		makeValidSecretManagerTestCaseCustom(setPubRSAKey),
		makeValidSecretManagerTestCaseCustom(setPubECKey),
		makeValidSecretManagerTestCaseCustom(setCertificate),
		makeValidSecretManagerTestCaseCustom(badSecretType),
		makeValidSecretManagerTestCaseCustom(setSecretWithTag),
		makeValidSecretManagerTestCaseCustom(badSecretWithTag),
		makeValidSecretManagerTestCaseCustom(setSecretWithNoSpecificTag),
		makeValidSecretManagerTestCaseCustom(setSecretWithNoTags),
		makeValidSecretManagerTestCaseCustom(setCertWithTag),
		makeValidSecretManagerTestCaseCustom(badCertWithTag),
		makeValidSecretManagerTestCaseCustom(setCertWithNoSpecificTag),
		makeValidSecretManagerTestCaseCustom(setCertWithNoTags),
		makeValidSecretManagerTestCaseCustom(setKeyWithTag),
		makeValidSecretManagerTestCaseCustom(badKeyWithTag),
		makeValidSecretManagerTestCaseCustom(setKeyWithNoSpecificTag),
		makeValidSecretManagerTestCaseCustom(setKeyWithNoTags),
		makeValidSecretManagerTestCaseCustom(badPropertyTag),
	}

	sm := Azure{
		provider: &esv1beta1.AzureKVProvider{VaultURL: pointer.StringPtr(fakeURL)},
	}
	for k, v := range successCases {
		sm.baseClient = v.mockClient
		out, err := sm.GetSecret(context.Background(), *v.ref)
		if !utils.ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
	}
}

func TestAzureKeyVaultSecretManagerGetSecretMap(t *testing.T) {
	secretString := "changedvalue"
	secretCertificate := "certificate_value"
	tagMap := getTagMap()

	badSecretString := func(smtc *secretManagerTestCase) {
		smtc.expectedSecret = secretString
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString,
		}
		smtc.expectError = "error unmarshalling json data: invalid character 'c' looking for beginning of value"
	}

	setSecretJSON := func(smtc *secretManagerTestCase) {
		jsonString := jsonSingleTestString
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &jsonString,
		}
		smtc.expectedData["Name"] = []byte("External")
		smtc.expectedData["LastName"] = []byte("Secret")
	}

	setSecretJSONWithProperty := func(smtc *secretManagerTestCase) {
		jsonString := jsonTestString
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &jsonString,
		}
		smtc.ref.Property = "Address"

		smtc.expectedData["Street"] = []byte("Myroad st.")
		smtc.expectedData["CP"] = []byte("J4K4T4")
	}

	badSecretWithProperty := func(smtc *secretManagerTestCase) {
		jsonString := jsonTestString
		smtc.expectedSecret = ""
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &jsonString,
		}
		smtc.ref.Property = "Age"
		smtc.expectError = fmt.Sprintf("property %s does not exist in key %s", smtc.ref.Property, smtc.ref.Key)
		smtc.apiErr = errors.New(smtc.expectError)
	}

	badPubRSAKey := func(smtc *secretManagerTestCase) {
		smtc.secretName = keyName
		smtc.expectedSecret = jwkPubRSA
		smtc.keyOutput = keyvault.KeyBundle{
			Key: newKVJWK([]byte(jwkPubRSA)),
		}
		smtc.ref.Key = smtc.secretName
		smtc.expectError = "cannot get use dataFrom to get key secret"
	}

	badCertificate := func(smtc *secretManagerTestCase) {
		byteArrString := []byte(secretCertificate)
		smtc.secretName = certName
		smtc.expectedSecret = secretCertificate
		smtc.certOutput = keyvault.CertificateBundle{
			Cer: &byteArrString,
		}
		smtc.ref.Key = smtc.secretName
		smtc.expectError = "cannot get use dataFrom to get certificate secret"
	}

	badSecretType := func(smtc *secretManagerTestCase) {
		smtc.secretName = "name"
		smtc.expectedSecret = ""
		smtc.expectError = fmt.Sprintf("unknown Azure Keyvault object Type for %s", smtc.secretName)
		smtc.ref.Key = fmt.Sprintf("dummy/%s", smtc.secretName)
	}

	setSecretTags := func(smtc *secretManagerTestCase) {
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.secretOutput = keyvault.SecretBundle{
			Tags: tagMap,
		}
		smtc.expectedData[testsecret+"_"+tagname] = []byte(tagvalue)
		smtc.expectedData[testsecret+"_"+tagname2] = []byte(tagvalue2)
	}

	setSecretWithJSONTag := func(smtc *secretManagerTestCase) {
		tagJSONMap := make(map[string]*string)
		tagJSONData := `{"keyname":"keyvalue","x":"y"}`
		tagJSONMap["json"] = &tagJSONData
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString, Tags: tagJSONMap,
		}
		smtc.expectedData[testsecret+"_json_keyname"] = []byte("keyvalue")
		smtc.expectedData[testsecret+"_json_x"] = []byte("y")
	}

	setSecretWithNoTags := func(smtc *secretManagerTestCase) {
		smtc.ref.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		tagMapTestEmpty := make(map[string]*string)
		smtc.secretOutput = keyvault.SecretBundle{
			Tags: tagMapTestEmpty,
		}
		smtc.expectedSecret = ""
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(badSecretString),
		makeValidSecretManagerTestCaseCustom(setSecretJSON),
		makeValidSecretManagerTestCaseCustom(setSecretJSONWithProperty),
		makeValidSecretManagerTestCaseCustom(badSecretWithProperty),
		makeValidSecretManagerTestCaseCustom(badPubRSAKey),
		makeValidSecretManagerTestCaseCustom(badCertificate),
		makeValidSecretManagerTestCaseCustom(badSecretType),
		makeValidSecretManagerTestCaseCustom(setSecretTags),
		makeValidSecretManagerTestCaseCustom(setSecretWithJSONTag),
		makeValidSecretManagerTestCaseCustom(setSecretWithNoTags),
	}

	sm := Azure{
		provider: &esv1beta1.AzureKVProvider{VaultURL: pointer.StringPtr(fakeURL)},
	}
	for k, v := range successCases {
		sm.baseClient = v.mockClient
		out, err := sm.GetSecretMap(context.Background(), *v.ref)
		if !utils.ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if err == nil && !reflect.DeepEqual(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedData, out)
		}
	}
}

func TestAzureKeyVaultSecretManagerGetAllSecrets(t *testing.T) {
	secretString := secretString
	secretName := secretName
	wrongName := "not-valid"
	environment := "dev"
	author := "seb"
	enabled := true

	getNextPage := func(ctx context.Context, list keyvault.SecretListResult) (result keyvault.SecretListResult, err error) {
		return keyvault.SecretListResult{
			Value:    nil,
			NextLink: nil,
		}, nil
	}

	setOneSecretByName := func(smtc *secretManagerTestCase) {
		enabledAtt := keyvault.SecretAttributes{
			Enabled: &enabled,
		}
		secretItem := keyvault.SecretItem{
			ID:         &secretName,
			Attributes: &enabledAtt,
		}

		secretList := make([]keyvault.SecretItem, 0)
		secretList = append(secretList, secretItem)

		list := keyvault.SecretListResult{
			Value: &secretList,
		}

		resultPage := keyvault.NewSecretListResultPage(list, getNextPage)
		smtc.listOutput = keyvault.NewSecretListResultIterator(resultPage)

		smtc.expectedSecret = secretString
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString,
		}

		smtc.expectedData[secretName] = []byte(secretString)
	}

	setTwoSecretsByName := func(smtc *secretManagerTestCase) {
		enabledAtt := keyvault.SecretAttributes{
			Enabled: &enabled,
		}
		secretItemOne := keyvault.SecretItem{
			ID:         &secretName,
			Attributes: &enabledAtt,
		}

		secretItemTwo := keyvault.SecretItem{
			ID:         &wrongName,
			Attributes: &enabledAtt,
		}

		secretList := make([]keyvault.SecretItem, 1)
		secretList = append(secretList, secretItemOne, secretItemTwo)

		list := keyvault.SecretListResult{
			Value: &secretList,
		}

		resultPage := keyvault.NewSecretListResultPage(list, getNextPage)
		smtc.listOutput = keyvault.NewSecretListResultIterator(resultPage)

		smtc.expectedSecret = secretString
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString,
		}

		smtc.expectedData[secretName] = []byte(secretString)
	}

	setOneSecretByTag := func(smtc *secretManagerTestCase) {
		enabledAtt := keyvault.SecretAttributes{
			Enabled: &enabled,
		}
		secretItem := keyvault.SecretItem{
			ID:         &secretName,
			Attributes: &enabledAtt,
			Tags:       map[string]*string{"environment": &environment},
		}

		secretList := make([]keyvault.SecretItem, 0)
		secretList = append(secretList, secretItem)

		list := keyvault.SecretListResult{
			Value: &secretList,
		}

		resultPage := keyvault.NewSecretListResultPage(list, getNextPage)
		smtc.listOutput = keyvault.NewSecretListResultIterator(resultPage)

		smtc.expectedSecret = secretString
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString,
		}
		smtc.refFind.Tags = map[string]string{"environment": environment}

		smtc.expectedData[secretName] = []byte(secretString)
	}

	setTwoSecretsByTag := func(smtc *secretManagerTestCase) {
		enabled := true
		enabledAtt := keyvault.SecretAttributes{
			Enabled: &enabled,
		}
		secretItem := keyvault.SecretItem{
			ID:         &secretName,
			Attributes: &enabledAtt,
			Tags:       map[string]*string{"environment": &environment, "author": &author},
		}

		secretList := make([]keyvault.SecretItem, 0)
		secretList = append(secretList, secretItem)

		list := keyvault.SecretListResult{
			Value: &secretList,
		}

		resultPage := keyvault.NewSecretListResultPage(list, getNextPage)
		smtc.listOutput = keyvault.NewSecretListResultIterator(resultPage)

		smtc.expectedSecret = secretString
		smtc.secretOutput = keyvault.SecretBundle{
			Value: &secretString,
		}
		smtc.refFind.Tags = map[string]string{"environment": environment, "author": author}

		smtc.expectedData[secretName] = []byte(secretString)
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setOneSecretByName),
		makeValidSecretManagerTestCaseCustom(setTwoSecretsByName),
		makeValidSecretManagerTestCaseCustom(setOneSecretByTag),
		makeValidSecretManagerTestCaseCustom(setTwoSecretsByTag),
	}

	sm := Azure{
		provider: &esv1beta1.AzureKVProvider{VaultURL: pointer.StringPtr(fakeURL)},
	}
	for k, v := range successCases {
		sm.baseClient = v.mockClient
		out, err := sm.GetAllSecrets(context.Background(), *v.refFind)
		if !utils.ErrorContains(err, v.expectError) {
			t.Errorf(unexpectedError, k, err.Error(), v.expectError)
		}
		if err == nil && !reflect.DeepEqual(out, v.expectedData) {
			t.Errorf(unexpectedSecretData, k, v.expectedData, out)
		}
	}
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:      "test-secret",
		Version:  "default",
		Property: "",
	}
}

func makeValidFind() *esv1beta1.ExternalSecretFind {
	return &esv1beta1.ExternalSecretFind{
		Name: &esv1beta1.FindName{
			RegExp: "^example",
		},
		Tags: map[string]string{},
	}
}

func TestValidateStore(t *testing.T) {
	type args struct {
		store *esv1beta1.SecretStore
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "storeIsNil",
			wantErr: true,
		},
		{
			name:    "specIsNil",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{},
			},
		},
		{
			name:    "providerIsNil",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{},
				},
			},
		},
		{
			name:    "azureKVIsNil",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{},
					},
				},
			},
		},
		{
			name:    "empty auth",
			wantErr: false,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AzureKV: &esv1beta1.AzureKVProvider{},
						},
					},
				},
			},
		},
		{
			name:    "empty client id",
			wantErr: false,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AzureKV: &esv1beta1.AzureKVProvider{
								AuthSecretRef: &esv1beta1.AzureKVAuth{},
							},
						},
					},
				},
			},
		},
		{
			name:    "invalid client id",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AzureKV: &esv1beta1.AzureKVProvider{
								AuthSecretRef: &esv1beta1.AzureKVAuth{
									ClientID: &v1.SecretKeySelector{
										Namespace: pointer.StringPtr("invalid"),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "invalid client secret",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AzureKV: &esv1beta1.AzureKVProvider{
								AuthSecretRef: &esv1beta1.AzureKVAuth{
									ClientSecret: &v1.SecretKeySelector{
										Namespace: pointer.StringPtr("invalid"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Azure{}
			if tt.name == "storeIsNil" {
				if err := a.ValidateStore(nil); (err != nil) != tt.wantErr {
					t.Errorf(errStore, err, tt.wantErr)
				}
			} else if err := a.ValidateStore(tt.args.store); (err != nil) != tt.wantErr {
				t.Errorf(errStore, err, tt.wantErr)
			}
		})
	}
}
