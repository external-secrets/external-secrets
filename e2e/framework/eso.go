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

package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	//nolint
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/external-secrets/external-secrets-e2e/framework/log"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// WaitForSecretValue waits until a secret comes into existence and compares the secret.Data
// with the provided values.
func (f *Framework) WaitForSecretValue(namespace, name string, expected *v1.Secret) (*v1.Secret, error) {
	secret := &v1.Secret{}
	err := wait.PollImmediate(time.Second*10, time.Minute, func() (bool, error) {
		err := f.CRClient.Get(context.Background(), types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, secret)
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return equalSecrets(expected, secret), nil
	})
	return secret, err
}

func (f *Framework) printESDebugLogs(esName, esNamespace string) {
	// fetch es and print status condition
	var es esv1beta1.ExternalSecret
	err := f.CRClient.Get(context.Background(), types.NamespacedName{
		Name:      esName,
		Namespace: esNamespace,
	}, &es)
	Expect(err).ToNot(HaveOccurred())
	log.Logf("resourceVersion=%s", es.Status.SyncedResourceVersion)
	for _, cond := range es.Status.Conditions {
		log.Logf("condition: status=%s type=%s reason=%s message=%s", cond.Status, cond.Type, cond.Reason, cond.Message)
	}
	// list events for given
	evs, err := f.KubeClientSet.CoreV1().Events(esNamespace).List(context.Background(), metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + esName + ",involvedObject.kind=ExternalSecret",
	})
	Expect(err).ToNot(HaveOccurred())
	for _, ev := range evs.Items {
		log.Logf("ev reason=%s message=%s", ev.Reason, ev.Message)
	}

	// print most recent logs of default eso installation
	podList, err := f.KubeClientSet.CoreV1().Pods("default").List(
		context.Background(),
		metav1.ListOptions{LabelSelector: "app.kubernetes.io/instance=eso,app.kubernetes.io/name=external-secrets"})
	Expect(err).ToNot(HaveOccurred())
	numLines := int64(60)
	for i := range podList.Items {
		pod := podList.Items[i]
		for _, con := range pod.Spec.Containers {
			for _, b := range []bool{true, false} {
				resp := f.KubeClientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{
					Container: con.Name,
					Previous:  b,
					TailLines: &numLines,
				}).Do(context.TODO())
				err := resp.Error()
				if err != nil {
					continue
				}
				logs, err := resp.Raw()
				if err != nil {
					continue
				}
				log.Logf("[%s]: %s", "eso", string(logs))
			}
		}
	}
}

func equalSecrets(exp, ts *v1.Secret) bool {
	if exp.Type != ts.Type {
		return false
	}

	// secret contains label owner which must be ignored
	delete(ts.ObjectMeta.Labels, esv1beta1.LabelOwner)
	if len(ts.ObjectMeta.Labels) == 0 {
		ts.ObjectMeta.Labels = nil
	}

	expLabels, _ := json.Marshal(exp.ObjectMeta.Labels)
	tsLabels, _ := json.Marshal(ts.ObjectMeta.Labels)
	if !bytes.Equal(expLabels, tsLabels) {
		return false
	}

	// secret contains data hash property which must be ignored
	delete(ts.ObjectMeta.Annotations, esv1beta1.AnnotationDataHash)
	if len(ts.ObjectMeta.Annotations) == 0 {
		ts.ObjectMeta.Annotations = nil
	}
	expAnnotations, _ := json.Marshal(exp.ObjectMeta.Annotations)
	tsAnnotations, _ := json.Marshal(ts.ObjectMeta.Annotations)
	if !bytes.Equal(expAnnotations, tsAnnotations) {
		return false
	}

	expData, _ := json.Marshal(exp.Data)
	tsData, _ := json.Marshal(ts.Data)
	return bytes.Equal(expData, tsData)
}
