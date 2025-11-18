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
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var (
	fetchStoreFile  string
	fetchKey        string
	fetchProperty   string
	fetchFormat     string
	fetchOutFile    string
	fetchSilent     bool
	fetchDryRun     bool
	fetchNamespace  string
	fetchVersion    string
	fetchStandalone bool
	fetchAuthFile   string
)

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringVar(&fetchStoreFile, "store", "", "Path to SecretStore or ClusterSecretStore YAML file (required)")
	fetchCmd.Flags().StringVar(&fetchKey, "key", "", "Secret key/path to fetch (required)")
	fetchCmd.Flags().StringVar(&fetchProperty, "property", "", "Property/field to extract from the secret (optional), if not defined the entire secret is fetched")
	fetchCmd.Flags().StringVar(&fetchFormat, "format", "text", "Output format: text, json")
	fetchCmd.Flags().StringVar(&fetchOutFile, "outfile", "", "Write output to file instead of stdout")
	fetchCmd.Flags().BoolVar(&fetchSilent, "silent", false, "Don't output log messages")
	fetchCmd.Flags().BoolVar(&fetchDryRun, "dry-run", false, "Don't actually do anything other than verifying that the input is okay")
	fetchCmd.Flags().StringVar(&fetchNamespace, "namespace", "default", "Namespace for referent secrets (default: default)")
	fetchCmd.Flags().StringVar(&fetchVersion, "version", "", "Secret version (provider-specific, optional)")
	fetchCmd.Flags().BoolVar(&fetchStandalone, "standalone", false, "Use standalone mode (no Kubernetes cluster required)")
	fetchCmd.Flags().StringVar(&fetchAuthFile, "auth-file", "", "Path to auth secrets file (YAML with Secret objects)")

	_ = fetchCmd.MarkFlagRequired("store")
	_ = fetchCmd.MarkFlagRequired("key")
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch a single secret from a secret store",
	Long: `Fetch a single secret value from a configured secret store.

This command reads a SecretStore or ClusterSecretStore configuration and fetches
a single secret value from the provider. Requires access to a Kubernetes cluster
for authentication credentials.

Examples:
  # Fetch a secret and display as text
  esoctl fetch --store vault-store.yaml --key /secret/data/myapp/password

  # Fetch and extract a specific property
  esoctl fetch --store vault-store.yaml --key /secret/data/myapp/config --property dbHost

  # Fetch as JSON
  esoctl fetch --store vault-store.yaml --key /secret/data/myapp/config --format json

  # Fetch and save to file (redacting logs)
  esoctl fetch --store vault-store.yaml --key /secret/data/myapp/password --outfile secret.txt --redact-logs

  # Fetch with specific version
  esoctl fetch --store vault-store.yaml --key /secret/data/myapp/password --version v2
`,
	RunE: fetchRun,
}

func fetchRun(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	logInfo(fetchSilent, "Loading store from: %s", fetchStoreFile)

	store, err := loadStore(fetchStoreFile)
	if err != nil {
		return err
	}

	logInfo(fetchSilent, "Store loaded: %s/%s (kind: %s)", store.GetNamespace(), store.GetName(), store.GetObjectKind().GroupVersionKind().Kind)

	client, err := getSecretsClient(ctx, store, fetchNamespace, fetchStandalone, fetchAuthFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	logInfo(fetchSilent, "Fetching secret with key: %s", fetchKey)

	remoteRef := esv1.ExternalSecretDataRemoteRef{
		Key:      fetchKey,
		Property: fetchProperty,
		Version:  fetchVersion,
	}

	var output string
	if fetchProperty != "" {
		secretData, err := client.GetSecret(ctx, remoteRef)
		if err != nil {
			if errors.Is(err, esv1.NoSecretErr) {
				return fmt.Errorf("secret not found: %s", fetchKey)
			}
			return fmt.Errorf("failed to fetch secret: %w", err)
		}

		if output, err = formatSecretData(secretData, fetchFormat); err != nil {
			return fmt.Errorf("failed to format secret data: %w", err)
		}
	} else {
		secretData, err := client.GetSecretMap(ctx, remoteRef)
		if err != nil {
			if errors.Is(err, esv1.NoSecretErr) {
				return fmt.Errorf("secret not found: %s", fetchKey)
			}
			return fmt.Errorf("failed to fetch secret map: %w", err)
		}
		if len(secretData) == 0 {
			logInfo(fetchSilent, "No secrets found at key: %s", fetchKey)

			return nil
		}

		if output, err = formatSecretMap(secretData, fetchFormat); err != nil {
			return fmt.Errorf("failed to format secret data map: %w", err)
		}
	}

	return writeOutput(output, fetchOutFile)
}
