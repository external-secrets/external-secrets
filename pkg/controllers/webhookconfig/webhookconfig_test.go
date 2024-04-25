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

package webhookconfig

import (
	"bytes"
	"context"
	"time"

	admissionregistration "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	pointer "k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const defaultCACert = `-----BEGIN CERTIFICATE-----
MIIDRjCCAi6gAwIBAgIBADANBgkqhkiG9w0BAQsFADA2MRkwFwYDVQQKExBleHRl
cm5hbC1zZWNyZXRzMRkwFwYDVQQDExBleHRlcm5hbC1zZWNyZXRzMB4XDTIyMDIx
NzEwMDYxMFoXDTMyMDIxNTExMDYxMFowNjEZMBcGA1UEChMQZXh0ZXJuYWwtc2Vj
cmV0czEZMBcGA1UEAxMQZXh0ZXJuYWwtc2VjcmV0czCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBAKSINgqU2dBdX8JpPjRHWSdpxuoltGl6xXmQHOhbTXAt
/STDu7oi6eiFgepQ2QHuWLGwZgbbYnEhtLvw4dUwPcLyv6WIdeiUSA4pdFxL7asc
WV4tjiRkRTJVrixJTxXpry/CsPqXBlvnu1YGESkrLOYCmA2xnDH8voEBbwYvXXB9
3g5rOJncSh/7g+H55ZFFyWrIPyDUnfwE3CREjZXpsagFhRYpkuRlXTnU6t0OTEEh
qLHlZ+ebUzL8NaegEgEHD32PrQPXpls1yurIrsA+I6McWkXGykykYHVK+1a1pL1g
e+PBkegtwtX+EmB2ux7PVVeB4TTYqzCKbnObW4mJLZkCAwEAAaNfMF0wDgYDVR0P
AQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFHgSu/Im2gyu4TU0
AWrMSFbtoVokMBsGA1UdEQQUMBKCEGV4dGVybmFsLXNlY3JldHMwDQYJKoZIhvcN
AQELBQADggEBAJU88jCcPsAHN8DKLu+QMCoKYbeftX4gXxyoijGSde2w2O8NPtMP
awu4Y5x3LNTwyIIxXi78UD0RI53GbUgHvS+X9v6CC2IZMS65xqKR+EsjzEh7Ldbm
vZoF4ZDnfb2s5SK6MeYf67BE7XWpGfbHmjt6h80xsYjL6ovcik+dlu/AixMyLslS
tDbMybAR8kR0zdQLYcZq7XEX5QsOO8qBn5rTfD6MiYik8ZrP7FqUMHyVpHiBuNio
krnSOvynvuA9mlf2F2727dMt2Ij9uER+9QnhWBQex1h8CwALmm2k9G5Gt+RjB8oe
lNjvmHAXUfOE/cbD7EP++X17kWt41FjmePc=
-----END CERTIFICATE-----
`

type testCase struct {
	vwc       *admissionregistration.ValidatingWebhookConfiguration
	service   *corev1.Service
	endpoints *corev1.Endpoints
	secret    *corev1.Secret
	assert    func()
}

var _ = Describe("ValidatingWebhookConfig reconcile", Ordered, func() {
	var test *testCase

	BeforeEach(func() {
		test = makeDefaultTestcase()
	})

	AfterEach(func() {
		ctx := context.Background()
		k8sClient.Delete(ctx, test.vwc)
		k8sClient.Delete(ctx, test.secret)
		k8sClient.Delete(ctx, test.service)
		k8sClient.Delete(ctx, test.endpoints)
	})

	// Should patch VWC
	PatchAndReady := func(tc *testCase) {
		tc.endpoints.Subsets = nil

		// endpoints become ready in a moment
		go func() {
			<-time.After(time.Second * 4)
			eps := makeEndpoints()
			err := k8sClient.Update(context.Background(), eps)
			Expect(err).ToNot(HaveOccurred())
		}()
		tc.assert = func() {
			Eventually(func() bool {
				// the controller should become ready at some point!
				err := reconciler.ReadyCheck(nil)
				return err == nil
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())

			Eventually(func() bool {
				var vwc admissionregistration.ValidatingWebhookConfiguration
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name: tc.vwc.Name,
				}, &vwc)
				if err != nil {
					return false
				}
				for _, wc := range vwc.Webhooks {
					if !bytes.Equal(wc.ClientConfig.CABundle, []byte(defaultCACert)) {
						return false
					}
					if wc.ClientConfig.Service == nil {
						return false
					}
					if wc.ClientConfig.Service.Name != ctrlSvcName {
						return false
					}
					if wc.ClientConfig.Service.Namespace != ctrlSvcNamespace {
						return false
					}
				}
				return true
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())
		}
	}

	IgnoreNoMatch := func(tc *testCase) {
		delete(tc.vwc.ObjectMeta.Labels, wellKnownLabelKey)
		tc.assert = func() {
			Consistently(func() bool {
				var vwc admissionregistration.ValidatingWebhookConfiguration

				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name: tc.vwc.Name,
				}, &vwc)
				if err != nil {
					return false
				}
				for _, wc := range vwc.Webhooks {
					if bytes.Equal(wc.ClientConfig.CABundle, []byte(defaultCACert)) {
						return false
					}
					if wc.ClientConfig.Service.Name == ctrlSvcName {
						return false
					}
					if wc.ClientConfig.Service.Namespace == ctrlSvcNamespace {
						return false
					}
				}
				return true
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())
		}
	}

	// Should patch and update VWC after requeue duration has passed
	PatchAndUpdate := func(tc *testCase) {
		foobar := "new value"
		// ca cert will change after some time
		go func() {
			<-time.After(time.Second * 4)
			sec := makeSecret()
			sec.Data[caCertName] = []byte(foobar)
			err := k8sClient.Update(context.Background(), sec)
			Expect(err).ToNot(HaveOccurred())
		}()
		tc.assert = func() {

			Eventually(func() bool {
				var vwc admissionregistration.ValidatingWebhookConfiguration
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name: tc.vwc.Name,
				}, &vwc)
				if err != nil {
					return false
				}
				for _, wc := range vwc.Webhooks {
					if !bytes.Equal(wc.ClientConfig.CABundle, []byte(foobar)) {
						return false
					}
					if wc.ClientConfig.Service == nil {
						return false
					}
					if wc.ClientConfig.Service.Name != ctrlSvcName {
						return false
					}
					if wc.ClientConfig.Service.Namespace != ctrlSvcNamespace {
						return false
					}
				}
				return true
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())
		}
	}

	DescribeTable("Controller Reconcile logic", func(muts ...func(tc *testCase)) {
		for _, mut := range muts {
			mut(test)
		}
		ctx := context.Background()

		err := k8sClient.Create(ctx, test.vwc)
		Expect(err).ToNot(HaveOccurred())

		err = k8sClient.Create(ctx, test.secret)
		Expect(err).ToNot(HaveOccurred())

		err = k8sClient.Create(ctx, test.service)
		Expect(err).ToNot(HaveOccurred())

		err = k8sClient.Create(ctx, test.endpoints)
		Expect(err).ToNot(HaveOccurred())

		test.assert()
	},

		Entry("should patch matching webhook configs", PatchAndReady),
		Entry("should update vwc with new ca cert after requeue duration", PatchAndUpdate),
		Entry("should ignore when vwc labels are missing", IgnoreNoMatch),
	)

})

