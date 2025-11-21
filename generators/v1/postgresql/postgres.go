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

// Package postgresql implements PostgreSQL user generator.
package postgresql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"time"

	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/gommon/log"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/generators/v1/password"
	utils "github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/scheduler"
)

// Generator implements the PostgreSQL user generator.
type Generator struct{}

const (
	defaultPort       = "5432"
	defaultUser       = "postgres"
	defaultDbName     = "postgres"
	defaultSuffixSize = 8
	schedIDFmt        = "psql-session-observation-%s-%s:%s"
)

var mapAttributes = map[string]genv1alpha1.PostgreSQLUserAttributesEnum{
	string(genv1alpha1.PostgreSQLUserSuperUser):   genv1alpha1.PostgreSQLUserSuperUser,
	string(genv1alpha1.PostgreSQLUserCreateDb):    genv1alpha1.PostgreSQLUserCreateDb,
	string(genv1alpha1.PostgreSQLUserCreateRole):  genv1alpha1.PostgreSQLUserCreateRole,
	string(genv1alpha1.PostgreSQLUserReplication): genv1alpha1.PostgreSQLUserReplication,
	string(genv1alpha1.PostgreSQLUserNoInherit):   genv1alpha1.PostgreSQLUserNoInherit,
	string(genv1alpha1.PostgreSQLUserByPassRls):   genv1alpha1.PostgreSQLUserByPassRls,
	"CONNECTION_LIMIT":                            genv1alpha1.PostgreSQLUserConnectionLimit,
	string(genv1alpha1.PostgreSQLUserLogin):       genv1alpha1.PostgreSQLUserLogin,
	string(genv1alpha1.PostgreSQLUserPassword):    genv1alpha1.PostgreSQLUserPassword,
}

// Generate creates a new user in the database.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, err
	}

	db, err := newConnection(ctx, &res.Spec, kube, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create db connection: %w", err)
	}
	defer func() {
		err := db.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to ping the database: %w", err)
	}

	cleanupPolicy, err := g.GetCleanupPolicy(jsonSpec)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get cleanup policy: %w", err)
	}
	if cleanupPolicy != nil && cleanupPolicy.Type == genv1alpha1.IdleCleanupPolicy {
		err = setupObservation(ctx, db)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to setup observation: %w", err)
		}
		schedID := fmt.Sprintf(schedIDFmt, res.UID, res.Spec.Host, res.Spec.Port)
		scheduler.Global().ScheduleInterval(schedID, res.Spec.CleanupPolicy.ActivityTrackingInterval.Duration, time.Minute, func(ctx context.Context, log logr.Logger) {
			err := triggerSessionSnapshot(ctx, &res.Spec, kube, namespace)
			if err != nil {
				log.Error(err, "failed to trigger session observation")
				return
			}
		})
	}

	user, err := createUser(ctx, db, &res.Spec)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create or update user: %w", err)
	}

	username, ok := user["username"]
	if !ok {
		return nil, nil, fmt.Errorf("user not found in response")
	}

	rawState, err := json.Marshal(&genv1alpha1.PostgreSQLUserState{
		Username: string(username),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal state: %w", err)
	}

	return user, &apiextensions.JSON{Raw: rawState}, nil
}

