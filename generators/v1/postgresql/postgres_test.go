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

// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/go-logr/logr/testr"

	"github.com/jackc/pgx/v5"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	testUser                = "generated_user"
	testPass                = "strongpassword"
	testNamespace           = "default"
	testSecretName          = "testpass"
	testSecretKey           = "password"
	testGeneratedSecretName = "userpass"
)

type generatorMockClient struct {
	client.Client
	userPassword []byte
	t            *testing.T
}

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

type PostgresTestSuite struct {
	suite.Suite
	ctx    context.Context
	client generatorMockClient
	db     *pgx.Conn
	host   string
	port   nat.Port
	pg     *tcpostgres.PostgresContainer
}

func TestPostgresGeneratorTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresTestSuite))
}

func (s *PostgresTestSuite) SetupSuite() {
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx

	s.client = generatorMockClient{t: s.T()}

	s.host = "localhost"
	pgContainer, err := tcpostgres.Run(s.ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword(testPass),
		// testcontainers.WithExposedPorts("5432/tcp"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(s.T(), err)
	s.pg = pgContainer
	s.port, err = pgContainer.MappedPort(s.ctx, "5432/tcp")
	require.NoError(s.T(), err)

	connStr, err := pgContainer.ConnectionString(s.ctx, "sslmode=disable")
	require.NoError(s.T(), err)

	conn, err := pgx.Connect(s.ctx, connStr)
	require.NoError(s.T(), err)
	require.NoError(s.T(), conn.Ping(s.ctx))

	s.db = conn

	cl := fake.NewClientBuilder().Build()
	log := testr.NewWithOptions(s.T(), testr.Options{Verbosity: 1})
	sched := scheduler.New(cl, log)
	scheduler.SetGlobal(sched)
	go func() {
		if err := sched.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.T().Errorf("scheduler.Start: %v", err)
		}
	}()

	s.T().Cleanup(func() {
		conn.Close(s.ctx)
		if err := testcontainers.TerminateContainer(pgContainer); err != nil {
			s.T().Logf("failed to terminate container: %s", err)
		}
		cancel()
	})
}

func newGeneratorSpec(t *testing.T, host, port, username string, destructive bool, reassignTo *string) *genv1alpha1.PostgreSQL {
	t.Helper()

	return &genv1alpha1.PostgreSQL{
		Spec: genv1alpha1.PostgreSQLSpec{
			Host:     host,
			Port:     port,
			Database: "postgres",
			Auth: genv1alpha1.PostgreSQLAuth{
				Username: "postgres",
				Password: esmeta.SecretKeySelector{
					Name: testSecretName,
					Key:  testSecretKey,
				},
			},
			User: &genv1alpha1.PostgreSQLUser{
				Username: username,
				Attributes: []genv1alpha1.PostgreSQLUserAttribute{
					{Name: "CREATEDB"},
				},
				Roles:              []string{"pg_read_all_data", "customrole"},
				DestructiveCleanup: destructive,
				ReassignTo:         reassignTo,
			},
		},
	}
}

func (s *PostgresTestSuite) verifyAttributes(rolname string,
	canLogin, createDb, createRole, superuser bool,
	replication, inherit, byPassRls bool,
	connLimit int,
) {
	var (
		rolcanlogin    bool
		rolcreatedb    bool
		rolcreaterole  bool
		rolsuper       bool
		rolreplication bool
		rolinherit     bool
		rolconnlimit   int
		rolpassword    string
		rolbypassrls   bool
	)
	row := s.db.QueryRow(s.ctx, `
		SELECT rolcanlogin, rolcreatedb, rolcreaterole, rolsuper, rolreplication, 
		rolinherit, rolconnlimit, rolpassword, rolbypassrls
		FROM pg_roles
		WHERE rolname = $1
	`, rolname)
	err := row.Scan(
		&rolcanlogin, &rolcreatedb, &rolcreaterole, &rolsuper, &rolreplication,
		&rolinherit, &rolconnlimit, &rolpassword, &rolbypassrls,
	)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), canLogin, rolcanlogin, "user should be able to login")
	assert.Equal(s.T(), createDb, rolcreatedb, "user should have CREATEDB")
	assert.Equal(s.T(), createRole, rolcreaterole, "user should not have CREATEROLE")
	assert.Equal(s.T(), superuser, rolsuper, "user should not be SUPERUSER")
	assert.Equal(s.T(), replication, rolreplication, "user should not have REPLICATION")
	assert.Equal(s.T(), inherit, rolinherit, "user should have INHERIT")
	assert.Equal(s.T(), byPassRls, rolbypassrls, "user should not have BYPASSRLS")
	assert.Equal(s.T(), connLimit, rolconnlimit, "user should have no connection limit")
	assert.NotNil(s.T(), rolpassword, "user should have PASSWORD")
}

