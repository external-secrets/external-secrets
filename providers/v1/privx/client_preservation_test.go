/*
Copyright © 2026 SSH Communications

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

package privx

// =============================================================================
// Preservation Property Tests: Authorized Access Unchanged
// These tests verify that authorized access paths work correctly on UNFIXED code.
// They must PASS on both unfixed and fixed code — confirming no regressions.
//
// **Validates: Requirements 3.1, 3.2, 3.3, 3.4**
// =============================================================================

import (
	"context"
	"math/rand"
	"regexp"
	"testing"
	"testing/quick"

	"github.com/SSHcom/privx-sdk-go/v2/api/filters"
	"github.com/SSHcom/privx-sdk-go/v2/api/response"
	"github.com/SSHcom/privx-sdk-go/v2/api/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// =============================================================================
// Generators for property-based testing
// =============================================================================

// validNamespaceChars contains characters valid in Kubernetes namespace names.
const validNamespaceChars = "abcdefghijklmnopqrstuvwxyz0123456789-"

// generateNamespace generates a random valid Kubernetes namespace name.
func generateNamespace(rng *rand.Rand) string {
	// Namespace names: 1-63 chars, [a-z0-9-], must start/end with alphanumeric
	const alphanumeric = "abcdefghijklmnopqrstuvwxyz0123456789"
	length := rng.Intn(20) + 1 // 1-20 chars

	name := make([]byte, length)
	// First char must be alphanumeric
	name[0] = alphanumeric[rng.Intn(len(alphanumeric))]
	// Middle chars can include hyphens
	for i := 1; i < length-1; i++ {
		name[i] = validNamespaceChars[rng.Intn(len(validNamespaceChars))]
	}
	// Last char must be alphanumeric (if length > 1)
	if length > 1 {
		name[length-1] = alphanumeric[rng.Intn(len(alphanumeric))]
	}
	return string(name)
}

// generateNamespaceList generates a random list of namespace names (1-5 items).
func generateNamespaceList(rng *rand.Rand, mustInclude string) []string {
	count := rng.Intn(4) + 1 // 1-4 additional namespaces
	namespaces := make([]string, 0, count+1)
	namespaces = append(namespaces, mustInclude)
	for i := 0; i < count; i++ {
		namespaces = append(namespaces, generateNamespace(rng))
	}
	return namespaces
}

// generateMatchingRegex generates a regex pattern that matches the given namespace.
func generateMatchingRegex(rng *rand.Rand, namespace string) string {
	// Use strategies that are guaranteed to match the namespace
	strategies := []func() string{
		// Strategy 1: exact match with anchors
		func() string { return "^" + regexp.QuoteMeta(namespace) + "$" },
		// Strategy 2: prefix match
		func() string {
			if len(namespace) > 1 {
				prefixLen := rng.Intn(len(namespace)-1) + 1
				return "^" + regexp.QuoteMeta(namespace[:prefixLen])
			}
			return "^" + regexp.QuoteMeta(namespace)
		},
		// Strategy 3: match all
		func() string { return ".*" },
		// Strategy 4: suffix match
		func() string {
			if len(namespace) > 1 {
				suffixLen := rng.Intn(len(namespace)-1) + 1
				return regexp.QuoteMeta(namespace[len(namespace)-suffixLen:]) + "$"
			}
			return regexp.QuoteMeta(namespace) + "$"
		},
	}

	return strategies[rng.Intn(len(strategies))]()
}

// =============================================================================
// Property Test: SecretStore (namespace-scoped) GetSecret always succeeds
// Validates: Requirement 3.1
// =============================================================================

// TestPreservation_SecretStore_GetSecret_Property verifies that GetSecret via a
// namespace-scoped SecretStore always succeeds without namespace validation errors,
// regardless of the namespace name.
func TestPreservation_SecretStore_GetSecret_Property(t *testing.T) {
	// **Validates: Requirements 3.1, 3.2**
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(42)),
	}

	err := quick.Check(func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		namespace := generateNamespace(rng)

		// Namespace-scoped SecretStore — always authorized
		store := &esv1.SecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind: "SecretStore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "privx-store",
				Namespace: namespace,
			},
			Spec: esv1.SecretStoreSpec{},
		}

		c := &SecretsClient{
			vault: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]any{"key": "value-for-" + namespace}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			store:     store,
			namespace: namespace,
		}

		got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key:      "test-secret",
			Property: "key",
		})

		// Must succeed — namespace-scoped stores never have namespace validation
		if err != nil {
			t.Logf("FAIL: namespace=%q, err=%v", namespace, err)
			return false
		}
		if string(got) != "value-for-"+namespace {
			t.Logf("FAIL: namespace=%q, got=%q, want=%q", namespace, string(got), "value-for-"+namespace)
			return false
		}
		return true
	}, config)

	require.NoError(t, err, "Property violated: SecretStore GetSecret should always succeed for namespace-scoped stores")
}

// =============================================================================
// Property Test: ClusterSecretStore with empty conditions GetSecret succeeds
// Validates: Requirement 3.2
// =============================================================================

// TestPreservation_ClusterSecretStore_EmptyConditions_GetSecret_Property verifies
// that GetSecret via a ClusterSecretStore with empty conditions (no restrictions)
// always succeeds, regardless of the requesting namespace.
func TestPreservation_ClusterSecretStore_EmptyConditions_GetSecret_Property(t *testing.T) {
	// **Validates: Requirements 3.2**
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(43)),
	}

	err := quick.Check(func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		namespace := generateNamespace(rng)

		// ClusterSecretStore with NO conditions — all namespaces allowed
		store := &esv1.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterSecretStore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "privx-cluster-store",
			},
			Spec: esv1.SecretStoreSpec{
				Conditions: []esv1.ClusterSecretStoreCondition{},
			},
		}

		c := &SecretsClient{
			vault: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]any{"secret": "data-" + namespace}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			store:     store,
			namespace: namespace,
		}

		got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key:      "test-secret",
			Property: "secret",
		})

		// Must succeed — empty conditions means no restrictions
		if err != nil {
			t.Logf("FAIL: namespace=%q, err=%v", namespace, err)
			return false
		}
		if string(got) != "data-"+namespace {
			t.Logf("FAIL: namespace=%q, got=%q, want=%q", namespace, string(got), "data-"+namespace)
			return false
		}
		return true
	}, config)

	require.NoError(t, err, "Property violated: ClusterSecretStore with empty conditions should allow all namespaces")
}

// =============================================================================
// Property Test: ClusterSecretStore with matching namespace GetSecret succeeds
// Validates: Requirement 3.2
// =============================================================================

// TestPreservation_ClusterSecretStore_MatchingNamespace_GetSecret_Property verifies
// that GetSecret via a ClusterSecretStore where the requesting namespace IS in the
// allowed list always succeeds.
func TestPreservation_ClusterSecretStore_MatchingNamespace_GetSecret_Property(t *testing.T) {
	// **Validates: Requirements 3.2**
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(44)),
	}

	err := quick.Check(func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		namespace := generateNamespace(rng)

		// ClusterSecretStore with conditions that INCLUDE the requesting namespace
		store := &esv1.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterSecretStore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "privx-cluster-store-permitted",
			},
			Spec: esv1.SecretStoreSpec{
				Conditions: []esv1.ClusterSecretStoreCondition{
					{
						Namespaces: generateNamespaceList(rng, namespace),
					},
				},
			},
		}

		c := &SecretsClient{
			vault: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]any{"permitted": "yes-" + namespace}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			store:     store,
			namespace: namespace,
		}

		got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key:      "test-secret",
			Property: "permitted",
		})

		// Must succeed — namespace is in the allowed list
		if err != nil {
			t.Logf("FAIL: namespace=%q, conditions=%+v, err=%v", namespace, store.Spec.Conditions, err)
			return false
		}
		if string(got) != "yes-"+namespace {
			t.Logf("FAIL: namespace=%q, got=%q, want=%q", namespace, string(got), "yes-"+namespace)
			return false
		}
		return true
	}, config)

	require.NoError(t, err, "Property violated: ClusterSecretStore with matching namespace should allow access")
}

// =============================================================================
// Property Test: ClusterSecretStore with matching regex GetSecret succeeds
// Validates: Requirement 3.2
// =============================================================================

// TestPreservation_ClusterSecretStore_MatchingRegex_GetSecret_Property verifies
// that GetSecret via a ClusterSecretStore where the requesting namespace matches
// a NamespaceRegex always succeeds.
func TestPreservation_ClusterSecretStore_MatchingRegex_GetSecret_Property(t *testing.T) {
	// **Validates: Requirements 3.2**
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(45)),
	}

	err := quick.Check(func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		namespace := generateNamespace(rng)
		matchingRegex := generateMatchingRegex(rng, namespace)

		// Verify our generated regex actually matches (sanity check)
		re, compileErr := regexp.Compile(matchingRegex)
		if compileErr != nil {
			// Skip invalid regex (shouldn't happen with our generator)
			return true
		}
		if !re.MatchString(namespace) {
			t.Logf("SKIP: generated regex %q does not match namespace %q", matchingRegex, namespace)
			return true
		}

		// ClusterSecretStore with regex conditions that MATCH the requesting namespace
		store := &esv1.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterSecretStore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "privx-cluster-store-regex",
			},
			Spec: esv1.SecretStoreSpec{
				Conditions: []esv1.ClusterSecretStoreCondition{
					{
						NamespaceRegexes: []string{matchingRegex},
					},
				},
			},
		}

		c := &SecretsClient{
			vault: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]any{"regex-match": "ok-" + namespace}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			store:     store,
			namespace: namespace,
		}

		got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key:      "test-secret",
			Property: "regex-match",
		})

		// Must succeed — namespace matches the regex
		if err != nil {
			t.Logf("FAIL: namespace=%q, regex=%q, err=%v", namespace, matchingRegex, err)
			return false
		}
		if string(got) != "ok-"+namespace {
			t.Logf("FAIL: namespace=%q, got=%q, want=%q", namespace, string(got), "ok-"+namespace)
			return false
		}
		return true
	}, config)

	require.NoError(t, err, "Property violated: ClusterSecretStore with matching regex should allow access")
}

// =============================================================================
// Property Test: GetAllSecrets via authorized store returns matching secrets
// Validates: Requirement 3.3
// =============================================================================

// TestPreservation_GetAllSecrets_AuthorizedStore_Property verifies that
// GetAllSecrets via an authorized store (SecretStore or ClusterSecretStore with
// matching conditions) returns matching secrets correctly.
func TestPreservation_GetAllSecrets_AuthorizedStore_Property(t *testing.T) {
	// **Validates: Requirements 3.3**
	config := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(46)),
	}

	err := quick.Check(func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		namespace := generateNamespace(rng)

		// Randomly choose between SecretStore and authorized ClusterSecretStore
		var store esv1.GenericStore
		if rng.Intn(2) == 0 {
			// Namespace-scoped SecretStore
			store = &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "privx-store",
					Namespace: namespace,
				},
				Spec: esv1.SecretStoreSpec{},
			}
		} else {
			// ClusterSecretStore with namespace in allowed list
			store = &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "ClusterSecretStore",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "privx-cluster-store",
				},
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{
						{
							Namespaces: generateNamespaceList(rng, namespace),
						},
					},
				},
			}
		}

		c := &SecretsClient{
			vault: &fakeVaultClient{
				getSecretsFn: func(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
					return &response.ResultSet[vault.Secret]{
						Count: 1,
						Items: []vault.Secret{
							{
								SecretRequest: vault.SecretRequest{
									Name: "app-secret",
								},
							},
						},
					}, nil
				},
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]any{"data": "from-" + namespace}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			store:     store,
			namespace: namespace,
		}

		got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})

		// Must succeed — store is authorized
		if err != nil {
			t.Logf("FAIL: namespace=%q, storeKind=%q, err=%v", namespace, store.GetKind(), err)
			return false
		}
		if len(got) != 1 {
			t.Logf("FAIL: namespace=%q, expected 1 secret, got %d", namespace, len(got))
			return false
		}
		if _, ok := got["app-secret"]; !ok {
			t.Logf("FAIL: namespace=%q, missing 'app-secret' key", namespace)
			return false
		}
		return true
	}, config)

	require.NoError(t, err, "Property violated: GetAllSecrets via authorized store should return matching secrets")
}

// =============================================================================
// Property Test: PushSecret via authorized store succeeds
// Validates: Requirement 3.4
// =============================================================================

// TestPreservation_PushSecret_AuthorizedStore_Property verifies that PushSecret
// via an authorized store is unaffected by namespace validation changes.
func TestPreservation_PushSecret_AuthorizedStore_Property(t *testing.T) {
	// **Validates: Requirements 3.4**
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(47)),
	}

	err := quick.Check(func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		namespace := generateNamespace(rng)
		secretName := "secret-" + generateNamespace(rng)

		// Randomly choose store type
		var store esv1.GenericStore
		switch rng.Intn(3) {
		case 0:
			// Namespace-scoped SecretStore
			store = &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "privx-store",
					Namespace: namespace,
				},
				Spec: esv1.SecretStoreSpec{},
			}
		case 1:
			// ClusterSecretStore with empty conditions
			store = &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "ClusterSecretStore",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "privx-cluster-store",
				},
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{},
				},
			}
		case 2:
			// ClusterSecretStore with matching namespace
			store = &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "ClusterSecretStore",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "privx-cluster-store",
				},
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{
						{
							Namespaces: generateNamespaceList(rng, namespace),
						},
					},
				},
			}
		}

		var pushedName string
		c := &SecretsClient{
			vault: &fakeVaultClient{
				createSecretFn: func(req *vault.SecretRequest) (vault.SecretCreate, error) {
					pushedName = req.Name
					return vault.SecretCreate{}, nil
				},
			},
			store:     store,
			namespace: namespace,
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "local-secret"},
			Data:       map[string][]byte{"key": []byte("value-" + namespace)},
		}

		pushErr := c.PushSecret(context.Background(), secret, &fakePushSecretData{
			secretKey: "key",
			remoteKey: secretName,
		})

		// Must succeed — PushSecret is unaffected by namespace validation
		if pushErr != nil {
			t.Logf("FAIL: namespace=%q, storeKind=%q, err=%v", namespace, store.GetKind(), pushErr)
			return false
		}
		if pushedName != secretName {
			t.Logf("FAIL: namespace=%q, pushed name=%q, want=%q", namespace, pushedName, secretName)
			return false
		}
		return true
	}, config)

	require.NoError(t, err, "Property violated: PushSecret via authorized store should always succeed")
}

// =============================================================================
// Property Test: ClusterSecretStore with nil conditions GetSecret succeeds
// Validates: Requirement 3.2
// =============================================================================

// TestPreservation_ClusterSecretStore_NilConditions_GetSecret_Property verifies
// that GetSecret via a ClusterSecretStore with nil conditions (no restrictions)
// always succeeds, regardless of the requesting namespace.
func TestPreservation_ClusterSecretStore_NilConditions_GetSecret_Property(t *testing.T) {
	// **Validates: Requirements 3.2**
	config := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(48)),
	}

	err := quick.Check(func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		namespace := generateNamespace(rng)

		// ClusterSecretStore with nil conditions (default) — all namespaces allowed
		store := &esv1.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterSecretStore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "privx-cluster-store-nil",
			},
			Spec: esv1.SecretStoreSpec{
				// Conditions field is nil (not set)
			},
		}

		c := &SecretsClient{
			vault: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]any{"nil-cond": "ok-" + namespace}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			store:     store,
			namespace: namespace,
		}

		got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key:      "test-secret",
			Property: "nil-cond",
		})

		// Must succeed — nil conditions means no restrictions
		if err != nil {
			t.Logf("FAIL: namespace=%q, err=%v", namespace, err)
			return false
		}
		if string(got) != "ok-"+namespace {
			t.Logf("FAIL: namespace=%q, got=%q, want=%q", namespace, string(got), "ok-"+namespace)
			return false
		}
		return true
	}, config)

	require.NoError(t, err, "Property violated: ClusterSecretStore with nil conditions should allow all namespaces")
}

// =============================================================================
// Deterministic unit tests for preservation (complement the property tests)
// =============================================================================

// TestPreservation_SecretStore_GetSecret_Deterministic verifies specific examples
// of authorized access via namespace-scoped SecretStore.
func TestPreservation_SecretStore_GetSecret_Deterministic(t *testing.T) {
	// **Validates: Requirements 3.1, 3.2**
	namespaces := []string{"default", "kube-system", "production", "dev", "monitoring"}

	for _, ns := range namespaces {
		t.Run("namespace="+ns, func(t *testing.T) {
			store := &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{Kind: "SecretStore"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "privx-store",
					Namespace: ns,
				},
				Spec: esv1.SecretStoreSpec{},
			}

			c := &SecretsClient{
				vault: &fakeVaultClient{
					getSecretFn: func(secretName string) (*vault.Secret, error) {
						data := map[string]any{"password": "secret123"}
						return &vault.Secret{
							SecretRequest: vault.SecretRequest{Data: &data},
						}, nil
					},
				},
				store:     store,
				namespace: ns,
			}

			got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
				Key:      "db-creds",
				Property: "password",
			})

			require.NoError(t, err)
			assert.Equal(t, []byte("secret123"), got)
		})
	}
}

// TestPreservation_ClusterSecretStore_PermittedNamespace_Deterministic verifies
// specific examples of authorized access via ClusterSecretStore with matching conditions.
func TestPreservation_ClusterSecretStore_PermittedNamespace_Deterministic(t *testing.T) {
	// **Validates: Requirements 3.2**
	tests := []struct {
		name       string
		namespace  string
		conditions []esv1.ClusterSecretStoreCondition
	}{
		{
			name:      "namespace in list",
			namespace: "prod",
			conditions: []esv1.ClusterSecretStoreCondition{
				{Namespaces: []string{"prod", "staging"}},
			},
		},
		{
			name:      "namespace matches regex",
			namespace: "prod-us-east-1",
			conditions: []esv1.ClusterSecretStoreCondition{
				{NamespaceRegexes: []string{"^prod-.*"}},
			},
		},
		{
			name:       "empty conditions",
			namespace:  "any-namespace",
			conditions: []esv1.ClusterSecretStoreCondition{},
		},
		{
			name:      "multiple conditions - matches second",
			namespace: "staging",
			conditions: []esv1.ClusterSecretStoreCondition{
				{Namespaces: []string{"prod"}},
				{Namespaces: []string{"staging", "dev"}},
			},
		},
		{
			name:      "regex matches all",
			namespace: "anything-goes",
			conditions: []esv1.ClusterSecretStoreCondition{
				{NamespaceRegexes: []string{".*"}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &esv1.ClusterSecretStore{
				TypeMeta:   metav1.TypeMeta{Kind: "ClusterSecretStore"},
				ObjectMeta: metav1.ObjectMeta{Name: "privx-cluster-store"},
				Spec:       esv1.SecretStoreSpec{Conditions: tc.conditions},
			}

			c := &SecretsClient{
				vault: &fakeVaultClient{
					getSecretFn: func(secretName string) (*vault.Secret, error) {
						data := map[string]any{"token": "abc123"}
						return &vault.Secret{
							SecretRequest: vault.SecretRequest{Data: &data},
						}, nil
					},
				},
				store:     store,
				namespace: tc.namespace,
			}

			got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
				Key:      "api-token",
				Property: "token",
			})

			require.NoError(t, err)
			assert.Equal(t, []byte("abc123"), got)
		})
	}
}
