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

package common

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type NamespacedProviderSyncConfig struct {
	Description        string
	ExternalSecretName string
	TargetSecretName   string
	RemoteKey          string
	RemoteSecretValue  string
	RemoteProperty     string
	SecretKey          string
	ExpectedValue      string
}

func NamespacedProviderSync(_ *framework.Framework, cfg NamespacedProviderSyncConfig) (string, func(*framework.TestCase)) {
	return cfg.Description, func(tc *framework.TestCase) {
		tc.ExternalSecret.ObjectMeta.Name = cfg.ExternalSecretName
		tc.ExternalSecret.Spec.Target.Name = cfg.TargetSecretName
		tc.Secrets = map[string]framework.SecretEntry{
			cfg.RemoteKey: {Value: cfg.RemoteSecretValue},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				cfg.SecretKey: []byte(cfg.ExpectedValue),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: cfg.SecretKey,
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      cfg.RemoteKey,
				Property: cfg.RemoteProperty,
			},
		}}
	}
}

type NamespacedProviderRefreshConfig struct {
	Description         string
	ExternalSecretName  string
	TargetSecretName    string
	RemoteKey           string
	InitialSecretValue  string
	UpdatedSecretValue  string
	RemoteProperty      string
	SecretKey           string
	InitialExpectedData string
	UpdatedExpectedData string
	RefreshInterval     time.Duration
	WaitTimeout         time.Duration
	UpdateRemoteSecret  func(tc *framework.TestCase, prov framework.SecretStoreProvider)
}

func NamespacedProviderRefresh(_ *framework.Framework, cfg NamespacedProviderRefreshConfig) (string, func(*framework.TestCase)) {
	return cfg.Description, func(tc *framework.TestCase) {
		waitTimeout := cfg.WaitTimeout
		if waitTimeout == 0 {
			waitTimeout = 30 * time.Second
		}

		tc.ExternalSecret.ObjectMeta.Name = cfg.ExternalSecretName
		tc.ExternalSecret.Spec.Target.Name = cfg.TargetSecretName
		tc.ExternalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: cfg.RefreshInterval}
		tc.Secrets = map[string]framework.SecretEntry{
			cfg.RemoteKey: {Value: cfg.InitialSecretValue},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				cfg.SecretKey: []byte(cfg.InitialExpectedData),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: cfg.SecretKey,
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      cfg.RemoteKey,
				Property: cfg.RemoteProperty,
			},
		}}
		tc.AfterSync = func(prov framework.SecretStoreProvider, _ *corev1.Secret) {
			if cfg.UpdateRemoteSecret != nil {
				cfg.UpdateRemoteSecret(tc, prov)
			} else {
				prov.DeleteSecret(cfg.RemoteKey)
				prov.CreateSecret(cfg.RemoteKey, framework.SecretEntry{
					Value: cfg.UpdatedSecretValue,
				})
			}
			waitForSecretData(tc.Framework, tc.ExternalSecret.Namespace, tc.ExternalSecret.Spec.Target.Name, map[string][]byte{
				cfg.SecretKey: []byte(cfg.UpdatedExpectedData),
			}, waitTimeout)
		}
	}
}

type NamespacedProviderFindConfig struct {
	Description        string
	ExternalSecretName string
	TargetSecretName   string
	MatchRegExp        string
	MatchingSecrets    map[string]string
	IgnoredSecrets     map[string]string
}

func NamespacedProviderFind(_ *framework.Framework, cfg NamespacedProviderFindConfig) (string, func(*framework.TestCase)) {
	return cfg.Description, func(tc *framework.TestCase) {
		secrets := make(map[string]framework.SecretEntry, len(cfg.MatchingSecrets)+len(cfg.IgnoredSecrets))
		for key, value := range cfg.MatchingSecrets {
			secrets[key] = framework.SecretEntry{Value: value}
		}
		for key, value := range cfg.IgnoredSecrets {
			secrets[key] = framework.SecretEntry{Value: value}
		}

		expectedData := make(map[string][]byte, len(cfg.MatchingSecrets))
		for key, value := range cfg.MatchingSecrets {
			expectedData[key] = []byte(value)
		}

		tc.ExternalSecret.ObjectMeta.Name = cfg.ExternalSecretName
		tc.ExternalSecret.Spec.Target.Name = cfg.TargetSecretName
		tc.Secrets = secrets
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: expectedData,
		}
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{{
			Find: &esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: cfg.MatchRegExp,
				},
			},
		}}
	}
}