func makeValidatingWebhookConfig() *admissionregistration.ValidatingWebhookConfiguration {
	return &admissionregistration.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "name-shouldnt-matter",
			Labels: map[string]string{
				wellKnownLabelKey: wellKnownLabelValue,
			},
		},
		Webhooks: []admissionregistration.ValidatingWebhook{
			{
				Name:                    "secretstores.external-secrets.io",
				SideEffects:             (*admissionregistration.SideEffectClass)(pointer.To(string(admissionregistration.SideEffectClassNone))),
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig: admissionregistration.WebhookClientConfig{
					CABundle: []byte("Cg=="),
					Service: &admissionregistration.ServiceReference{
						Name:      "noop",
						Namespace: "noop",
						Path:      pointer.To("/validate-secretstore"),
					},
				},
			},
			{
				Name:                    "clustersecretstores.external-secrets.io",
				SideEffects:             (*admissionregistration.SideEffectClass)(pointer.To(string(admissionregistration.SideEffectClassNone))),
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig: admissionregistration.WebhookClientConfig{
					CABundle: []byte("Cg=="),
					Service: &admissionregistration.ServiceReference{
						Name:      "noop",
						Namespace: "noop",
						Path:      pointer.To("/validate-clustersecretstore"),
					},
				},
			},
		},
	}
}

func makeSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrlSecretName,
			Namespace: ctrlSecretNamespace,
		},
		Data: map[string][]byte{
			caCertName: []byte(defaultCACert),
		},
	}
}

func makeService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrlSvcName,
			Namespace: ctrlSvcNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
}

func makeEndpoints() *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ctrlSvcName,
			Namespace: ctrlSvcNamespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.2.3.4",
					},
				},
			},
		},
	}
}

func makeDefaultTestcase() *testCase {
	return &testCase{
		assert: func() {
			// this is a noop by default
		},
		vwc:       makeValidatingWebhookConfig(),
		secret:    makeSecret(),
		service:   makeService(),
		endpoints: makeEndpoints(),
	}
}
