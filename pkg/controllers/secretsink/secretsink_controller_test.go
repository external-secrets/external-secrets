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
	"github.com/external-secrets/external-secrets/pkg/controllers/secretsink/internal"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("#Reconcile", func() {
	var (
		client       *internal.FakeClient
		statusWriter *internal.FakeStatusWriter
		reconciler   *Reconciler
	)

	BeforeEach(func() {
		client = new(internal.FakeClient)
		statusWriter = new(internal.FakeStatusWriter)
		client.StatusReturns(statusWriter)
		reconciler = &Reconciler{client, logr.Discard(), nil, nil, 0, ""}
	})

	It("succeeds", func() {
		namspacedName := types.NamespacedName{Namespace: "foo", Name: "Bar"}
		_, err := reconciler.Reconcile(nil, ctrl.Request{NamespacedName: namspacedName})
		Expect(err).NotTo(HaveOccurred())
		Expect(client.GetCallCount()).To(Equal(1))
		Expect(client.StatusCallCount()).To(Equal(1))

		_, gotNamespacedName, _ := client.GetArgsForCall(0)
		Expect(gotNamespacedName).To(Equal(namspacedName))

		Expect(statusWriter.PatchCallCount()).To(Equal(1))
		_, _, patch, _ := statusWriter.PatchArgsForCall(0)
		Expect(patch.Type()).To(Equal(types.MergePatchType))
	})
})
