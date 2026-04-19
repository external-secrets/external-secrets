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

package register

import (
	"reflect"
	"testing"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	awsgen "github.com/external-secrets/external-secrets/providers/v2/aws/generator"
	fakegen "github.com/external-secrets/external-secrets/providers/v2/fake/generator"
)

func TestProviderOwnedGeneratorsAreRegistered(t *testing.T) {
	t.Helper()

	testCases := []struct {
		kind string
		want any
	}{
		{
			kind: string(genv1alpha1.GeneratorKindECRAuthorizationToken),
			want: &awsgen.ECRGenerator{},
		},
		{
			kind: string(genv1alpha1.GeneratorKindSTSSessionToken),
			want: &awsgen.STSGenerator{},
		},
		{
			kind: string(genv1alpha1.GeneratorKindFake),
			want: &fakegen.Generator{},
		},
	}

	for _, tc := range testCases {
		got, ok := genv1alpha1.GetGeneratorByName(tc.kind)
		if !ok {
			t.Fatalf("generator %q not registered", tc.kind)
		}
		if tName, gName := typeName(tc.want), typeName(got); tName != gName {
			t.Fatalf("generator %q = %s, want %s", tc.kind, gName, tName)
		}
	}
}

func typeName(v any) string {
	return reflect.TypeOf(v).String()
}
