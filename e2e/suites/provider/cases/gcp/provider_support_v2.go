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

package gcp

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	gcpsmv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/gcp/v2alpha1"
)

const (
	defaultV2WaitTimeout  = 60 * time.Second
	defaultV2PollInterval = 2 * time.Second
	withWorkloadIdentity  = "with workload identity"
)

type ProviderV2 struct {
	access    gcpAccessConfig
	backend   *GcpProvider
	framework *framework.Framework
}

type v2ClusterProviderScenario struct {
	AuthScope            esv1.AuthenticationScope
	ConfigName           string
	ConfigNamespace      string
	NamePrefix           string
	ProviderNamespace    string
	ProviderRefNamespace string
	WorkloadNamespace    string
}

func NewProviderV2(f *framework.Framework) *ProviderV2 {
	access := newGCPAccessConfigFromEnv()
	backend := &GcpProvider{
		ServiceAccountName:      access.ServiceAccountName,
		ServiceAccountNamespace: "default",
		framework:               f,
		credentials:             access.Credentials,
		projectID:               access.ProjectID,
		clusterLocation:         access.ClusterLocation,
		clusterName:             access.ClusterName,
		access:                  access,
	}
	prov := &ProviderV2{
		access:    access,
		backend:   backend,
		framework: f,
	}

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			return
		}
		skipIfGCPStaticEnvMissing(access)
	})

	return prov
}

func (p *ProviderV2) CreateSecret(key string, val framework.SecretEntry) {
	p.backend.CreateSecret(key, val)
}

func (p *ProviderV2) UpdateSecret(key string, val framework.SecretEntry) {
	p.backend.UpdateSecret(key, val)
}

func (p *ProviderV2) DeleteSecret(key string) {
	p.backend.DeleteSecret(key)
}

func useV2StaticAuth(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProviderWithStaticAuthAtAddress(
			providerConfigNamespaceForStaticAuth(prov),
			frameworkv2.ProviderAddress("gcp"),
		)
	}
}

func useV2WorkloadIdentity(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProviderWithWorkloadIdentityAtAddress(
			providerConfigNamespaceForWorkloadIdentity(prov),
			frameworkv2.ProviderAddress("gcp"),
		)
	}
}

func useV2MountedPodIdentity(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProviderWithWorkloadIdentityAtAddress(
			providerConfigNamespaceForWorkloadIdentity(prov),
			frameworkv2.ProviderAddressInNamespace("gcp", prov.backend.ServiceAccountNamespace),
		)
	}
}

func useV2ReferencedServiceAccount(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareReferencedServiceAccountProvider(
			frameworkv2.ProviderAddressInNamespace("gcp", prov.backend.ServiceAccountNamespace),
		)
	}
}

func providerConfigNamespaceForStaticAuth(prov *ProviderV2) string {
	return prov.framework.Namespace.Name
}

func providerConfigNamespaceForWorkloadIdentity(prov *ProviderV2) string {
	return prov.backend.ServiceAccountNamespace
}

func (p *ProviderV2) prepareNamespacedProviderWithStaticAuthAtAddress(configNamespace, address string) func(*framework.TestCase, framework.SecretStoreProvider) {
	return func(_ *framework.TestCase, _ framework.SecretStoreProvider) {
		createSecretManagerV2StaticConfig(p.framework, configNamespace, p.framework.Namespace.Name, p.access)
		frameworkv2.CreateProviderConnection(
			p.framework,
			p.framework.Namespace.Name,
			p.framework.Namespace.Name,
			address,
			gcpsmv2alpha1.GroupVersion.String(),
			gcpsmv2alpha1.SecretManagerKind,
			p.framework.Namespace.Name,
			configNamespace,
		)
		frameworkv2.WaitForProviderConnectionReady(p.framework, p.framework.Namespace.Name, p.framework.Namespace.Name, defaultV2WaitTimeout)
	}
}

func (p *ProviderV2) prepareNamespacedProviderWithWorkloadIdentityAtAddress(configNamespace, address string) func(*framework.TestCase, framework.SecretStoreProvider) {
	return func(_ *framework.TestCase, _ framework.SecretStoreProvider) {
		skipIfGCPManagedEnvMissing(p.access)

		createSecretManagerV2WorkloadIdentityConfig(
			p.framework,
			configNamespace,
			p.framework.Namespace.Name,
			p.access,
			configNamespace,
		)
		frameworkv2.CreateProviderConnection(
			p.framework,
			p.framework.Namespace.Name,
			p.framework.Namespace.Name,
			address,
			gcpsmv2alpha1.GroupVersion.String(),
			gcpsmv2alpha1.SecretManagerKind,
			p.framework.Namespace.Name,
			configNamespace,
		)
		frameworkv2.WaitForProviderConnectionReady(p.framework, p.framework.Namespace.Name, p.framework.Namespace.Name, defaultV2WaitTimeout)
	}
}

