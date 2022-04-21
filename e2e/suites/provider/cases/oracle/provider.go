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
package oracle

import (
	"context"
	"os"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	"github.com/oracle/oci-go-sdk/v56/common"
	vault "github.com/oracle/oci-go-sdk/v56/vault"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

type oracleProvider struct {
	tenancy     string
	user        string
	region      string
	fingerprint string
	privateKey  string
	framework   *framework.Framework
	ctx         context.Context
}

const (
	secretName = "secretName"
)

func newOracleProvider(f *framework.Framework, tenancy, user, region, fingerprint, privateKey string) *oracleProvider {
	prov := &oracleProvider{
		tenancy:     tenancy,
		user:        user,
		region:      region,
		fingerprint: fingerprint,
		privateKey:  privateKey,
		framework:   f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func newFromEnv(f *framework.Framework) *oracleProvider {
	tenancy := os.Getenv("OCI_TENANCY_OCID")
	user := os.Getenv("OCI_USER_OCID")
	region := os.Getenv("OCI_REGION")
	fingerprint := os.Getenv("OCI_FINGERPRINT")
	privateKey := os.Getenv("OCI_PRIVATE_KEY")
	return newOracleProvider(f, tenancy, user, region, fingerprint, privateKey)
}

func (p *oracleProvider) CreateSecret(key string, val framework.SecretEntry) {
	configurationProvider := common.NewRawConfigurationProvider(p.tenancy, p.user, p.region, p.fingerprint, p.privateKey, nil)
	client, err := vault.NewVaultsClientWithConfigurationProvider(configurationProvider)
	Expect(err).ToNot(HaveOccurred())
	vmssecretrequest := vault.CreateSecretRequest{}
	vmssecretrequest.SecretName = utilpointer.StringPtr(secretName)
	vmssecretrequest.SecretContent = vault.Base64SecretContentDetails{
		Name:    utilpointer.StringPtr(key),
		Content: utilpointer.StringPtr(val.Value),
	}
	_, err = client.CreateSecret(p.ctx, vmssecretrequest)
	Expect(err).ToNot(HaveOccurred())
}

func (p *oracleProvider) DeleteSecret(key string) {
	configurationProvider := common.NewRawConfigurationProvider(p.tenancy, p.user, p.region, p.fingerprint, p.privateKey, nil)
	client, err := vault.NewVaultsClientWithConfigurationProvider(configurationProvider)
	Expect(err).ToNot(HaveOccurred())
	vmssecretrequest := vault.ScheduleSecretDeletionRequest{}
	vmssecretrequest.SecretId = utilpointer.StringPtr(key)
	_, err = client.ScheduleSecretDeletion(p.ctx, vmssecretrequest)
	Expect(err).ToNot(HaveOccurred())
}

func (p *oracleProvider) BeforeEach() {
	OracleCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: p.framework.Namespace.Name,
		},
		StringData: map[string]string{
			secretName: "value",
		},
	}
	err := p.framework.CRClient.Create(context.Background(), OracleCreds)
	Expect(err).ToNot(HaveOccurred())

	secretStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.framework.Namespace.Name,
			Namespace: p.framework.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Oracle: &esv1beta1.OracleProvider{
					Region: p.region,
					Vault:  "vaultOCID",
					Auth: &esv1beta1.OracleAuth{
						Tenancy: p.tenancy,
						User:    p.user,
						SecretRef: esv1beta1.OracleSecretRef{
							Fingerprint: esmeta.SecretKeySelector{
								Name: "vms-secret",
								Key:  "keyid",
							},
							PrivateKey: esmeta.SecretKeySelector{
								Name: "vms-secret",
								Key:  "accesskey",
							},
						},
					},
				},
			},
		},
	}
	err = p.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