func (s *PostgresTestSuite) verifyGrantedRoles(rolname string, expectedRoles []string) {
	rows, err := s.db.Query(s.ctx, `
		SELECT r.rolname
		FROM pg_auth_members m
		JOIN pg_roles r ON r.oid = m.roleid
		JOIN pg_roles u ON u.oid = m.member
		WHERE u.rolname = $1
	`, rolname)
	require.NoError(s.T(), err)

	defer rows.Close()
	var grantedRoles []string
	for rows.Next() {
		var role string
		require.NoError(s.T(), rows.Scan(&role))
		grantedRoles = append(grantedRoles, role)
	}

	require.NoError(s.T(), rows.Err())
	for _, expectedRole := range expectedRoles {
		assert.Contains(s.T(), grantedRoles, expectedRole, "user should have role %s", expectedRole)
	}
}

func (s *PostgresTestSuite) TestGenerateAndCleanupUser() {
	username := fmt.Sprintf("%s_TestGenerate", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), username, true, nil)

	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}

	// Call Generate
	result, statusRaw, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")
	regex := regexp.MustCompile(fmt.Sprintf(`%s_[a-zA-Z0-9]{8}`, username))
	assert.Regexp(s.T(), regex, string(result["username"]))

	generatedUsername := string(result["username"])
	// Verify attributes
	s.verifyAttributes(generatedUsername, true, true, false, false, false, true, false, -1)

	// Verify granted roles
	s.verifyGrantedRoles(generatedUsername, []string{"pg_read_all_data", "customrole"})

	// Cleanup
	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, statusRaw, s.client, testNamespace)
	require.NoError(s.T(), err)

	// Verify user was dropped
	row := s.db.QueryRow(s.ctx, `SELECT 1 FROM pg_roles WHERE rolname = $1`, generatedUsername)
	var dummy int
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)
}

func (s *PostgresTestSuite) TestGenerateWithIdleCleanup() {
	username := fmt.Sprintf("%s_TestGenerate", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), username, true, nil)
	spec.Spec.CleanupPolicy = &genv1alpha1.PostgreSQLCleanupPolicy{
		ActivityTrackingInterval: metav1.Duration{Duration: time.Second * 2},
		CleanupPolicy: genv1alpha1.CleanupPolicy{
			Type:        genv1alpha1.IdleCleanupPolicy,
			IdleTimeout: metav1.Duration{Duration: time.Second * 10},
			GracePeriod: metav1.Duration{Duration: time.Second * 10},
		},
	}

	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}

	// Ensure table and function are not created previously
	_, err = s.db.Exec(s.ctx, `DROP TABLE IF EXISTS session_observation;`)
	require.NoError(s.T(), err)
	_, err = s.db.Exec(s.ctx, `DROP FUNCTION IF EXISTS snapshot_pg_stat_activity;`)
	require.NoError(s.T(), err)

	// Call Generate
	result, _, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")
	regex := regexp.MustCompile(fmt.Sprintf(`%s_[a-zA-Z0-9]{8}`, username))
	assert.Regexp(s.T(), regex, string(result["username"]))

	row := s.db.QueryRow(s.ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public'
    AND table_name   = 'session_observation'
) AS table_exists;
`)

	var tableExists bool
	err = row.Scan(&tableExists)
	require.NoError(s.T(), err)
	assert.True(s.T(), tableExists, "session_observation table should exist")

	row = s.db.QueryRow(s.ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.routines
  WHERE specific_schema = 'public'
    AND routine_name    = 'snapshot_pg_stat_activity'
    AND routine_type    = 'FUNCTION'
) AS function_exists;
`)

	var functionExists bool
	err = row.Scan(&functionExists)
	require.NoError(s.T(), err)
	assert.True(s.T(), functionExists, "snapshot_pg_stat_activity function should exist")
}

