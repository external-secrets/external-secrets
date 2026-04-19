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

package controller

import (
	"testing"

	awsv2 "github.com/external-secrets/external-secrets/apis/provider/aws/v2alpha1"
	fakev2alpha1 "github.com/external-secrets/external-secrets/apis/provider/fake/v2alpha1"
	k8sv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestStoreRequeueIntervalDefault(t *testing.T) {
	flag := rootCmd.Flags().Lookup("store-requeue-interval")
	if flag == nil {
		t.Fatal("store-requeue-interval flag not found")
	}

	if flag.DefValue != "30s" {
		t.Fatalf("expected store-requeue-interval default 30s, got %q", flag.DefValue)
	}
}

func TestSchemeIncludesProviderV2Kinds(t *testing.T) {
	t.Helper()

	testCases := []struct {
		name string
		gvk  schema.GroupVersionKind
	}{
		{
			name: "aws secretsmanager",
			gvk:  awsv2.GroupVersion.WithKind(awsv2.SecretsManagerKind),
		},
		{
			name: "aws parameterstore",
			gvk:  awsv2.GroupVersion.WithKind(awsv2.ParameterStoreKind),
		},
		{
			name: "fake",
			gvk:  fakev2alpha1.GroupVersion.WithKind(fakev2alpha1.Kind),
		},
		{
			name: "kubernetes",
			gvk:  k8sv2alpha1.GroupVersion.WithKind(k8sv2alpha1.Kind),
		},
	}

	for _, tc := range testCases {
		if _, err := scheme.New(tc.gvk); err != nil {
			t.Fatalf("%s kind not registered in controller scheme: %v", tc.name, err)
		}
	}
}
