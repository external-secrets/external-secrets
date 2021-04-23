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
package externalsecret

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

var (
	fakeProvider *fake.Client
	metric       dto.Metric
	timeout      = time.Second * 5
	interval     = time.Millisecond * 250
)

var _ = Describe("ExternalSecret controller", func() {
	const (
		ExternalSecretName             = "test-es"
		ExternalSecretStore            = "test-store"
		ExternalSecretTargetSecretName = "test-secret"
	)

	var ExternalSecretNamespace string

	BeforeEach(func() {
		var err error
		ExternalSecretNamespace, err = CreateNamespace("test-ns", k8sClient)
		Expect(err).ToNot(HaveOccurred())
		Expect(k8sClient.Create(context.Background(), &esv1alpha1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretStore,
				Namespace: ExternalSecretNamespace,
			},
			Spec: esv1alpha1.SecretStoreSpec{
				Provider: &esv1alpha1.SecretStoreProvider{
					AWS: &esv1alpha1.AWSProvider{
						Service: esv1alpha1.AWSServiceSecretsManager,
					},
				},
			},
		})).To(Succeed())

		metric.Reset()
		syncCallsTotal.Reset()
		syncCallsError.Reset()
		externalSecretCondition.Reset()
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ExternalSecretNamespace,
			},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())
		Expect(k8sClient.Delete(context.Background(), &esv1alpha1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretStore,
				Namespace: ExternalSecretNamespace,
			},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())
	})

	Context("When creating an ExternalSecret", func() {
		It("should set the condition eventually", func() {
			ctx := context.Background()
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			esLookupKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}
			createdES := &esv1alpha1.ExternalSecret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, esLookupKey, createdES)
				if err != nil {
					return false
				}
				cond := GetExternalSecretCondition(createdES.Status, esv1alpha1.ExternalSecretReady)
				if cond == nil || cond.Status != v1.ConditionTrue {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 1.0)).To(BeTrue())

			Eventually(func() float64 {
				Expect(syncCallsTotal.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue()
			}, timeout, interval).Should(Equal(2.0))
		})
	})

	Context("When updating an ExternalSecret", func() {
		It("should increment the syncCallsTotal metric", func() {
			ctx := context.Background()
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
				},
			}

			Expect(k8sClient.Create(ctx, es)).Should(Succeed())

			createdES := &esv1alpha1.ExternalSecret{}

			Eventually(func() error {
				esLookupKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}

				err := k8sClient.Get(ctx, esLookupKey, createdES)
				if err != nil {
					return err
				}

				createdES.Spec.RefreshInterval = &metav1.Duration{Duration: 10 * time.Second}

				err = k8sClient.Update(ctx, createdES)
				if err != nil {
					return err
				}

				return nil
			}, timeout, interval).Should(Succeed())

			Eventually(func() float64 {
				Expect(syncCallsTotal.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue()
			}, timeout, interval).Should(Equal(3.0))
		})
	})

	Context("When syncing ExternalSecret value", func() {
		It("should set the secret value and sync labels/annotations", func() {
			ctx := context.Background()
			const targetProp = "targetProperty"
			const secretVal = "someValue"
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
					Labels: map[string]string{
						"fooobar": "bazz",
						"bazzing": "booze",
					},
					Annotations: map[string]string{
						"hihihih":   "hehehe",
						"harharhra": "yallayalla",
					},
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
					Data: []esv1alpha1.ExternalSecretData{
						{
							SecretKey: targetProp,
							RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
								Key:      "barz",
								Property: "bang",
							},
						},
					},
				},
			}

			fakeProvider.WithGetSecret([]byte(secretVal), nil)
			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace}
			syncedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, syncedSecret)
				if err != nil {
					return false
				}
				v := syncedSecret.Data[targetProp]
				return string(v) == secretVal
			}, timeout, interval).Should(BeTrue())
			Expect(syncedSecret.ObjectMeta.Labels).To(BeEquivalentTo(es.ObjectMeta.Labels))
			Expect(syncedSecret.ObjectMeta.Annotations).To(BeEquivalentTo(es.ObjectMeta.Annotations))
		})

		It("should set the secret value and use the provided secret template", func() {
			By("creating an ExternalSecret")
			ctx := context.Background()
			const targetProp = "targetProperty"
			const secretVal = "someValue"
			const templateSecretKey = "tplkey"
			const templateSecretVal = "{{ .targetProperty | toString | upper }}"
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
					Labels: map[string]string{
						"fooobar": "bazz",
					},
					Annotations: map[string]string{
						"hihihih": "hehehe",
					},
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
						Template: &esv1alpha1.ExternalSecretTemplate{
							Metadata: esv1alpha1.ExternalSecretTemplateMetadata{
								Labels: map[string]string{
									"foos": "ball",
								},
								Annotations: map[string]string{
									"hihi": "ga",
								},
							},
							Data: map[string][]byte{
								templateSecretKey: []byte(templateSecretVal),
							},
						},
					},
					Data: []esv1alpha1.ExternalSecretData{
						{
							SecretKey: targetProp,
							RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
								Key:      "barz",
								Property: "bang",
							},
						},
					},
				},
			}

			fakeProvider.WithGetSecret([]byte(secretVal), nil)
			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace}
			syncedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, syncedSecret)
				if err != nil {
					return false
				}
				v1 := syncedSecret.Data[targetProp]
				v2 := syncedSecret.Data[templateSecretKey]
				return string(v1) == secretVal && string(v2) == "SOMEVALUE" // templated
			}, timeout, interval).Should(BeTrue())
			Expect(syncedSecret.ObjectMeta.Labels).To(BeEquivalentTo(
				es.Spec.Target.Template.Metadata.Labels))
			Expect(syncedSecret.ObjectMeta.Annotations).To(BeEquivalentTo(
				es.Spec.Target.Template.Metadata.Annotations))
		})

		It("should refresh secret value", func() {
			ctx := context.Background()
			const targetProp = "targetProperty"
			const secretVal = "someValue"
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Second},
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
					Data: []esv1alpha1.ExternalSecretData{
						{
							SecretKey: targetProp,
							RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
								Key: "barz",
							},
						},
					},
				},
			}

			fakeProvider.WithGetSecret([]byte(secretVal), nil)
			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace}
			syncedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, syncedSecret)
				if err != nil {
					return false
				}
				v := syncedSecret.Data[targetProp]
				return string(v) == secretVal
			}, timeout, interval).Should(BeTrue())

			newValue := "NEW VALUE"
			fakeProvider.WithGetSecret([]byte(newValue), nil)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, syncedSecret)
				if err != nil {
					return false
				}
				v := syncedSecret.Data[targetProp]
				return string(v) == newValue
			}, timeout, interval).Should(BeTrue())
		})

		It("should fetch secrets using dataFrom", func() {
			ctx := context.Background()
			const secretVal = "someValue"
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
					DataFrom: []esv1alpha1.ExternalSecretDataRemoteRef{
						{
							Key: "barz",
						},
					},
				},
			}

			fakeProvider.WithGetSecretMap(map[string][]byte{
				"foo": []byte("bar"),
				"baz": []byte("bang"),
			}, nil)
			fakeProvider.WithGetSecret([]byte(secretVal), nil)
			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace}
			syncedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, syncedSecret)
				if err != nil {
					return false
				}
				x := syncedSecret.Data["foo"]
				y := syncedSecret.Data["baz"]
				return string(x) == "bar" && string(y) == "bang"
			}, timeout, interval).Should(BeTrue())
		})

		It("should set an error condition when provider errors", func() {
			ctx := context.Background()
			const targetProp = "targetProperty"
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
					Data: []esv1alpha1.ExternalSecretData{
						{
							SecretKey: targetProp,
							RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
								Key:      "barz",
								Property: "bang",
							},
						},
					},
				},
			}

			fakeProvider.WithGetSecret(nil, fmt.Errorf("artificial testing error"))
			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			esLookupKey := types.NamespacedName{
				Name:      ExternalSecretName,
				Namespace: ExternalSecretNamespace}
			createdES := &esv1alpha1.ExternalSecret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, esLookupKey, createdES)
				if err != nil {
					return false
				}
				// condition must be false
				cond := GetExternalSecretCondition(createdES.Status, esv1alpha1.ExternalSecretReady)
				if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1alpha1.ConditionReasonSecretSyncedError {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Eventually(func() float64 {
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue()
			}, timeout, interval).Should(Equal(2.0))

			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		})

		It("should set an error condition when store does not exist", func() {
			ctx := context.Background()
			const targetProp = "targetProperty"
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: "storeshouldnotexist",
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
					Data: []esv1alpha1.ExternalSecretData{
						{
							SecretKey: targetProp,
							RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
								Key:      "barz",
								Property: "bang",
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			esLookupKey := types.NamespacedName{
				Name:      ExternalSecretName,
				Namespace: ExternalSecretNamespace}
			createdES := &esv1alpha1.ExternalSecret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, esLookupKey, createdES)
				if err != nil {
					return false
				}
				// condition must be false
				cond := GetExternalSecretCondition(createdES.Status, esv1alpha1.ExternalSecretReady)
				if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1alpha1.ConditionReasonSecretSyncedError {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Eventually(func() float64 {
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue()
			}, timeout, interval).Should(Equal(2.0))

			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		})

		It("should set an error condition when store provider constructor fails", func() {
			ctx := context.Background()
			const targetProp = "targetProperty"
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
					Data: []esv1alpha1.ExternalSecretData{
						{
							SecretKey: targetProp,
							RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
								Key:      "barz",
								Property: "bang",
							},
						},
					},
				},
			}

			fakeProvider.WithNew(func(context.Context, esv1alpha1.GenericStore, client.Client,
				string) (provider.SecretsClient, error) {
				return nil, fmt.Errorf("artificial constructor error")
			})
			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			esLookupKey := types.NamespacedName{
				Name:      ExternalSecretName,
				Namespace: ExternalSecretNamespace}
			createdES := &esv1alpha1.ExternalSecret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, esLookupKey, createdES)
				if err != nil {
					return false
				}
				// condition must be false
				cond := GetExternalSecretCondition(createdES.Status, esv1alpha1.ExternalSecretReady)
				if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1alpha1.ConditionReasonSecretSyncedError {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Eventually(func() float64 {
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue()
			}, timeout, interval).Should(Equal(2.0))

			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		})

		It("should not process stores with mismatching controller field", func() {
			ctx := context.Background()
			storeName := "example-ts-foo"
			Expect(k8sClient.Create(context.Background(), &esv1alpha1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      storeName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.SecretStoreSpec{
					Controller: "some-other-controller",
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Service: esv1alpha1.AWSServiceSecretsManager,
						},
					},
				},
			})).To(Succeed())
			defer func() {
				Expect(k8sClient.Delete(context.Background(), &esv1alpha1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      storeName,
						Namespace: ExternalSecretNamespace,
					},
				})).To(Succeed())
			}()
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: storeName,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
					Data: []esv1alpha1.ExternalSecretData{
						{
							SecretKey: "doesnothing",
							RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
								Key:      "barz",
								Property: "bang",
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretName,
				Namespace: ExternalSecretNamespace,
			}

			// COND
			createdES := &esv1alpha1.ExternalSecret{}
			Consistently(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, createdES)
				if err != nil {
					return false
				}
				cond := GetExternalSecretCondition(createdES.Status, esv1alpha1.ExternalSecretReady)
				return cond == nil
			}, timeout, interval).Should(BeTrue())

			// Condition True and False should be 0, since the Condition was not created
			Eventually(func() float64 {
				Expect(externalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1alpha1.ExternalSecretReady), string(v1.ConditionTrue)).Write(&metric)).To(Succeed())
				return metric.GetGauge().GetValue()
			}, timeout, interval).Should(Equal(0.0))

			Eventually(func() float64 {
				Expect(externalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1alpha1.ExternalSecretReady), string(v1.ConditionFalse)).Write(&metric)).To(Succeed())
				return metric.GetGauge().GetValue()
			}, timeout, interval).Should(Equal(0.0))

			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		})
	})
})

// CreateNamespace creates a new namespace in the cluster.
func CreateNamespace(baseName string, c client.Client) (string, error) {
	genName := fmt.Sprintf("ctrl-test-%v", baseName)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: genName,
		},
	}
	var err error
	err = wait.Poll(time.Second, 10*time.Second, func() (bool, error) {
		err = c.Create(context.Background(), ns)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return ns.Name, nil
}

func externalSecretConditionShouldBe(name, ns string, ct esv1alpha1.ExternalSecretConditionType, cs v1.ConditionStatus, v float64) bool {
	return Eventually(func() float64 {
		Expect(externalSecretCondition.WithLabelValues(name, ns, string(ct), string(cs)).Write(&metric)).To(Succeed())
		return metric.GetGauge().GetValue()
	}, timeout, interval).Should(Equal(v))
}

func init() {
	fakeProvider = fake.New()
	schema.ForceRegister(fakeProvider, &esv1alpha1.SecretStoreProvider{
		AWS: &esv1alpha1.AWSProvider{
			Service: esv1alpha1.AWSServiceSecretsManager,
		},
	})
}