// Cleanup removes the user from the database.
func (g *Generator) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, previousStatus genv1alpha1.GeneratorProviderState, kclient client.Client, namespace string) error {
	if previousStatus == nil {
		return fmt.Errorf("missing previous status")
	}
	status, err := parseStatus(previousStatus.Raw)
	if err != nil {
		return err
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return err
	}
	db, err := newConnection(ctx, &res.Spec, kclient, namespace)
	if err != nil {
		return err
	}
	defer func() {
		err := db.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping(ctx)
	if err != nil {
		return fmt.Errorf("unable to ping the database: %w", err)
	}

	err = dropUser(ctx, db, status.Username, res.Spec)
	if err != nil {
		return fmt.Errorf("unable to drop user: %w", err)
	}

	return nil
}

// GetCleanupPolicy returns the cleanup policy of the generator.
func (g *Generator) GetCleanupPolicy(obj *apiextensions.JSON) (*genv1alpha1.CleanupPolicy, error) {
	res, err := parseSpec(obj.Raw)
	if err != nil {
		return nil, err
	}
	if res.Spec.CleanupPolicy == nil {
		return nil, nil
	}

	policy := genv1alpha1.CleanupPolicy{
		Type:        res.Spec.CleanupPolicy.Type,
		IdleTimeout: res.Spec.CleanupPolicy.IdleTimeout,
		GracePeriod: res.Spec.CleanupPolicy.GracePeriod,
	}
	return &policy, nil
}

// LastActivityTime returns the last activity time of the user.
func (g *Generator) LastActivityTime(ctx context.Context, obj *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, kube client.Client, namespace string) (time.Time, bool, error) {
	status, err := parseStatus(state.Raw)
	if err != nil {
		return time.Time{}, false, err
	}
	res, err := parseSpec(obj.Raw)
	if err != nil {
		return time.Time{}, false, err
	}
	db, err := newConnection(ctx, &res.Spec, kube, namespace)
	if err != nil {
		return time.Time{}, false, err
	}
	defer func() {
		err := db.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping(ctx)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("unable to ping the database: %w", err)
	}

	lastActivity, err := getUserActivity(ctx, db, status.Username)
	if err != nil {
		return time.Time{}, false, err
	}
	return lastActivity, true, nil
}

// GetKeys returns the keys that are generated by the generator.
func (g *Generator) GetKeys() map[string]string {
	return map[string]string{
		"username": "PostgreSQL database username",
		"password": "PostgreSQL user password",
	}
}

func newConnection(ctx context.Context, spec *genv1alpha1.PostgreSQLSpec, kclient client.Client, ns string) (*pgx.Conn, error) {
	dbName := defaultDbName
	if spec.Database != "" {
		dbName = spec.Database
	}

	port := defaultPort
	if spec.Port != "" {
		port = spec.Port
	}

	username := defaultUser
	if spec.Auth.Username != "" {
		username = spec.Auth.Username
	}
	password, err := resolvers.SecretKeyRef(ctx, kclient, resolvers.EmptyStoreKind, ns, &esmeta.SecretKeySelector{
		Namespace: &ns,
		Name:      spec.Auth.Password.Name,
		Key:       spec.Auth.Password.Key,
	})
	if err != nil {
		return nil, err
	}

	psqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		spec.Host, port, username, password, dbName,
	)

	return pgx.Connect(ctx, psqlInfo)
}

