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

package secretsink

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	v1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretsink/internal/fakes"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
)

var fakeProvider *fake.Client

var _ = Describe("secretsink", func() {
	var (
		reconciler *Reconciler
		client     *fakes.Client
	)
	BeforeEach(func() {
		client = new(fakes.Client)
		reconciler = &Reconciler{client, logr.Discard(), nil, nil, 0, ""}
	})
	Describe("#Reconcile", func() {
		var (
			statusWriter *fakes.StatusWriter
		)

		BeforeEach(func() {
			statusWriter = new(fakes.StatusWriter)
			client.StatusReturns(statusWriter)
		})

		It("succeeds", func() {
			namspacedName := types.NamespacedName{Namespace: "foo", Name: "Bar"}
			_, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: namspacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(client.GetCallCount()).To(Equal(2))
			Expect(client.StatusCallCount()).To(Equal(1))

			_, gotNamespacedName, _ := client.GetArgsForCall(0)
			Expect(gotNamespacedName).To(Equal(namspacedName))

			Expect(statusWriter.PatchCallCount()).To(Equal(1))
			_, _, patch, _ := statusWriter.PatchArgsForCall(0)
			Expect(patch.Type()).To(Equal(types.MergePatchType))
		})

		When("an error returns in get", func() {
			BeforeEach(func() {
				client.GetReturns(errors.New("UnknownError"))
			})

			It("returns the error", func() {
				namspacedName := types.NamespacedName{Namespace: "foo", Name: "Bar"}
				_, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: namspacedName})

				Expect(err).To(MatchError("get resource: UnknownError"))
				Expect(client.GetCallCount()).To(Equal(1))
				Expect(client.StatusCallCount()).To(Equal(0))
			})
		})

		When("an object is not found", func() {
			BeforeEach(func() {
				client.GetReturns(statusErrorNotFound{})
			})

			It("returns an empty result without error", func() {
				namspacedName := types.NamespacedName{Namespace: "foo", Name: "Bar"}
				_, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: namspacedName})

				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("#GetSecretSinkCondition", func() {
		It("returns nil for empty secret sink status", func() {
			secretSinkStatus := new(esapi.SecretSinkStatus)
			secretSinkConditionType := new(esapi.SecretSinkConditionType)

			Expect(GetSecretSinkCondition(*secretSinkStatus, *secretSinkConditionType)).To(BeNil())
		})

		It("returns correct condition for secret sink status", func() {
			secretSinkStatusCondition := esapi.SecretSinkStatusCondition{Type: esapi.SecretSinkReady}
			secretSinkStatus := esapi.SecretSinkStatus{Conditions: []esapi.SecretSinkStatusCondition{secretSinkStatusCondition}}
			secretSinkConditionType := esapi.SecretSinkReady

			Expect(GetSecretSinkCondition(secretSinkStatus, secretSinkConditionType)).To(Equal(&secretSinkStatusCondition))
		})
	})

	Describe("#SetSecretSinkCondition", func() {
		It("appends a condition", func() {
			secretSink := esapi.SecretSink{}

			secretSinkStatusCondition := esapi.SecretSinkStatusCondition{}
			secretSinkStatus := esapi.SecretSinkStatus{Conditions: []esapi.SecretSinkStatusCondition{secretSinkStatusCondition}}
			expected := esapi.SecretSink{Status: secretSinkStatus}
			Expect(SetSecretSinkCondition(secretSink, secretSinkStatusCondition)).To(Equal(expected))
		})

		It("changes an existing condition", func() {
			conditionStatusTrue := v1.ConditionTrue
			secretSinkWithCondition := esapi.SecretSink{Status: esapi.SecretSinkStatus{Conditions: []esapi.SecretSinkStatusCondition{
				{
					Status: conditionStatusTrue,
					Type:   esapi.SecretSinkReady,
				},
			}},
			}
			secretSinkStatusConditionTrue := esapi.SecretSinkStatusCondition{Status: conditionStatusTrue,
				Type:    esapi.SecretSinkReady,
				Message: "Update status",
			}

			got := SetSecretSinkCondition(secretSinkWithCondition, secretSinkStatusConditionTrue)
			Expect(len(got.Status.Conditions)).To(Equal(1))
			Expect(got.Status.Conditions[0]).To(Equal(secretSinkStatusConditionTrue))
		})
	})
	Describe("#GetSecret", func() {
		It("returns a secret if it exists", func() {
			sink := esapi.SecretSink{
				Spec: esapi.SecretSinkSpec{
					Selector: esapi.SecretSinkSelector{
						Secret: esapi.SecretSinkSecret{
							Name: "foo",
						},
					},
				},
			}
			sink.Namespace = "foobar"
			_, err := reconciler.GetSecret(context.TODO(), sink)
			Expect(err).To(BeNil())
			_, name, _ := client.GetArgsForCall(0)
			Expect(name.Namespace).To(Equal("foobar"))
			Expect(name.Name).To(Equal("foo"))

		})

		It("returns an error if it doesn't exist", func() {
			client.GetReturns(errors.New("secret not found"))
			_, err := reconciler.GetSecret(context.TODO(), esapi.SecretSink{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("#GetSecretStore", func() {
		sink := esapi.SecretSink{
			Spec: esapi.SecretSinkSpec{
				SecretStoreRefs: []esapi.SecretSinkStoreRef{
					{
						Name: "foo",
					},
				},
			},
		}
		sink.Namespace = "bar"

		clusterSink := esapi.SecretSink{
			Spec: esapi.SecretSinkSpec{
				SecretStoreRefs: []esapi.SecretSinkStoreRef{
					{
						Name: "foo",
						Kind: "ClusterSecretStore",
					},
				},
			},
		}

		It("returns a secretstore if it exists", func() {
			_, err := reconciler.GetSecretStores(context.TODO(), sink)
			Expect(err).To(BeNil())
			Expect(client.GetCallCount()).To(Equal(1))
			_, name, store := client.GetArgsForCall(0)
			Expect(name.Namespace).To(Equal("bar"))
			Expect(name.Name).To(Equal("foo"))
			Expect(store).To(BeAssignableToTypeOf(&v1beta1.SecretStore{}))
		})

		It("returns an error if it doesn't exist", func() {
			client.GetReturns(errors.New("secretstore not found"))
			_, err := reconciler.GetSecretStores(context.TODO(), sink)
			Expect(err).To(HaveOccurred())
		})

		It("returns a clustersecretstore if it exists", func() {
			_, err := reconciler.GetSecretStores(context.TODO(), clusterSink)
			Expect(err).To(BeNil())
			Expect(client.GetCallCount()).To(Equal(1))
			_, name, store := client.GetArgsForCall(0)
			Expect(store).To(BeAssignableToTypeOf(&v1beta1.ClusterSecretStore{}))
			Expect(name.Name).To(Equal("foo"))
		})
	})
	Describe("#SetSecretToProviders", func() {
		val := "bar"
		secret := &v1.Secret{
			Data: map[string][]byte{
				"foo": []byte(val),
			},
		}
		sink := esapi.SecretSink{
			Spec: esapi.SecretSinkSpec{
				SecretStoreRefs: []esapi.SecretSinkStoreRef{
					{
						Name: "foo",
					},
				},
				Data: []esapi.SecretSinkData{
					{
						Match: []esapi.SecretSinkMatch{
							{
								SecretKey: "foo",
								RemoteRefs: []esapi.SecretSinkRemoteRefs{
									{
										RemoteKey: "bar",
									},
								},
							},
						},
					},
				},
			},
		}
		sink.Namespace = "bar"

		secretStore := v1beta1.SecretStore{}
		stores := make([]v1beta1.GenericStore, 0)
		stores = append(stores, &secretStore)

		It("gets the provider and client and then sets the secret", func() {

			Expect(reconciler.SetSecretToProviders(context.TODO(), []v1beta1.GenericStore{}, sink, secret)).To(BeNil())
		})

		It("returns an error if it can't get a provider", func() {
			err := reconciler.SetSecretToProviders(context.TODO(), stores, sink, secret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errGetProviderFailed))
		})

		It("returns an if it can't get a client", func() {
			specWithProvider := v1beta1.SecretStoreSpec{
				Provider: &v1beta1.SecretStoreProvider{
					Fake: &v1beta1.FakeProvider{},
				},
			}
			fakeProvider.WithNew(func(context.Context, v1beta1.GenericStore, kubeclient.Client,
				string) (v1beta1.SecretsClient, error) {
				return nil, fmt.Errorf("Something went wrong")
			})
			secretStore = v1beta1.SecretStore{
				Spec: specWithProvider,
			}

			stores[0] = &secretStore
			err := reconciler.SetSecretToProviders(context.TODO(), stores, sink, secret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errGetSecretsClientFailed))
		})
		It("returns an error if set secret fails", func() {
			specWithProvider := v1beta1.SecretStoreSpec{
				Provider: &v1beta1.SecretStoreProvider{
					Fake: &v1beta1.FakeProvider{},
				},
			}
			fakeProvider.Reset()
			fakeProvider.WithSetSecret(fmt.Errorf("something went wrong"))
			secretStore = v1beta1.SecretStore{
				Spec: specWithProvider,
			}

			stores[0] = &secretStore
			err := reconciler.SetSecretToProviders(context.TODO(), stores, sink, secret)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(errSetSecretFailed, "foo", "", "something went wrong")))
		})
	})
})

func init() {
	fakeProvider = fake.New()
	v1beta1.ForceRegister(fakeProvider, &v1beta1.SecretStoreProvider{
		Fake: &v1beta1.FakeProvider{},
	})
}