func (s *PostgresTestSuite) TestGenerateUserWithSameUsername() {
	username := fmt.Sprintf("%s_TestGenerateUserWithSameUsername", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), username, true, nil)
	spec.Spec.User.SuffixSize = ptr.To(0)
	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}

	// Call Generate
	result, _, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")
	assert.Equal(s.T(), username, string(result["username"]))

	generatedUsername := string(result["username"])
	s.verifyAttributes(generatedUsername, true, true, false, false, false, true, false, -1)
	s.verifyGrantedRoles(generatedUsername, []string{"pg_read_all_data", "customrole"})

	// Call Generate again with new attributes
	spec.Spec.User.SuffixSize = ptr.To(0)
	spec.Spec.User.Attributes = []genv1alpha1.PostgreSQLUserAttribute{
		{Name: "NOINHERIT"},
		{Name: "CONNECTION_LIMIT", Value: ptr.To("5")},
	}
	spec.Spec.User.Roles = []string{"pg_write_all_data"}

	specJSON, err = yaml.Marshal(spec)
	require.NoError(s.T(), err)
	result, statusRaw, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")
	assert.Equal(s.T(), username, string(result["username"]))

	generatedUsername = string(result["username"])
	s.verifyAttributes(generatedUsername, true, false, false, false, false, false, false, 5)
	s.verifyGrantedRoles(generatedUsername, []string{"pg_write_all_data"})

	// Cleanup
	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, statusRaw, s.client, testNamespace)
	require.NoError(s.T(), err)

	// Verify user was dropped
	row := s.db.QueryRow(s.ctx, `SELECT 1 FROM pg_roles WHERE rolname = $1`, generatedUsername)
	var dummy int
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)
}

func (s *PostgresTestSuite) TestGenerateUserWithoutSuffix() {
	username := fmt.Sprintf("%s_TestGenerateWithoutSuffix", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), username, true, nil)
	spec.Spec.User.SuffixSize = ptr.To(0)

	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}

	result, statusRaw, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")
	assert.Equal(s.T(), username, string(result["username"]))

	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, statusRaw, s.client, testNamespace)
	require.NoError(s.T(), err)
}

func (s *PostgresTestSuite) TestNonDestructiveCleanup() {
	username := fmt.Sprintf("%s_NonDestructive", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), username, false, nil)
	spec.Spec.User.Attributes = []genv1alpha1.PostgreSQLUserAttribute{{Name: "SUPERUSER"}}
	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")

	generatedUsername := string(result["username"])
	password := string(result["password"])

	userSpec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), generatedUsername, false, nil)
	userSpec.Spec.Auth = genv1alpha1.PostgreSQLAuth{
		Username: generatedUsername,
		Password: esmeta.SecretKeySelector{
			Name: testGeneratedSecretName,
			Key:  testSecretKey,
		},
	}
	userClient := generatorMockClient{t: s.T(), userPassword: []byte(password)}
	userDB, err := newConnection(s.ctx, &userSpec.Spec, userClient, testNamespace)
	require.NoError(s.T(), err)
	defer userDB.Close(s.ctx)

	_, err = userDB.Exec(s.ctx, `CREATE TABLE cleanup_test (id INT)`)
	require.NoError(s.T(), err)

	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, rawStatus, s.client, testNamespace)
	require.NoError(s.T(), err)

	row := s.db.QueryRow(s.ctx, `SELECT 1 FROM pg_roles WHERE rolname = $1`, generatedUsername)
	var dummy int
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)

	row = s.db.QueryRow(s.ctx, `
		SELECT tableowner
		FROM pg_tables
		WHERE tablename = 'cleanup_test'
	`)
	var owner string
	err = row.Scan(&owner)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "postgres", owner)

	_, err = s.db.Exec(s.ctx, `DROP TABLE IF EXISTS cleanup_test`)
	require.NoError(s.T(), err)
}

