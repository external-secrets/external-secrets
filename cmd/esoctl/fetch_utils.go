/*
Copyright © 2025 ESO Maintainer Team

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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	formatText   = "text"
	formatJSON   = "json"
	formatEnv    = "env"
	formatDotenv = "dotenv"
	tenMB        = 10 * 1024 * 1024
)

// loadStore loads a SecretStore or ClusterSecretStore from a YAML file.
func loadStore(file string) (esv1.GenericStore, error) {
	content, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, fmt.Errorf("failed to read store file: %w", err)
	}

	// basic sanity checks to avoid flagging
	if len(content) == 0 {
		return nil, fmt.Errorf("store file is empty")
	}

	if len(content) > tenMB {
		return nil, fmt.Errorf("store file too large")
	}

	obj := &unstructured.Unstructured{}
	// CodeQL flags this as untrusted input, but this is a CLI tool where
	// users have direct filesystem access. The file path is user-provided
	// by design, and YAML parsing is the intended functionality.
	if err := yaml.Unmarshal(content, obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal store: %w", err)
	}

	switch obj.GetKind() {
	case "SecretStore":
		store := &esv1.SecretStore{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, store); err != nil {
			return nil, fmt.Errorf("failed to convert to SecretStore (ensure apiVersion is external-secrets.io/v1): %w", err)
		}
		return store, nil

	case "ClusterSecretStore":
		store := &esv1.ClusterSecretStore{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, store); err != nil {
			return nil, fmt.Errorf("failed to convert to ClusterSecretStore (ensure apiVersion is external-secrets.io/v1): %w", err)
		}
		return store, nil

	default:
		return nil, fmt.Errorf("unsupported kind: %s (expected SecretStore or ClusterSecretStore)", obj.GetKind())
	}
}

// getKubeClient creates a Kubernetes client.
// If `--standalone` mode or no kubeconfig is available, returns a fake client with secrets.
// Otherwise, returns a real Kubernetes client.
func getKubeClient(standalone bool, authFile string) (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := esv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add v1 to scheme: %w", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add corev1 to scheme: %w", err)
	}

	if standalone {
		return createStandaloneFakeClient(authFile, scheme)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		logInfof(fetchSilent, "No kubeconfig found, using standalone mode")
		return createStandaloneFakeClient(authFile, scheme)
	}

	kubeClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return kubeClient, nil
}

// createStandaloneFakeClient creates a fake client pre-populated with secrets from auth sources.
func createStandaloneFakeClient(authFile string, scheme *runtime.Scheme) (client.Client, error) {
	var objects []client.Object //nolint:prealloc // no.

	if authFile != "" {
		secrets, err := loadSecretsFromFile(authFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load auth file: %w", err)
		}
		for i := range secrets {
			objects = append(objects, &secrets[i])
		}
	}

	envSecrets := createSecretsFromEnvironment()
	for i := range envSecrets {
		objects = append(objects, &envSecrets[i])
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()

	return fakeClient, nil
}

// loadSecretsFromFile loads secrets from a YAML file (supports multi-document with ---).
func loadSecretsFromFile(file string) ([]corev1.Secret, error) {
	content, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var secrets []corev1.Secret

	docs := strings.Split(string(content), "\n---\n")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var secret corev1.Secret
		if err := yaml.Unmarshal([]byte(doc), &secret); err == nil && secret.Kind == "Secret" {
			if secret.Namespace == "" {
				secret.Namespace = "default"
			}
			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}
			for k, v := range secret.StringData {
				secret.Data[k] = []byte(v)
			}
			secret.StringData = nil

			secrets = append(secrets, secret)
			continue
		}

		var list corev1.SecretList
		if err := yaml.Unmarshal([]byte(doc), &list); err == nil && len(list.Items) > 0 {
			for _, s := range list.Items {
				if s.Namespace == "" {
					s.Namespace = "default"
				}
				if s.Data == nil {
					s.Data = make(map[string][]byte)
				}
				for k, v := range s.StringData {
					s.Data[k] = []byte(v)
				}
				s.StringData = nil
				secrets = append(secrets, s)
			}
		}
	}

	if len(secrets) == 0 {
		return nil, fmt.Errorf("no valid Secret objects found in file")
	}

	return secrets, nil
}

// createSecretsFromEnvironment creates virtual secrets from environment variables.
func createSecretsFromEnvironment() []corev1.Secret {
	// absolute hackerman hackery.
	envSecrets := map[string]map[string]string{
		"vault-token": {
			"token": os.Getenv("VAULT_TOKEN"),
		},
		"aws-credentials": {
			"access-key-id":     os.Getenv("AWS_ACCESS_KEY_ID"),
			"secret-access-key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"session-token":     os.Getenv("AWS_SESSION_TOKEN"),
		},
		"gcp-credentials": {
			"service-account-key": os.Getenv("GCP_SERVICE_ACCOUNT_KEY"),
		},
		"azure-credentials": {
			"client-id":     os.Getenv("AZURE_CLIENT_ID"),
			"client-secret": os.Getenv("AZURE_CLIENT_SECRET"),
			"tenant-id":     os.Getenv("AZURE_TENANT_ID"),
		},
		"generic-token": {
			"token": os.Getenv("AUTH_TOKEN"),
		},
		"api-key": {
			"api-key": os.Getenv("API_KEY"),
		},
	}

	var secrets []corev1.Secret //nolint:prealloc // it's dynamic, it wouldn't be accurate to preallocate it.
	for secretName, data := range envSecrets {
		secretData := make(map[string][]byte)
		hasData := false
		for key, val := range data {
			if val != "" {
				secretData[key] = []byte(val)
				hasData = true
			}
		}

		if hasData {
			secrets = append(secrets, corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: secretData,
			})
		}
	}

	// ESO_SECRET_<NAME>_<KEY> pattern
	secretsMap := make(map[types.NamespacedName]map[string][]byte)
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "ESO_SECRET_") {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		key = strings.TrimPrefix(key, "ESO_SECRET_")
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 2 {
			continue
		}

		secretName := strings.ToLower(strings.Join(keyParts[:len(keyParts)-1], "-"))
		dataKey := strings.ToLower(keyParts[len(keyParts)-1])

		nsName := types.NamespacedName{
			Namespace: "default",
			Name:      secretName,
		}

		if secretsMap[nsName] == nil {
			secretsMap[nsName] = make(map[string][]byte)
		}
		secretsMap[nsName][dataKey] = []byte(value)
	}

	for nsName, data := range secretsMap {
		secrets = append(secrets, corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsName.Name,
				Namespace: nsName.Namespace,
			},
			Data: data,
		})
	}

	return secrets
}

// formatSecretData formats secret data based on the specified format.
func formatSecretData(data []byte, format string) (string, error) {
	switch format {
	case formatText, "":
		return string(data), nil
	case formatJSON:
		// try parsing, if failed, return as a json string
		var obj interface{}
		if err := json.Unmarshal(data, &obj); err == nil {
			formatted, err := json.MarshalIndent(obj, "", "  ")
			if err != nil {
				return "", fmt.Errorf("failed to format JSON: %w", err)
			}
			return string(formatted), nil
		}

		// if we still failed, just wrap it
		formatted, err := json.Marshal(string(data))
		if err != nil {
			return "", fmt.Errorf("failed to marshal to JSON: %w", err)
		}
		return string(formatted), nil
	default:
		return "", fmt.Errorf("unsupported format for single secret: %s (use text or json)", format)
	}
}

// formatSecretMap formats a map of secrets based on the specified format.
func formatSecretMap(data map[string][]byte, format string) (string, error) {
	switch format {
	case formatJSON, "":
		stringMap := make(map[string]string)
		for k, v := range data {
			stringMap[k] = string(v)
		}
		formatted, err := json.MarshalIndent(stringMap, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to format JSON: %w", err)
		}
		return string(formatted), nil

	case formatEnv:
		return formatAsEnv(data, false), nil

	case formatDotenv:
		return formatAsEnv(data, true), nil

	case formatText:
		return formatAsText(data), nil

	default:
		return "", fmt.Errorf("unsupported format: %s (use json, env, dotenv, or text)", format)
	}
}

// formatAsEnv formats secrets as environment variable assignments.
func formatAsEnv(data map[string][]byte, useExport bool) string {
	var lines []string

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := data[k]
		key := strings.ToUpper(strings.ReplaceAll(k, ".", "_"))
		key = strings.ReplaceAll(key, "-", "_")

		value := string(v)

		// much security, such wow
		value = strings.ReplaceAll(value, "\\", "\\\\")
		value = strings.ReplaceAll(value, "\"", "\\\"")

		if useExport {
			lines = append(lines, fmt.Sprintf("export %s=%q", key, value))
		} else {
			lines = append(lines, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return strings.Join(lines, "\n")
}

// formatAsText formats secrets as key=value pairs (plain text).
func formatAsText(data map[string][]byte) string {
	// sorting for deterministic output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", k, string(data[k])))
	}

	return strings.Join(lines, "\n")
}

// writeOutput writes the output to a file or stdout.
func writeOutput(output, outFile string) error {
	if outFile != "" {
		if err := os.WriteFile(filepath.Clean(outFile), []byte(output), 0o600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Output written to: %s\n", outFile)
	} else {
		// CodeQL flags this as clear text logging of sensitive information. Well. Yes. That's the purpose.
		fmt.Println(output)
	}
	return nil
}

// logInfof prints info messages to stderr (so they don't interfere with output).
func logInfof(silent bool, format string, args ...interface{}) {
	if silent {
		return
	}

	_, _ = fmt.Fprintf(os.Stderr, "INFO: "+format+"\n", args...)
}

// getSecretsClient creates a secrets client from a store.
func getSecretsClient(ctx context.Context, store esv1.GenericStore, namespace string, standalone bool, authFile string) (esv1.SecretsClient, error) {
	kubeClient, err := getKubeClient(standalone, authFile)
	if err != nil {
		return nil, err
	}

	provider, err := esv1.GetProvider(store)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	klient, err := provider.NewClient(ctx, store, kubeClient, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider client: %w", err)
	}

	return klient, nil
}
