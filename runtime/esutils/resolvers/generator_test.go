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
package resolvers

import (
	"testing"

	"github.com/stretchr/testify/require"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

func TestClusterGeneratorToVirtualRDSIAMAuthToken(t *testing.T) {
	spec := &genv1alpha1.RDSIAMAuthTokenSpec{
		Controller: "rds-iam",
		Region:     "ap-southeast-2",
		Hostname:   "database.example.com",
		Port:       5432,
		Username:   "db_user",
	}
	clusterGenerator := &genv1alpha1.ClusterGenerator{
		Spec: genv1alpha1.ClusterGeneratorSpec{
			Kind: genv1alpha1.GeneratorKindRDSIAMAuthToken,
			Generator: genv1alpha1.GeneratorSpec{
				RDSIAMAuthTokenSpec: spec,
			},
		},
	}

	got, err := clusterGeneratorToVirtual(clusterGenerator)
	require.NoError(t, err)

	token, ok := got.(*genv1alpha1.RDSIAMAuthToken)
	require.True(t, ok)
	require.Equal(t, genv1alpha1.SchemeGroupVersion.String(), token.APIVersion)
	require.Equal(t, genv1alpha1.RDSIAMAuthTokenKind, token.Kind)
	require.Equal(t, *spec, token.Spec)
}

func TestClusterGeneratorToVirtualRDSIAMAuthTokenRequiresSpec(t *testing.T) {
	clusterGenerator := &genv1alpha1.ClusterGenerator{
		Spec: genv1alpha1.ClusterGeneratorSpec{
			Kind: genv1alpha1.GeneratorKindRDSIAMAuthToken,
		},
	}

	_, err := clusterGeneratorToVirtual(clusterGenerator)
	require.EqualError(t, err, "when kind is RDSIAMAuthToken, RDSIAMAuthTokenSpec must be set")
}