func (p *ProviderV2) prepareReferencedServiceAccountProvider(address string) func(*framework.TestCase, framework.SecretStoreProvider) {
	return func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
		skipIfGCPManagedEnvMissing(p.access)

		configNamespace := providerConfigNamespaceForWorkloadIdentity(p)
		configName := p.framework.Namespace.Name
		clusterProviderName := referencedServiceAccountClusterProviderName(p.framework.Namespace.Name)

		createSecretManagerV2WorkloadIdentityConfig(
			p.framework,
			configNamespace,
			configName,
			p.access,
			configNamespace,
		)
		frameworkv2.CreateClusterProviderConnection(
			p.framework,
			clusterProviderName,
			address,
			gcpsmv2alpha1.GroupVersion.String(),
			gcpsmv2alpha1.SecretManagerKind,
			configName,
			configNamespace,
			esv1.AuthenticationScopeManifestNamespace,
			nil,
		)
		frameworkv2.WaitForClusterProviderReady(p.framework, clusterProviderName, defaultV2WaitTimeout)
		configureV2ReferencedServiceAccountStoreRef(tc, clusterProviderName)
	}
}

func referencedServiceAccountClusterProviderName(namespace string) string {
	return namespace + "-referenced-service-account"
}

func configureV2ReferencedServiceAccountStoreRef(tc *framework.TestCase, clusterProviderName string) {
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterProviderKindStr
	tc.ExternalSecret.Spec.SecretStoreRef.Name = clusterProviderName
}

func newSecretManagerV2StaticConfig(namespace, name string, access gcpAccessConfig) *gcpsmv2alpha1.SecretManager {
	return &gcpsmv2alpha1.SecretManager{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gcpsmv2alpha1.GroupVersion.String(),
			Kind:       gcpsmv2alpha1.SecretManagerKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gcpsmv2alpha1.SecretManagerSpec{
			ProjectID: access.ProjectID,
			Auth: esv1.GCPSMAuth{
				SecretRef: &esv1.GCPSMAuthSecretRef{
					SecretAccessKey: esmeta.SecretKeySelector{
						Name: staticCredentialsSecretName,
						Key:  serviceAccountKey,
					},
				},
			},
		},
	}
}

func createSecretManagerV2StaticConfig(f *framework.Framework, namespace, name string, access gcpAccessConfig) *gcpsmv2alpha1.SecretManager {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      staticCredentialsSecretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			serviceAccountKey: access.Credentials,
		},
	}
	Expect(f.CreateObjectWithRetry(secret)).To(Succeed())

	cfg := newSecretManagerV2StaticConfig(namespace, name, access)
	Expect(f.CreateObjectWithRetry(cfg)).To(Succeed())
	return cfg
}

func newSecretManagerV2WorkloadIdentityConfig(namespace, name string, access gcpAccessConfig, serviceAccountNamespace string) *gcpsmv2alpha1.SecretManager {
	return &gcpsmv2alpha1.SecretManager{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gcpsmv2alpha1.GroupVersion.String(),
			Kind:       gcpsmv2alpha1.SecretManagerKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gcpsmv2alpha1.SecretManagerSpec{
			ProjectID: access.ProjectID,
			Auth: esv1.GCPSMAuth{
				WorkloadIdentity: &esv1.GCPWorkloadIdentity{
					ClusterLocation: access.ClusterLocation,
					ClusterName:     access.ClusterName,
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name:      access.ServiceAccountName,
						Namespace: utilpointer.String(serviceAccountNamespace),
					},
				},
			},
		},
	}
}

func createSecretManagerV2WorkloadIdentityConfig(f *framework.Framework, namespace, name string, access gcpAccessConfig, serviceAccountNamespace string) *gcpsmv2alpha1.SecretManager {
	cfg := newSecretManagerV2WorkloadIdentityConfig(namespace, name, access, serviceAccountNamespace)
	Expect(f.CreateObjectWithRetry(cfg)).To(Succeed())
	return cfg
}

func providerConfigNamespace(authScope esv1.AuthenticationScope, providerNamespace, workloadNamespace string) string {
	if authScope == esv1.AuthenticationScopeProviderNamespace {
		return providerNamespace
	}
	return workloadNamespace
}

func providerReferenceNamespace(authScope esv1.AuthenticationScope, providerNamespace string) string {
	if authScope == esv1.AuthenticationScopeProviderNamespace {
		return providerNamespace
	}
	return ""
}

func newV2ClusterProviderScenario(workloadNamespace, prefix string, authScope esv1.AuthenticationScope, createProviderNamespace func(prefix string) string) v2ClusterProviderScenario {
	providerNamespace := workloadNamespace
	if authScope == esv1.AuthenticationScopeProviderNamespace && createProviderNamespace != nil {
		providerNamespace = createProviderNamespace(prefix + "-provider")
	}

	return v2ClusterProviderScenario{
		AuthScope:            authScope,
		ConfigName:           fmt.Sprintf("%s-config", prefix),
		ConfigNamespace:      providerConfigNamespace(authScope, providerNamespace, workloadNamespace),
		NamePrefix:           fmt.Sprintf("%s-%s", workloadNamespace, prefix),
		ProviderNamespace:    providerNamespace,
		ProviderRefNamespace: providerReferenceNamespace(authScope, providerNamespace),
		WorkloadNamespace:    workloadNamespace,
	}
}

func (s v2ClusterProviderScenario) ClusterProviderName() string {
	return fmt.Sprintf("%s-cluster-provider", s.NamePrefix)
}