func (s *PostgresTestSuite) TestNonDestructiveCleanupWithExistentReassignUser() {
	username := fmt.Sprintf("%s_NonDestructiveWithExistentReassignUser", testUser)

	// Generate reassign user as `username` without suffix
	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), username, false, nil)
	spec.Spec.User.Attributes = []genv1alpha1.PostgreSQLUserAttribute{{Name: "SUPERUSER"}}
	spec.Spec.User.SuffixSize = ptr.To(0)
	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	reassignGen := &Generator{}
	_, _, err = reassignGen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)

	spec = newGeneratorSpec(s.T(), "localhost", s.port.Port(), username, false, &username)
	spec.Spec.User.Attributes = []genv1alpha1.PostgreSQLUserAttribute{{Name: "SUPERUSER"}}
	specJSON, err = yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")

	generatedUsername := string(result["username"])
	password := string(result["password"])

	userSpec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), generatedUsername, false, nil)
	userSpec.Spec.Auth = genv1alpha1.PostgreSQLAuth{
		Username: generatedUsername,
		Password: esmeta.SecretKeySelector{
			Name: testGeneratedSecretName,
			Key:  testSecretKey,
		},
	}
	userClient := generatorMockClient{t: s.T(), userPassword: []byte(password)}
	userDB, err := newConnection(s.ctx, &userSpec.Spec, userClient, testNamespace)
	require.NoError(s.T(), err)
	defer userDB.Close(s.ctx)

	_, err = userDB.Exec(s.ctx, `CREATE TABLE cleanup_test (id INT)`)
	require.NoError(s.T(), err)

	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, rawStatus, s.client, testNamespace)
	require.NoError(s.T(), err)

	row := s.db.QueryRow(s.ctx, `SELECT 1 FROM pg_roles WHERE rolname = $1`, generatedUsername)
	var dummy int
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)

	row = s.db.QueryRow(s.ctx, `
		SELECT tableowner
		FROM pg_tables
		WHERE tablename = 'cleanup_test'
	`)
	var owner string
	err = row.Scan(&owner)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), username, owner)

	_, err = s.db.Exec(s.ctx, `DROP TABLE IF EXISTS cleanup_test`)
	require.NoError(s.T(), err)
}

func (s *PostgresTestSuite) TestNonDestructiveCleanupWithNonExistentReassignUser() {
	username := fmt.Sprintf("%s_NonDestructiveWithNonExistentReassignUser", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), username, false, &username)
	spec.Spec.User.Attributes = []genv1alpha1.PostgreSQLUserAttribute{{Name: "SUPERUSER"}}
	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")

	generatedUsername := string(result["username"])
	password := string(result["password"])

	userSpec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), generatedUsername, false, nil)
	userSpec.Spec.Auth = genv1alpha1.PostgreSQLAuth{
		Username: generatedUsername,
		Password: esmeta.SecretKeySelector{
			Name: testGeneratedSecretName,
			Key:  testSecretKey,
		},
	}
	userClient := generatorMockClient{t: s.T(), userPassword: []byte(password)}
	userDB, err := newConnection(s.ctx, &userSpec.Spec, userClient, testNamespace)
	require.NoError(s.T(), err)
	defer userDB.Close(s.ctx)

	_, err = userDB.Exec(s.ctx, `CREATE TABLE cleanup_test (id INT)`)
	require.NoError(s.T(), err)

	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, rawStatus, s.client, testNamespace)
	require.NoError(s.T(), err)

	row := s.db.QueryRow(s.ctx, `SELECT 1 FROM pg_roles WHERE rolname = $1`, generatedUsername)
	var dummy int
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)

	row = s.db.QueryRow(s.ctx, `
		SELECT tableowner
		FROM pg_tables
		WHERE tablename = 'cleanup_test'
	`)
	var owner string
	err = row.Scan(&owner)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), username, owner)

	_, err = s.db.Exec(s.ctx, `DROP TABLE IF EXISTS cleanup_test`)
	require.NoError(s.T(), err)
}

func (s *PostgresTestSuite) TestDestructiveCleanup() {
	username := fmt.Sprintf("%s_Destructive", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), username, true, nil)
	spec.Spec.User.Attributes = []genv1alpha1.PostgreSQLUserAttribute{{Name: "SUPERUSER"}}
	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")

	generatedUsername := string(result["username"])
	password := string(result["password"])

	userSpec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), generatedUsername, true, nil)
	userSpec.Spec.Auth = genv1alpha1.PostgreSQLAuth{
		Username: generatedUsername,
		Password: esmeta.SecretKeySelector{
			Name: testGeneratedSecretName,
			Key:  testSecretKey,
		},
	}
	userClient := generatorMockClient{t: s.T(), userPassword: []byte(password)}
	userDB, err := newConnection(s.ctx, &userSpec.Spec, userClient, testNamespace)
	require.NoError(s.T(), err)
	defer userDB.Close(s.ctx)

	_, err = userDB.Exec(s.ctx, `CREATE TABLE cleanup_test (id INT)`)
	require.NoError(s.T(), err)

	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, rawStatus, s.client, testNamespace)
	require.NoError(s.T(), err)

	row := s.db.QueryRow(s.ctx, `SELECT 1 FROM pg_roles WHERE rolname = $1`, generatedUsername)
	var dummy int
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)

	row = s.db.QueryRow(s.ctx, `
		SELECT 1
		FROM pg_tables
		WHERE tablename = 'cleanup_test'
	`)
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)
}