func createSessionObservationTable(ctx context.Context, db *pgx.Conn) error {
	query := `
CREATE TABLE IF NOT EXISTS session_observation (
    pid              INTEGER     PRIMARY KEY,
    usename          TEXT        NOT NULL,
    client_addr      INET,
    application_name TEXT,
    state            TEXT,
    state_change     TIMESTAMPTZ,
    first_seen       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen        TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create session_observation table: %w", err)
	}

	return nil
}

func createSessionSnapshotFunction(ctx context.Context, db *pgx.Conn) error {
	query := `
CREATE OR REPLACE FUNCTION snapshot_pg_stat_activity() RETURNS void AS $$
BEGIN
  INSERT INTO session_observation (pid, usename, client_addr, application_name, state, state_change)
  SELECT
    pid,
    usename,
    client_addr,
    application_name,
    state,
    state_change
  FROM pg_stat_activity
  WHERE pid <> pg_backend_pid()  -- ignore yourself
  AND usename IS NOT NULL
  ON CONFLICT (pid)
  DO UPDATE SET
    state = EXCLUDED.state,
    state_change = EXCLUDED.state_change,
    last_seen = now();
END;
$$ LANGUAGE plpgsql;
`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create session_observation table: %w", err)
	}

	return nil
}

func triggerSessionSnapshot(ctx context.Context, spec *genv1alpha1.PostgreSQLSpec, client client.Client, namespace string) error {
	db, err := newConnection(ctx, spec, client, namespace)
	if err != nil {
		log.Error(err, "failed to create db connection")
		return err
	}
	defer func() {
		err := db.Close(ctx)
		if err != nil {
			log.Error(err, "failed to close db")
		}
	}()

	if _, err := db.Exec(ctx, "SELECT snapshot_pg_stat_activity()"); err != nil {
		return fmt.Errorf("failed to trigger session observation: %w", err)
	}
	return nil
}

func getUserActivity(ctx context.Context, db *pgx.Conn, username string) (time.Time, error) {
	var lastSeen time.Time

	const sqlQuery = `
        SELECT last_seen
        FROM session_observation
        WHERE usename = $1
        ORDER BY last_seen DESC
        LIMIT 1;
    `

	err := db.QueryRow(ctx, sqlQuery, username).Scan(&lastSeen)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Unix(0, 0), nil
		}
		return time.Time{}, fmt.Errorf("failed to get user activity: %w", err)
	}

	return lastSeen, nil
}

func setupObservation(ctx context.Context, db *pgx.Conn) error {
	if err := createSessionObservationTable(ctx, db); err != nil {
		return fmt.Errorf("failed to create session_observation table: %w", err)
	}
	if err := createSessionSnapshotFunction(ctx, db); err != nil {
		return fmt.Errorf("failed to create session_observation function: %w", err)
	}
	return nil
}

func getExistingRoles(ctx context.Context, db *pgx.Conn) ([]string, error) {
	var currentRows = make([]string, 0)
	rows, err := db.Query(ctx, "SELECT rolname FROM pg_roles")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var rolname string
		err = rows.Scan(&rolname)
		if err != nil {
			return nil, err
		}
		currentRows = append(currentRows, rolname)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return currentRows, nil
}

func addRolesAttributesToQueryString(query *strings.Builder, attributes []genv1alpha1.PostgreSQLUserAttribute) {
	if len(attributes) > 0 {
		query.WriteString(" WITH ")
		for i, attr := range attributes {
			if i > 0 {
				query.WriteString(" ")
			}
			if attr.Value != nil {
				if string(mapAttributes[attr.Name]) == string(genv1alpha1.PostgreSQLUserPassword) {
					fmt.Fprintf(query, `%s '%s'`, string(mapAttributes[attr.Name]), *attr.Value)
				} else {
					fmt.Fprintf(query, `%s %s`, string(mapAttributes[attr.Name]), *attr.Value)
				}
			} else {
				query.WriteString(string(mapAttributes[attr.Name]))
			}
		}
	}
}

func createRole(ctx context.Context, db *pgx.Conn, roleName string, attributes []genv1alpha1.PostgreSQLUserAttribute) error {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("CREATE ROLE %s", pgx.Identifier{roleName}.Sanitize()))
	addRolesAttributesToQueryString(&query, attributes)
	_, err := db.Exec(ctx, query.String())
	return err
}

func updateRole(ctx context.Context, db *pgx.Conn, roleName string, attributes []genv1alpha1.PostgreSQLUserAttribute) error {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("ALTER ROLE %s", pgx.Identifier{roleName}.Sanitize()))
	addRolesAttributesToQueryString(&query, attributes)
	_, err := db.Exec(ctx, query.String())
	return err
}

func resetRole(ctx context.Context, db *pgx.Conn, roleName string) error {
	sanitizedRole := pgx.Identifier{roleName}.Sanitize()

	_, err := db.Exec(ctx, fmt.Sprintf(`
		ALTER ROLE %s WITH NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOLOGIN NOREPLICATION NOBYPASSRLS
	`, sanitizedRole))
	if err != nil {
		return fmt.Errorf("failed to reset attributes for role %s: %w", roleName, err)
	}

	rows, err := db.Query(ctx, `
		SELECT r.rolname
		FROM pg_auth_members m
		JOIN pg_roles r ON r.oid = m.roleid
		JOIN pg_roles u ON u.oid = m.member
		WHERE u.rolname = $1
	`, roleName)
	if err != nil {
		return fmt.Errorf("failed to list granted roles for %s: %w", roleName, err)
	}
	defer rows.Close()

	var grantedRoles []string
	for rows.Next() {
		var grantedRole string
		if err := rows.Scan(&grantedRole); err != nil {
			return fmt.Errorf("failed to scan granted role: %w", err)
		}
		grantedRoles = append(grantedRoles, pgx.Identifier{grantedRole}.Sanitize())
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating granted roles: %w", err)
	}

	rolesCSV := strings.Join(grantedRoles, ", ")

	_, err = db.Exec(ctx, fmt.Sprintf("REVOKE %s FROM %s", rolesCSV, sanitizedRole))
	if err != nil {
		return fmt.Errorf("failed to revoke roles [%s] from %s: %w", rolesCSV, roleName, err)
	}

	return nil
}

func createUser(ctx context.Context, db *pgx.Conn, spec *genv1alpha1.PostgreSQLSpec) (map[string][]byte, error) {
	username := spec.User.Username
	suffixSize := defaultSuffixSize
	if spec.User.SuffixSize != nil {
		suffixSize = *spec.User.SuffixSize
	}
	suffix, err := utils.GenerateRandomString(suffixSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random suffix: %w", err)
	}

	if suffix != "" {
		username = fmt.Sprintf("%s_%s", username, suffix)
	}

	currentRoles, err := getExistingRoles(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing roles: %w", err)
	}

	pass, err := generatePassword(genv1alpha1.Password{
		Spec: genv1alpha1.PasswordSpec{
			SymbolCharacters: ptr.To("~!@#$%^&*()_+-={}|[]:<>?,./"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	spec.User.Attributes = append(spec.User.Attributes,
		genv1alpha1.PostgreSQLUserAttribute{
			Name: string(genv1alpha1.PostgreSQLUserLogin),
		}, genv1alpha1.PostgreSQLUserAttribute{
			Name:  string(genv1alpha1.PostgreSQLUserPassword),
			Value: ptr.To(string(pass)),
		},
	)

	if !slices.Contains(currentRoles, username) {
		err = createRole(ctx, db, username, spec.User.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to create role %s: %w", username, err)
		}
	} else {
		err = resetRole(ctx, db, username)
		if err != nil {
			return nil, fmt.Errorf("failed to reset role %s: %w", username, err)
		}
		err = updateRole(ctx, db, username, spec.User.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to create role %s: %w", username, err)
		}
	}

	err = grantRolesToUser(ctx, db, username, spec.User.Roles, currentRoles)
	if err != nil {
		return nil, fmt.Errorf("failed to add roles to user %s: %w", username, err)
	}

	return map[string][]byte{
		"username": []byte(username),
		"password": pass,
	}, nil
}

func grantRolesToUser(ctx context.Context, db *pgx.Conn, username string, roles, currentRoles []string) error {
	sanitizedUsername := pgx.Identifier{username}.Sanitize()

	toGrant := make([]string, 0, len(roles))
	for _, role := range roles {
		if !slices.Contains(currentRoles, role) {
			if err := createRole(ctx, db, role, nil); err != nil {
				return fmt.Errorf("failed to create role %s: %w", role, err)
			}
		}
		toGrant = append(toGrant, pgx.Identifier{role}.Sanitize())
	}

	if len(toGrant) == 0 {
		return nil
	}

	rolesCSV := strings.Join(toGrant, ", ")
	query := fmt.Sprintf("GRANT %s TO %s", rolesCSV, sanitizedUsername)

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to grant roles [%s] to user %s: %w", rolesCSV, username, err)
	}

	return nil
}

func dropUser(ctx context.Context, db *pgx.Conn, username string, spec genv1alpha1.PostgreSQLSpec) error {
	sanitizedUsername := pgx.Identifier{username}.Sanitize()
	if !spec.User.DestructiveCleanup {
		reassignToUser := spec.Auth.Username
		if spec.User.ReassignTo != nil && *spec.User.ReassignTo != "" {
			reassignToUser = *spec.User.ReassignTo
		}

		currentRoles, err := getExistingRoles(ctx, db)
		if err != nil {
			return fmt.Errorf("failed to get existing roles: %w", err)
		}
		if !slices.Contains(currentRoles, reassignToUser) {
			err = createRole(ctx, db, reassignToUser, nil)
			if err != nil {
				return fmt.Errorf("failed to create role %s: %w", reassignToUser, err)
			}
		}

		_, err = db.Exec(ctx, fmt.Sprintf(`REASSIGN OWNED BY %s TO %s`, sanitizedUsername, pgx.Identifier{reassignToUser}.Sanitize()))
		if err != nil {
			return fmt.Errorf("failed to reassign owned by %s to %s: %w", username, reassignToUser, err)
		}
	}
	dropQueries := []string{
		`DROP OWNED BY %s`,
		`DROP ROLE %s`,
	}
	for _, query := range dropQueries {
		_, err := db.Exec(ctx, fmt.Sprintf(query, sanitizedUsername))
		if err != nil {
			return err
		}
	}
	return nil
}

func generatePassword(
	passSpec genv1alpha1.Password,
) ([]byte, error) {
	gen := password.Generator{}
	rawPassSpec, err := yaml.Marshal(passSpec)
	if err != nil {
		return nil, err
	}
	passMap, _, err := gen.Generate(context.TODO(), &apiextensions.JSON{Raw: rawPassSpec}, nil, "")

	if err != nil {
		return nil, err
	}

	pass, ok := passMap["password"]
	if !ok {
		return nil, fmt.Errorf("password not found in generated map")
	}
	return pass, nil
}

func parseSpec(data []byte) (*genv1alpha1.PostgreSQL, error) {
	var spec genv1alpha1.PostgreSQL
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func parseStatus(data []byte) (*genv1alpha1.PostgreSQLUserState, error) {
	var state genv1alpha1.PostgreSQLUserState
	err := json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return genv1alpha1.PostgreSQLKind
}
