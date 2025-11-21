// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Copyright External Secrets Inc. All Rights Reserved
package neo4j

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	neo4jSDK "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcgNeo4j "github.com/testcontainers/testcontainers-go/modules/neo4j"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	testUser                = "generated_user"
	testPass                = "strongpassword"
	testNamespace           = "default"
	testSecretName          = "testpass"
	testSecretKey           = "password"
	testGeneratedSecretName = "userpass"
)

// mockClient implements the client.Client interface for testing.
type generatorMockClient struct {
	client.Client
	userPassword []byte
	t            *testing.T
}

// Get implements client.Client.
func (m generatorMockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	m.t.Helper()
	if key.Name == testSecretName {
		obj.(*corev1.Secret).Data = map[string][]byte{
			testSecretKey: []byte(testPass),
		}
	} else if key.Name == testGeneratedSecretName {
		obj.(*corev1.Secret).Data = map[string][]byte{
			testSecretKey: m.userPassword,
		}
	}
	return nil
}

func setupNeo4jContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	t.Helper()

	// Start Neo4j container
	neo4jContainer, err := tcgNeo4j.Run(ctx,
		"neo4j:2025.04.0",
		tcgNeo4j.WithAdminPassword(testPass),
	)
	require.NoError(t, err)

	// Automatically clean up the container when the test ends
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(neo4jContainer); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	})

	port, err := neo4jContainer.MappedPort(ctx, "7687")
	require.NoError(t, err)

	host, err := neo4jContainer.Host(ctx)
	require.NoError(t, err)

	uri := fmt.Sprintf("bolt://%s:%s", host, port.Port())
	return neo4jContainer, uri
}

func newGeneratorSpec(t *testing.T, uri, user string) *genv1alpha1.Neo4j {
	t.Helper()
	return &genv1alpha1.Neo4j{
		Spec: genv1alpha1.Neo4jSpec{
			Auth: genv1alpha1.Neo4jAuth{
				URI: uri,
				Basic: &genv1alpha1.Neo4jBasicAuth{
					Username: "neo4j",
					Password: esmeta.SecretKeySelector{
						Name: testSecretName,
						Key:  testSecretKey,
					},
				},
			},
			User: &genv1alpha1.Neo4jUser{
				User:     user,
				Roles:    []string{"reader"},
				Provider: genv1alpha1.Neo4jAuthProviderNative,
			},
		},
	}
}

func closeDriver(t *testing.T, ctx context.Context, driver neo4jSDK.DriverWithContext) {
	err := driver.Close(ctx)
	if err != nil {
		t.Logf("failed to close driver: %v", err)
	}
}

type Neo4jTestSuite struct {
	suite.Suite
	ctx    context.Context
	uri    string
	client generatorMockClient
}

func TestNeo4jGeneratorTestSuite(t *testing.T) {
	suite.Run(t, new(Neo4jTestSuite))
}

func (s *Neo4jTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.client = generatorMockClient{t: s.T()}
	neo4jContainer, uri := setupNeo4jContainer(s.T(), s.ctx)
	s.uri = uri

	require.True(s.T(), neo4jContainer.IsRunning())
}

func (s *Neo4jTestSuite) TestNeo4jGeneratorIntegration() {
	// build generator input
	user := fmt.Sprintf("%s_TestNeo4jGeneratorIntegration", testUser)
	spec := newGeneratorSpec(s.T(), s.uri, user)

	specJSON, _ := yaml.Marshal(spec)

	// call Generate()
	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)

	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "user")
	regex := regexp.MustCompile(fmt.Sprintf(`%s_[a-zA-Z0-9]{8}`, user))
	assert.Regexp(s.T(), regex, string(result["user"]))
	require.Contains(s.T(), result, testSecretKey)

	status, err := parseStatus(rawStatus.Raw)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), status.User, string(result["user"]))

	// Verify that the user was created in the Neo4j database
	customClient := generatorMockClient{t: s.T(), userPassword: result[testSecretKey]}

	driver, err := newDriver(s.ctx, &genv1alpha1.Neo4jAuth{
		URI: s.uri,
		Basic: &genv1alpha1.Neo4jBasicAuth{
			Username: string(result["user"]),
			Password: esmeta.SecretKeySelector{
				Name: testGeneratedSecretName,
				Key:  testSecretKey,
			},
		},
	}, customClient, testNamespace)
	require.NoError(s.T(), err)
	defer closeDriver(s.T(), s.ctx, driver)
	err = driver.VerifyConnectivity(s.ctx)
	require.NoError(s.T(), err)
}

func (s *Neo4jTestSuite) TestNeo4jGeneratorWithRandomSufix() {
	// build generator input
	spec := newGeneratorSpec(s.T(), s.uri, testUser)
	specJSON, _ := yaml.Marshal(spec)

	// call Generate()
	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)

	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "user")

	userRegex := regexp.MustCompile(fmt.Sprintf(`%s_[a-zA-Z0-9]{8}`, testUser))
	assert.Regexp(s.T(), userRegex, string(result["user"]))
	require.Contains(s.T(), result, testSecretKey)

	status, err := parseStatus(rawStatus.Raw)
	require.NoError(s.T(), err)

	assert.Regexp(s.T(), userRegex, status.User)
}

