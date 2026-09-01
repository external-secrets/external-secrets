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
package oracle

import (
	"encoding/base64"
	"os"
	"sync"
	"time"

	//nolint
	. "github.com/onsi/ginkgo/v2"

	//nolint
	. "github.com/onsi/gomega"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/vault"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	// credentialsSecret is the Kubernetes Secret the SecretStore reads the API
	// signing credentials from. Created in BeforeEach.
	credentialsSecret = "provider-secret"

	// retention is how far out a deleted secret is scheduled for removal. The
	// Vault service accepts 1 to 30 days and enforces the floor, so a
	// throwaway secret occupies a quota slot for a day whatever we do here.
	// The 30 day default would hold it for a month, which an always-free
	// tenancy (150 secrets) cannot absorb: one run of this suite creates 15.
	retention = 25 * time.Hour
)

type oracleProvider struct {
	tenancy       string
	user          string
	region        string
	fingerprint   string
	privateKey    string
	vaultID       string
	compartment   string
	encryptionKey string
	framework     *framework.Framework

	client vault.VaultsClient

	// secretIDs maps the name a secret was created under to the OCID the
	// service assigned it. Deletion addresses a secret by OCID, while the
	// framework only ever knows the name it asked for.
	mu        sync.Mutex
	secretIDs map[string]string
}

func newFromEnv(f *framework.Framework) *oracleProvider {
	prov := &oracleProvider{
		tenancy:       os.Getenv("ORACLE_TENANCY_OCID"),
		user:          os.Getenv("ORACLE_USER_OCID"),
		region:        os.Getenv("ORACLE_REGION"),
		fingerprint:   os.Getenv("ORACLE_FINGERPRINT"),
		privateKey:    os.Getenv("ORACLE_KEY"),
		vaultID:       os.Getenv("ORACLE_VAULT_OCID"),
		compartment:   os.Getenv("ORACLE_COMPARTMENT_OCID"),
		encryptionKey: os.Getenv("ORACLE_ENCRYPTION_KEY_OCID"),
		framework:     f,
		secretIDs:     map[string]string{},
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (p *oracleProvider) BeforeEach() {
	// The OCI client is built here rather than in the constructor. Ginkgo
	// builds the whole spec tree before applying a label filter, so the
	// constructor runs even on a leg testing some other provider, where these
	// variables are deliberately empty. Failing there would break every other
	// provider's run.
	client, err := vault.NewVaultsClientWithConfigurationProvider(
		// A nil passphrase means the API signing key must be unencrypted.
		common.NewRawConfigurationProvider(
			p.tenancy, p.user, p.region, p.fingerprint, p.privateKey, nil),
	)
	Expect(err).ToNot(HaveOccurred())

	// The SDK does not retry by default. Specs run in parallel and each one
	// creates and deletes, which is enough concurrent traffic for the Vaults
	// service to answer 429 TooManyRequests.
	policy := common.DefaultRetryPolicy()
	client.SetCustomClientConfiguration(common.CustomClientConfiguration{
		RetryPolicy: &policy,
	})
	p.client = client

	oracleCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credentialsSecret,
			Namespace: p.framework.Namespace.Name,
		},
		StringData: map[string]string{
			"keyid":     p.fingerprint,
			"accesskey": p.privateKey,
		},
	}
	err = p.framework.CRClient.Create(GinkgoT().Context(), oracleCreds)
	Expect(err).ToNot(HaveOccurred())

	secretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.framework.Namespace.Name,
			Namespace: p.framework.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Oracle: &esv1.OracleProvider{
					Region: p.region,
					Vault:  p.vaultID,
					Auth: &esv1.OracleAuth{
						Tenancy: p.tenancy,
						User:    p.user,
						SecretRef: esv1.OracleSecretRef{
							Fingerprint: esmeta.SecretKeySelector{
								Name: credentialsSecret,
								Key:  "keyid",
							},
							PrivateKey: esmeta.SecretKeySelector{
								Name: credentialsSecret,
								Key:  "accesskey",
							},
						},
					},
				},
			},
		},
	}
	err = p.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (p *oracleProvider) CreateSecret(key string, val framework.SecretEntry) {
	// The secret's name is what the provider resolves remoteRef.key against,
	// via GetSecretBundleByName, so it has to be the key the framework asked
	// for. SecretContent.Name is only a label on the version.
	content := base64.StdEncoding.EncodeToString([]byte(val.Value))
	resp, err := p.client.CreateSecret(GinkgoT().Context(), vault.CreateSecretRequest{
		CreateSecretDetails: vault.CreateSecretDetails{
			CompartmentId: &p.compartment,
			VaultId:       &p.vaultID,
			KeyId:         &p.encryptionKey,
			SecretName:    &key,
			SecretContent: vault.Base64SecretContentDetails{
				Name: &key,
				// Documented as base64; the raw value would be stored
				// mangled and fail to decode on read.
				Content: &content,
			},
		},
	})
	Expect(err).ToNot(HaveOccurred())

	p.mu.Lock()
	defer p.mu.Unlock()
	p.secretIDs[key] = *resp.Id
}

func (p *oracleProvider) DeleteSecret(key string) {
	p.mu.Lock()
	id, ok := p.secretIDs[key]
	delete(p.secretIDs, key)
	p.mu.Unlock()
	if !ok {
		// CreateSecret never recorded an OCID for this key, so it failed and
		// has already reported why. Staying quiet here keeps that first
		// failure as the one the report shows.
		return
	}

	// Vault state transitions are asynchronous. A secret is CREATING for a
	// few seconds after the create call returns, and scheduling deletion in
	// that window fails with 409 IncorrectState.
	p.waitActive(id)

	when := common.SDKTime{Time: time.Now().Add(retention)}
	_, err := p.client.ScheduleSecretDeletion(GinkgoT().Context(), vault.ScheduleSecretDeletionRequest{
		SecretId: &id,
		ScheduleSecretDeletionDetails: vault.ScheduleSecretDeletionDetails{
			TimeOfDeletion: &when,
		},
	})
	Expect(err).ToNot(HaveOccurred())
}

func (p *oracleProvider) waitActive(id string) {
	Eventually(func() (vault.SecretLifecycleStateEnum, error) {
		resp, err := p.client.GetSecret(GinkgoT().Context(), vault.GetSecretRequest{
			SecretId: &id,
		})
		if err != nil {
			return "", err
		}
		return resp.LifecycleState, nil
	}, time.Minute, 2*time.Second).Should(Equal(vault.SecretLifecycleStateActive))
}
