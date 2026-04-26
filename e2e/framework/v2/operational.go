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

package v2

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"

	. "github.com/onsi/gomega"
)

type BackendTarget struct {
	Namespace        string
	DeploymentName   string
	PodLabelSelector string
}

func ScaleDeploymentBySelector(f *framework.Framework, target BackendTarget, replicas int32) {
	deploymentName := target.DeploymentName
	if deploymentName == "" {
		deploymentName = findDeploymentNameBySelector(f, target)
	}
	scaleDeployment(f, target.Namespace, deploymentName, replicas)
}

func ScaleDeploymentBySelectorAndWait(f *framework.Framework, target BackendTarget, replicas int32, timeout time.Duration) {
	ScaleDeploymentBySelector(f, target, replicas)
	WaitForBackendTargetRunningReplicas(f, target, int(replicas), timeout)
}

func DeleteOneProviderPodBySelector(f *framework.Framework, target BackendTarget) {
	Expect(target.Namespace).NotTo(BeEmpty(), "backend target namespace must be set")
	Expect(target.PodLabelSelector).NotTo(BeEmpty(), "backend target pod label selector must be set")

	selector, err := labels.Parse(target.PodLabelSelector)
	Expect(err).NotTo(HaveOccurred())

	var podList corev1.PodList
	Expect(f.CRClient.List(context.Background(), &podList, &client.ListOptions{
		Namespace:     target.Namespace,
		LabelSelector: selector,
	})).To(Succeed())

	foundRunningPod := false
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Status.Phase == corev1.PodRunning {
			foundRunningPod = true
			Expect(f.CRClient.Delete(context.Background(), pod)).To(Succeed())
			return
		}
	}

	Expect(foundRunningPod).To(BeTrue(), fmt.Sprintf("no running pod found for selector %s in namespace %s", target.PodLabelSelector, target.Namespace))
}

func findDeploymentNameBySelector(f *framework.Framework, target BackendTarget) string {
	Expect(target.Namespace).NotTo(BeEmpty(), "backend target namespace must be set")
	Expect(target.PodLabelSelector).NotTo(BeEmpty(), "backend target pod label selector must be set")

	selector, err := labels.Parse(target.PodLabelSelector)
	Expect(err).NotTo(HaveOccurred())

	var deploymentList appsv1.DeploymentList
	Expect(f.CRClient.List(context.Background(), &deploymentList, &client.ListOptions{
		Namespace: target.Namespace,
	})).To(Succeed())

	matches := make([]appsv1.Deployment, 0, len(deploymentList.Items))
	for _, deployment := range deploymentList.Items {
		if selector.Matches(labels.Set(deployment.Spec.Template.GetLabels())) {
			matches = append(matches, deployment)
		}
	}

	Expect(matches).NotTo(BeEmpty(), "no deployment found for selector %s", target.PodLabelSelector)
	Expect(matches).To(HaveLen(1), "expected one deployment for selector %s", target.PodLabelSelector)

	return matches[0].Name
}

func WaitForBackendTargetRunningReplicas(f *framework.Framework, target BackendTarget, expectedReplicas int, timeout time.Duration) {
	Expect(target.Namespace).NotTo(BeEmpty(), "backend target namespace must be set")
	Expect(target.PodLabelSelector).NotTo(BeEmpty(), "backend target pod label selector must be set")

	selector, err := labels.Parse(target.PodLabelSelector)
	Expect(err).NotTo(HaveOccurred())

	Eventually(func(g Gomega) {
		var podList corev1.PodList
		g.Expect(f.CRClient.List(context.Background(), &podList, &client.ListOptions{
			Namespace:     target.Namespace,
			LabelSelector: selector,
		})).To(Succeed())

		running := 0
		for i := range podList.Items {
			if podList.Items[i].Status.Phase == corev1.PodRunning {
				running++
			}
		}
		g.Expect(running).To(Equal(expectedReplicas))
	}, timeout, time.Second).Should(Succeed())
}

func scaleDeployment(f *framework.Framework, namespace, name string, replicas int32) {
	Expect(namespace).NotTo(BeEmpty(), "deployment namespace must be set")
	Expect(name).NotTo(BeEmpty(), "deployment name must be set")

	Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var deployment appsv1.Deployment
		if err := f.CRClient.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, &deployment); err != nil {
			return err
		}
		deployment.Spec.Replicas = &replicas
		return f.CRClient.Update(context.Background(), &deployment)
	})).To(Succeed())
}
