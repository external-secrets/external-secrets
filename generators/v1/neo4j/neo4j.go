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

// Package neo4j implements Neo4j user generator.
package neo4j

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/labstack/gommon/log"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/generators/v1/password"
	utils "github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// Generator implements Neo4j user generation.
type Generator struct{}

const (
	defaultDatabase   = "neo4j"
	defaultProvider   = genv1alpha1.Neo4jAuthProviderNative
	defaultSuffixSize = 8
)

// Generate generates Neo4j user credentials.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, err
	}

	if strings.Contains(res.Spec.User.User, "-") {
		return nil, nil, fmt.Errorf("invalid username %q: must not contain dashes (-)", res.Spec.User.User)
	}

	driver, err := newDriver(ctx, &res.Spec.Auth, kube, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create driver: %w", err)
	}
	defer func() {
		err := driver.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close driver: %v", err)
		}
	}()

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to verify connectivity: %w", err)
	}

	if res.Spec.Database == "" {
		res.Spec.Database = defaultDatabase
	}

	if res.Spec.User.Provider == "" {
		res.Spec.User.Provider = defaultProvider
	}

	user, err := createOrReplaceUser(ctx, driver, &res.Spec)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create or replace user: %w", err)
	}

	if res.Spec.Enterprise {
		err = addRolesToUser(ctx, driver, &res.Spec)
		if err != nil {
			dropErr := dropUser(ctx, driver, res.Spec.User.User)
			if dropErr != nil {
				return nil, nil, fmt.Errorf("unable to drop user: %w", dropErr)
			}
			return nil, nil, fmt.Errorf("unable to add roles to user: %w", err)
		}
	}

	username, ok := user["user"]
	if !ok {
		return nil, nil, fmt.Errorf("user not found in response")
	}

	rawState, err := json.Marshal(&genv1alpha1.Neo4jUserState{
		User: string(username),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal state: %w", err)
	}

	return user, &apiextensions.JSON{Raw: rawState}, nil
}

// Cleanup cleans up generated Neo4j users.
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
	driver, err := newDriver(ctx, &res.Spec.Auth, kclient, namespace)
	if err != nil {
		return err
	}
	defer func() {
		err := driver.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close driver: %v", err)
		}
	}()

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		return fmt.Errorf("unable to verify connectivity: %w", err)
	}

	if res.Spec.Enterprise {
		err = suspendUser(ctx, driver, status.User)
		if err != nil {
			return fmt.Errorf("unable to suspend user: %w", err)
		}
	} else {
		err = dropUser(ctx, driver, status.User)
		if err != nil {
			return fmt.Errorf("unable to drop user: %w", err)
		}
	}

	return nil
}

// GetCleanupPolicy returns the cleanup policy for this generator.
func (g *Generator) GetCleanupPolicy(_ *apiextensions.JSON) (*genv1alpha1.CleanupPolicy, error) {
	return nil, nil
}

