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

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretsink/internal/fakes"
)

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
			Expect(client.GetCallCount()).To(Equal(1))
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
		secret := esapi.SecretSinkSecret{Name: ""}
		selector := esapi.SecretSinkSelector{Secret: secret}
		secretSink := esapi.SecretSink{}

		secretSinkStatusCondition := esapi.SecretSinkStatusCondition{}
		secretSinkStatus := esapi.SecretSinkStatus{Conditions: []esapi.SecretSinkStatusCondition{secretSinkStatusCondition}}
		secretSinkSpec := esapi.SecretSinkSpec{SecretStoreRefs: nil, Selector: selector, Data: nil}

		It("returns a secretsink", func() {
			toEqual := esapi.SecretSink{Spec: secretSinkSpec, Status: secretSinkStatus}
			Expect(SetSecretSinkCondition(secretSink, secretSinkStatusCondition)).To(Equal(toEqual))
		})

		It("appends secretsink condition if new", func() {
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

			Expect(SetSecretSinkCondition(secretSinkWithCondition, secretSinkStatusConditionTrue).Status.Conditions[0]).To(Equal(secretSinkStatusConditionTrue))

		})

	})
})

// currentCond != nil && currentCond.Status == condition.Status (SecretSinkReady SecretSinkConditionType = "Ready")
// currentCond.Reason != condition.Reason && currentCond.Message != condition.Message
