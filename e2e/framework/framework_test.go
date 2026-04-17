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

package framework

import (
	"testing"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	crfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRefreshClientsReloadsFrameworkClients(t *testing.T) {
	t.Helper()

	originalNewConfig := newFrameworkConfig
	t.Cleanup(func() {
		newFrameworkConfig = originalNewConfig
	})

	wantConfig := &rest.Config{Host: "https://refreshed.example"}
	wantClientset := &kubernetes.Clientset{}
	wantCRClient := crfake.NewClientBuilder().Build()
	newFrameworkConfig = func() (*rest.Config, *kubernetes.Clientset, crclient.Client) {
		return wantConfig, wantClientset, wantCRClient
	}

	f := &Framework{
		KubeConfig:    &rest.Config{Host: "https://stale.example"},
		KubeClientSet: &kubernetes.Clientset{},
		CRClient:      crfake.NewClientBuilder().Build(),
	}

	f.refreshClients()

	if f.KubeConfig != wantConfig {
		t.Fatalf("expected refreshed kube config")
	}
	if f.KubeClientSet != wantClientset {
		t.Fatalf("expected refreshed kube clientset")
	}
	if f.CRClient != wantCRClient {
		t.Fatalf("expected refreshed controller-runtime client")
	}
}
