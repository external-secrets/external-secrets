/*
Copyright 2020 The cert-manager Authors.
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

package generator

import (
	"context"
	"os"
	"strings"
	"time"

	//nolint
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grafanaclient "github.com/grafana/grafana-openapi-client-go/client"
	grafanasearch "github.com/grafana/grafana-openapi-client-go/client/search"
	grafanasa "github.com/grafana/grafana-openapi-client-go/client/service_accounts"
)

var _ = Describe("grafana generator", Label("grafana"), func() {
	f := framework.New("grafana")
	const grafanaCredsSecretName = "grafana-creds"

	grafanaClient := newGrafanaClient()

	BeforeEach(func() {
		// grafana instance may need to load for a bit
		// we'll wake it up here and wait for it to be ready
		Eventually(func() error {
			_, err := grafanaClient.Search.Search(&grafanasearch.SearchParams{})
			return err
		}).WithPolling(time.Second * 15).WithTimeout(time.Minute * 5).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		// ESO does clean up tokens, but not the service accounts.
		accounts, err := grafanaClient.ServiceAccounts.SearchOrgServiceAccountsWithPaging(&grafanasa.SearchOrgServiceAccountsWithPagingParams{
			Perpage: ptr.To(int64(100)),
			Page:    ptr.To(int64(1)),
			Query:   ptr.To(f.Namespace.Name),
		})
		Expect(err).ToNot(HaveOccurred())
		if accounts.GetPayload().ServiceAccounts != nil && len(accounts.GetPayload().ServiceAccounts) > 0 {
			for _, sa := range accounts.GetPayload().ServiceAccounts {
				_, err := grafanaClient.ServiceAccounts.DeleteServiceAccount(sa.ID)
				Expect(err).ToNot(HaveOccurred())
			}
		}
	})

	setupGenerator := func(tc *testCase) {
		err := f.CRClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      grafanaCredsSecretName,
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"grafana-token": []byte(os.Getenv("GRAFANA_TOKEN")),
			},
		})
		Expect(err).ToNot(HaveOccurred())
		tc.Generator = &genv1alpha1.Grafana{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
				Kind:       genv1alpha1.GrafanaKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatorName,
				Namespace: f.Namespace.Name,
			},
			Spec: genv1alpha1.GrafanaSpec{
				URL: os.Getenv("GRAFANA_URL"),
				ServiceAccount: genv1alpha1.GrafanaServiceAccount{
					Name: f.Namespace.Name,
					Role: "Viewer",
				},
				Auth: genv1alpha1.GrafanaAuth{
					Token: &genv1alpha1.SecretKeySelector{
						Name: grafanaCredsSecretName,
						Key:  "grafana-token",
					},
				},
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				SourceRef: &esv1beta1.StoreGeneratorSourceRef{
					GeneratorRef: &esv1beta1.GeneratorRef{
						Kind: "Grafana",
						Name: generatorName,
					},
				},
			},
		}
	}

	ensureExternalSecretPurgesGeneratorState := func(tc *testCase) {
		// delete ES to trigger cleanup of generator state
		err := f.CRClient.Delete(context.Background(), tc.ExternalSecret)
		Expect(err).ToNot(HaveOccurred())

		By("waiting for generator state to be cleaned up")
		// wait for generator state to be cleaned up
		Eventually(func() int {
			generatorStates := &genv1alpha1.GeneratorStateList{}
			err := f.CRClient.List(context.Background(), generatorStates, client.InNamespace(f.Namespace.Name))
			if err != nil {
				return -1
			}
			GinkgoLogr.Info("found generator states", "states", generatorStates.Items)
			return len(generatorStates.Items)
		}).WithPolling(time.Second * 1).WithTimeout(time.Minute * 2).Should(BeZero())
	}

	tokenIsUsable := func(tc *testCase) {
		tc.AfterSync = func(secret *v1.Secret) {
			// ensure token exists and is usable
			Expect(string(secret.Data["token"])).ToNot(BeEmpty())

			_, err := grafanaClient.Search.Search(&grafanasearch.SearchParams{
				Query: ptr.To(""),
			})
			Expect(err).ToNot(HaveOccurred())
			ensureExternalSecretPurgesGeneratorState(tc)
		}
	}

	cleanupServiceAccountsAfterDeletion := func(tc *testCase) {
		tc.ExternalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second * 10}
		tc.AfterSync = func(secret *v1.Secret) {
			// Wait for ES to be rotated a couple of times,
			// this should create a couple of service accounts.
			// This allows us to verify that the service accounts are cleaned up
			// after the generator is deleted.
			Eventually(func() bool {
				generatorStates := &genv1alpha1.GeneratorStateList{}
				err := f.CRClient.List(context.Background(), generatorStates, client.InNamespace(f.Namespace.Name))
				Expect(err).ToNot(HaveOccurred())
				GinkgoLogr.Info("generator states", "states", generatorStates.Items)
				return len(generatorStates.Items) > 2
			}).WithPolling(time.Second * 10).WithTimeout(time.Minute * 5).Should(BeTrue())

			ensureExternalSecretPurgesGeneratorState(tc)

			// ensure service accounts are cleaned up
			saList, err := grafanaClient.ServiceAccounts.SearchOrgServiceAccountsWithPaging(&grafanasa.SearchOrgServiceAccountsWithPagingParams{
				Perpage: ptr.To(int64(100)),
				Page:    ptr.To(int64(1)),
				Query:   ptr.To(f.Namespace.Name),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(saList.GetPayload().ServiceAccounts).To(HaveLen(1))
			tokens, err := grafanaClient.ServiceAccounts.ListTokensWithParams(&grafanasa.ListTokensParams{
				ServiceAccountID: saList.GetPayload().ServiceAccounts[0].ID,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(tokens.GetPayload()).To(BeEmpty())
		}
	}

	DescribeTable("generate secrets with grafana generator", generatorTableFunc,
		Entry("should generate a token that can be used to access the API", f, setupGenerator, tokenIsUsable),
		Entry("deleting a generator should cleanup the generated service accounts", f, setupGenerator, cleanupServiceAccountsAfterDeletion),
	)
})

func newGrafanaClient() *grafanaclient.GrafanaHTTPAPI {
	url := strings.TrimPrefix(os.Getenv("GRAFANA_URL"), "https://")
	return grafanaclient.NewHTTPClientWithConfig(nil, &grafanaclient.TransportConfig{
		Host:         url,
		BasePath:     "/api",
		Schemes:      []string{"https"},
		APIKey:       os.Getenv("GRAFANA_TOKEN"),
		NumRetries:   15,
		RetryTimeout: time.Second * 6,
	})
}