// LastActivityTime returns the last activity time for generated resources.
func (g *Generator) LastActivityTime(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

// GetKeys returns the keys generated by this generator.
func (g *Generator) GetKeys() map[string]string {
	return map[string]string{
		"user":     "Neo4j database username",
		"password": "Password for the Neo4j user",
	}
}

// EscapeNeo4jIdentifier escapes Neo4j identifiers for safe use in queries.
func EscapeNeo4jIdentifier(input string) (string, error) {
	if input == "" {
		return "", errors.New("identifier cannot be empty")
	}

	var sanitized strings.Builder
	for _, r := range input {
		if unicode.IsControl(r) {
			return "", errors.New("identifier contains control characters")
		}
		switch r {
		case '`':
			sanitized.WriteString("``") // escape backtick
		case '\'', '"':
			// skip quotes
		default:
			sanitized.WriteRune(r)
		}
	}

	return "`" + sanitized.String() + "`", nil
}

func newDriver(ctx context.Context, auth *genv1alpha1.Neo4jAuth, kclient client.Client, ns string) (neo4j.DriverWithContext, error) {
	dbURI := auth.URI
	var authToken neo4j.AuthToken
	if auth.Bearer != nil {
		bearerToken, err := resolvers.SecretKeyRef(ctx, kclient, resolvers.EmptyStoreKind, ns, &esmeta.SecretKeySelector{
			Namespace: &ns,
			Name:      auth.Bearer.Token.Name,
			Key:       auth.Bearer.Token.Key,
		})
		if err != nil {
			return nil, err
		}
		authToken = neo4j.BearerAuth(bearerToken)
	} else if auth.Basic != nil {
		dbUser := auth.Basic.Username
		dbPassword, err := resolvers.SecretKeyRef(ctx, kclient, resolvers.EmptyStoreKind, ns, &esmeta.SecretKeySelector{
			Namespace: &ns,
			Name:      auth.Basic.Password.Name,
			Key:       auth.Basic.Password.Key,
		})
		if err != nil {
			return nil, err
		}
		authToken = neo4j.BasicAuth(dbUser, dbPassword, "")
	}

	return neo4j.NewDriverWithContext(
		dbURI,
		authToken,
	)
}

func createOrReplaceUser(ctx context.Context, driver neo4j.DriverWithContext, spec *genv1alpha1.Neo4jSpec) (map[string][]byte, error) {
	var query strings.Builder
	username := spec.User.User
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
	sanitizedUsername, err := EscapeNeo4jIdentifier(username)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize username %q: %w", username, err)
	}
	query.WriteString(fmt.Sprintf("CREATE OR REPLACE USER %s\n", sanitizedUsername))

	authProvider := spec.User.Provider
	if spec.Enterprise {
		if spec.User.Home != nil {
			sanitizedHome, err := EscapeNeo4jIdentifier(*spec.User.Home)
			if err != nil {
				return nil, fmt.Errorf("failed to sanitize user home %q: %w", *spec.User.Home, err)
			}
			query.WriteString(fmt.Sprintf("SET HOME DATABASE %s\n", sanitizedHome))
		}
		authProvider = spec.User.Provider
	}

	query.WriteString(fmt.Sprintf("SET AUTH '%s' {\n", authProvider))

	if authProvider == genv1alpha1.Neo4jAuthProviderNative {
		pass, err := generatePassword(genv1alpha1.Password{
			Spec: genv1alpha1.PasswordSpec{
				SymbolCharacters: ptr.To("~!@#$%^&*()_+-={}|[]:<>?,./"),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate password: %w", err)
		}

		query.WriteString(fmt.Sprintf("\tSET PASSWORD '%s'\n", string(pass)))
		query.WriteString("\tSET PASSWORD CHANGE NOT REQUIRED\n")
		query.WriteString("}\n")

		_, err = neo4j.ExecuteQuery(ctx, driver,
			query.String(), map[string]any{},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(spec.Database),
		)
		if err != nil {
			return nil, err
		}

		return map[string][]byte{
			"user":     []byte(username),
			"password": pass,
		}, nil
	}
	return nil, fmt.Errorf("unsupported auth provider: %s", spec.User.Provider)
}

func addRolesToUser(ctx context.Context, driver neo4j.DriverWithContext, spec *genv1alpha1.Neo4jSpec) error {
	if len(spec.User.Roles) == 0 {
		return nil
	}

	existingRoles := make([]string, 0)
	result, err := neo4j.ExecuteQuery(ctx, driver,
		"SHOW ROLES", map[string]any{},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(spec.Database),
	)
	if err != nil {
		return err
	}

	for _, record := range result.Records {
		roleName, ok := record.AsMap()["role"].(string)
		if !ok {
			log.Errorf("failed to get role name from record %v", record)
			continue
		}
		existingRoles = append(existingRoles, roleName)
	}

	sanitizedRoles := make([]string, 0, len(spec.User.Roles))
	for _, role := range spec.User.Roles {
		if !slices.Contains(existingRoles, role) {
			err = createBasicRole(ctx, driver, spec.Database, role)
			if err != nil {
				return fmt.Errorf("failed to create role %s: %w", role, err)
			}
		}
		sanitizedRole, err := EscapeNeo4jIdentifier(role)
		if err != nil {
			return fmt.Errorf("failed to sanitize role %q: %w", role, err)
		}

		sanitizedRoles = append(sanitizedRoles, sanitizedRole)
	}

	sanitizedUsername, err := EscapeNeo4jIdentifier(spec.User.User)
	if err != nil {
		return fmt.Errorf("failed to sanitize username %q: %w", spec.User.User, err)
	}
	query := fmt.Sprintf("GRANT ROLE %s TO %s", strings.Join(sanitizedRoles, ", "), sanitizedUsername)
	_, err = neo4j.ExecuteQuery(ctx, driver,
		query, map[string]any{},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(spec.Database),
	)
	if err != nil {
		return err
	}
	return nil
}

func dropUser(ctx context.Context, driver neo4j.DriverWithContext, username string) error {
	sanitizedUsername, err := EscapeNeo4jIdentifier(username)
	if err != nil {
		return fmt.Errorf("failed to sanitize username %q: %w", username, err)
	}

	query := fmt.Sprintf("DROP USER %s IF EXISTS", sanitizedUsername)
	_, err = neo4j.ExecuteQuery(ctx, driver, query, nil, neo4j.EagerResultTransformer)
	if err != nil {
		return err
	}
	return nil
}

func suspendUser(ctx context.Context, driver neo4j.DriverWithContext, username string) error {
	sanitizedUsername, err := EscapeNeo4jIdentifier(username)
	if err != nil {
		return fmt.Errorf("failed to sanitize username %q: %w", username, err)
	}
	query := fmt.Sprintf(
		`ALTER USER %s IF EXISTS 
		SET STATUS SUSPENDED`,
		sanitizedUsername,
	)
	_, err = neo4j.ExecuteQuery(ctx, driver, query, nil, neo4j.EagerResultTransformer)
	if err != nil {
		return err
	}
	return nil
}

func createBasicRole(ctx context.Context, driver neo4j.DriverWithContext, dbName, roleName string) error {
	sanitizedRole, err := EscapeNeo4jIdentifier(roleName)
	if err != nil {
		return fmt.Errorf("failed to sanitize role %q: %w", roleName, err)
	}

	query := fmt.Sprintf("CREATE ROLE %s IF NOT EXISTS AS COPY OF PUBLIC", sanitizedRole)
	_, err = neo4j.ExecuteQuery(ctx, driver,
		query, map[string]any{},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(dbName),
	)
	if err != nil {
		return err
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

func parseSpec(data []byte) (*genv1alpha1.Neo4j, error) {
	var spec genv1alpha1.Neo4j
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func parseStatus(data []byte) (*genv1alpha1.Neo4jUserState, error) {
	var state genv1alpha1.Neo4jUserState
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
	return genv1alpha1.Neo4jKind
}