func (s *Neo4jTestSuite) TestNeo4jCleanup() {
	// build generator input
	user := fmt.Sprintf("%s_TestNeo4jCleanup", testUser)
	spec := newGeneratorSpec(s.T(), s.uri, user)
	specJSON, _ := yaml.Marshal(spec)

	// call Generate()
	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)

	require.NoError(s.T(), err)

	_, err = parseStatus(rawStatus.Raw)
	require.NoError(s.T(), err)

	// Verify that the user was created in the Neo4j database
	customClient := generatorMockClient{t: s.T(), userPassword: result[testSecretKey]}

	userAuth := &genv1alpha1.Neo4jBasicAuth{
		Username: string(result["user"]),
		Password: esmeta.SecretKeySelector{
			Name: testGeneratedSecretName,
			Key:  testSecretKey,
		},
	}

	driver, err := newDriver(s.ctx, &genv1alpha1.Neo4jAuth{
		URI:   s.uri,
		Basic: userAuth,
	}, customClient, testNamespace)
	require.NoError(s.T(), err)
	err = driver.VerifyConnectivity(s.ctx)
	require.NoError(s.T(), err)
	err = driver.Close(s.ctx)
	require.NoError(s.T(), err)

	// Cleanup
	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, rawStatus, customClient, testNamespace)
	require.NoError(s.T(), err)

	driver, err = newDriver(s.ctx, &genv1alpha1.Neo4jAuth{
		URI:   s.uri,
		Basic: userAuth,
	}, customClient, testNamespace)
	require.NoError(s.T(), err)
	defer closeDriver(s.T(), s.ctx, driver)
	err = driver.VerifyConnectivity(s.ctx)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "Neo.ClientError.Security.Unauthorized")
}

func (s *Neo4jTestSuite) TestNeo4jCleanupAfterUserDBManipulation() {
	// build generator input
	user := fmt.Sprintf("%s_TestNeo4jCleanupAfterUserDBManipulation", testUser)
	spec := newGeneratorSpec(s.T(), s.uri, user)
	specJSON, _ := yaml.Marshal(spec)

	// call Generate()
	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)

	require.NoError(s.T(), err)

	_, err = parseStatus(rawStatus.Raw)
	require.NoError(s.T(), err)

	// Create driver with user permissions
	customClient := generatorMockClient{t: s.T(), userPassword: result[testSecretKey]}

	userAuth := &genv1alpha1.Neo4jBasicAuth{
		Username: string(result["user"]),
		Password: esmeta.SecretKeySelector{
			Name: testGeneratedSecretName,
			Key:  testSecretKey,
		},
	}

	driver, err := newDriver(s.ctx, &genv1alpha1.Neo4jAuth{
		URI:   s.uri,
		Basic: userAuth,
	}, customClient, testNamespace)
	require.NoError(s.T(), err)
	err = driver.VerifyConnectivity(s.ctx)
	require.NoError(s.T(), err)

	resultQuery, err := neo4jSDK.ExecuteQuery(
		s.ctx, driver,
		`CREATE (:Test {message: "Hello from new user"})`,
		map[string]any{},
		neo4jSDK.EagerResultTransformer,
		neo4jSDK.ExecuteQueryWithDatabase(spec.Spec.Database),
	)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resultQuery)

	err = driver.Close(s.ctx)
	require.NoError(s.T(), err)

	// Cleanup
	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, rawStatus, customClient, testNamespace)
	require.NoError(s.T(), err)

	// Check if the node was not deleted
	driver, err = newDriver(s.ctx, &genv1alpha1.Neo4jAuth{
		URI:   s.uri,
		Basic: spec.Spec.Auth.Basic,
	}, s.client, testNamespace)
	require.NoError(s.T(), err)
	defer closeDriver(s.T(), s.ctx, driver)
	err = driver.VerifyConnectivity(s.ctx)
	require.NoError(s.T(), err)

	resultQuery, err = neo4jSDK.ExecuteQuery(
		s.ctx, driver,
		`MATCH (n:Test {message: "Hello from new user"}) RETURN n`,
		map[string]any{},
		neo4jSDK.EagerResultTransformer,
		neo4jSDK.ExecuteQueryWithDatabase(spec.Spec.Database),
	)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resultQuery)

	resultMap := resultQuery.Records[0].AsMap()
	require.Contains(s.T(), resultMap, "n")
	assert.NotNil(s.T(), resultMap["n"])
	assert.NotNil(s.T(), resultMap["n"].(neo4jSDK.Node).GetProperties()["message"])
	assert.Equal(s.T(), resultMap["n"].(neo4jSDK.Node).GetProperties()["message"], "Hello from new user")

	// Deleting the node
	_, err = neo4jSDK.ExecuteQuery(
		s.ctx, driver,
		`MATCH (n:Test {message: "Hello from new user"})
		DELETE n`,
		map[string]any{},
		neo4jSDK.EagerResultTransformer,
		neo4jSDK.ExecuteQueryWithDatabase(spec.Spec.Database),
	)
	require.NoError(s.T(), err)
}
