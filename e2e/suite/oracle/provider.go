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

	// nolint
	. "github.com/onsi/ginkgo"

	// nolint
	. "github.com/onsi/gomega"
	"github.com/oracle/oci-go-sdk/v45/common"
	vault "github.com/oracle/oci-go-sdk/v45/vault"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
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

func (p *oracleProvider) CreateSecret(key, val string) {
	configurationProvider := common.NewRawConfigurationProvider(p.tenancy, p.user, p.region, p.fingerprint, p.privateKey, nil)
	client, err := vault.NewVaultsClientWithConfigurationProvider(configurationProvider)
	Expect(err).ToNot(HaveOccurred())
	vmssecretrequest := vault.CreateSecretRequest{}
	vmssecretrequest.SecretName = utilpointer.StringPtr(secretName)
	vmssecretrequest.SecretContent = vault.Base64SecretContentDetails{
		Name:    utilpointer.StringPtr("secretName"),
		Content: utilpointer.StringPtr("secretContent"),
	}
	_, err = client.CreateSecret(p.ctx, vmssecretrequest)
	Expect(err).ToNot(HaveOccurred())
}

func (p *oracleProvider) DeleteSecret(key string) {
	configurationProvider := common.NewRawConfigurationProvider(p.tenancy, p.user, p.region, p.fingerprint, p.privateKey, nil)
	client, err := vault.NewVaultsClientWithConfigurationProvider(configurationProvider)
	Expect(err).ToNot(HaveOccurred())
	vmssecretrequest := vault.ScheduleSecretDeletionRequest{}
	vmssecretrequest.SecretId = utilpointer.StringPtr(secretName)
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

	secretStore := &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.framework.Namespace.Name,
			Namespace: p.framework.Namespace.Name,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				Oracle: &esv1alpha1.OracleProvider{
					Auth: esv1alpha1.OracleAuth{
						SecretRef: esv1alpha1.OracleSecretRef{
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
