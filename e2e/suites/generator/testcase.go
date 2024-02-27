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
	"time"

	//nolint
	. "github.com/onsi/gomega"

	// nolint
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type testCase struct {
	Framework      *framework.Framework
	ExternalSecret *esv1beta1.ExternalSecret
	Generator      client.Object
	AfterSync      func(*v1.Secret)
}

var (
	generatorName = "myfake"
)

func generatorTableFunc(f *framework.Framework, tweaks ...func(*testCase)) {
	tc := &testCase{
		Framework: f,
		ExternalSecret: &esv1beta1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-es",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1beta1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: time.Second * 5},
				Target: esv1beta1.ExternalSecretTarget{
					Name: "generated-secret",
				},
			},
		},
	}

	for _, t := range tweaks {
		t(tc)
	}

	err := f.CRClient.Create(context.Background(), tc.Generator)
	Expect(err).ToNot(HaveOccurred())

	err = f.CRClient.Create(context.Background(), tc.ExternalSecret)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() bool {
		var es esv1beta1.ExternalSecret
		err = f.CRClient.Get(context.Background(), types.NamespacedName{
			Namespace: tc.ExternalSecret.Namespace,
			Name:      tc.ExternalSecret.Name,
		}, &es)
		if err != nil {
			return false
		}

		cond := getESCond(es.Status, esv1beta1.ExternalSecretReady)
		if cond == nil || cond.Status != v1.ConditionTrue {
			return false
		}
		return true
	}).WithTimeout(time.Second * 30).Should(BeTrue())

	var secret v1.Secret
	err = f.CRClient.Get(context.Background(), types.NamespacedName{
		Namespace: tc.ExternalSecret.Namespace,
		Name:      tc.ExternalSecret.Spec.Target.Name,
	}, &secret)
	Expect(err).ToNot(HaveOccurred())

	tc.AfterSync(&secret)
}

// getESCond returns the condition with the provided type.
func getESCond(status esv1beta1.ExternalSecretStatus, condType esv1beta1.ExternalSecretConditionType) *esv1beta1.ExternalSecretStatusCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}
