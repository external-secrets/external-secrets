package keyvault

import (
	context "context"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"gotest.tools/assert"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

type azureMock struct {
	mock.Mock
}

func (_m *azureMock) getKeyVaultSecrets(ctx context.Context, vaultName, version, secretName string, withTags bool) (map[string][]byte, error) {
	ret := _m.Called(ctx, vaultName, version, secretName, withTags)

	var r0 map[string][]byte
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, bool) map[string][]byte); ok {
		r0 = rf(ctx, vaultName, version, secretName, withTags)
	} else if ret.Get(0) != nil {
		r0 = ret.Get(0).(map[string][]byte)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, bool) error); ok {
		r1 = rf(ctx, vaultName, version, secretName, withTags)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func TestGetSecret(t *testing.T) {
	testAzure := new(Azure)
	anAzureMock := new(azureMock)
	ctx := context.Background()
	testAzure.iAzure = anAzureMock
	property := "testProperty"
	version := "v1"

	rf := esv1alpha1.ExternalSecretDataRemoteRef{
		Key:      "testName",
		Property: property,
		Version:  version,
	}
	returnValue := make(map[string][]byte)
	returnValue["key"] = []byte{'A'}
	anAzureMock.On("getKeyVaultSecrets", ctx, "testName", "v1", "testProperty", false).Return(returnValue, nil)
	_, err := testAzure.GetSecret(ctx, rf)
	assert.NilError(t, err, "the return err should be nil")
	anAzureMock.AssertExpectations(t)
}

func TestGetSecretMap(t *testing.T) {
	testAzure := new(Azure)
	anAzureMock := new(azureMock)
	ctx := context.Background()
	testAzure.iAzure = anAzureMock
	property := "testProperty"
	version := "v1"
	rf := esv1alpha1.ExternalSecretDataRemoteRef{
		Key:      "testName",
		Property: property,
		Version:  version,
	}
	returnValue := make(map[string][]byte)
	returnValue["key"] = []byte{'a'}
	anAzureMock.On("getKeyVaultSecrets", ctx, "testName", "v1", "", true).Return(returnValue, nil)
	_, err := testAzure.GetSecretMap(ctx, rf)
	assert.NilError(t, err, "the return err should be nil")
	anAzureMock.AssertExpectations(t)
}

func TestGetCertBundleForPKCS(t *testing.T) {
	rawCertExample := "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURC" +
		"VENDQWUyZ0F3SUJBZ0lFUnIxWTdEQU5CZ2txaGtpRzl3MEJBUVVGQURBeU1Rc3d" +
		"DUVlEVlFRR0V3SkUKUlRFUU1BNEdBMVVFQ2hNSFFXMWhaR1YxY3pFUk1BOEdBMV" +
		"VFQXhNSVUwRlFJRkp2YjNRd0hoY05NVE13TWpFMApNVE15TmpRNVdoY05NelV4T" +
		"WpNeE1UTXlOalE1V2pBeU1Rc3dDUVlEVlFRR0V3SkVSVEVRTUE0R0ExVUVDaE1I" +
		"CnFWUlE3NjNGODFwWnorNXgyejJ6NmZyd0JHNUF3YUZKL1RmTE9HQzZQWnl5bW1" +
		"pSlllL2tjUDdVeUhMQnBUUVkKLzloNTF5dDB5NlRBS1JmRk1wMlhuVUZBaWdyL0" +
		"0xYVc1NjdORStQYzN5S0RWWlVHdU82UXZ0cExCZkpPS3pZSAowc3F3OElmYjRlN" +
		"0R6TkJuTmRoVDhzbGdUYkh5K3RzZUtPb0xHNi9rUktmRmRvSmRoeHAzeGNnbm56" +
		"ZkY0anUvCi9UZTRYaWsxNC9FMAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t"
	c, ok := getCertBundleForPKCS(rawCertExample, true, true)
	bundle := ""
	tassert.Nil(t, ok)
	tassert.Equal(t, c, bundle)
}

func TestAppendTagsToSecretMap(t *testing.T) {
	var secretsMap map[string][]byte
	var secretsMapOrigin map[string][]byte
	secret := "testsecret"
	var tags map[string]*string
	appendTagsToSecretMap(secret, secretsMap, tags)
	tassert.Equal(t, secretsMap, secretsMapOrigin)
}
