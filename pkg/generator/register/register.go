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

// Package register provides registration functionality for generators.
package register

import (
	// Import all generators for their side effects (registration).
	_ "github.com/external-secrets/external-secrets/pkg/generator/acr"
	_ "github.com/external-secrets/external-secrets/pkg/generator/cloudsmith"
	_ "github.com/external-secrets/external-secrets/pkg/generator/ecr"
	_ "github.com/external-secrets/external-secrets/pkg/generator/fake"
	_ "github.com/external-secrets/external-secrets/pkg/generator/gcr"
	_ "github.com/external-secrets/external-secrets/pkg/generator/github"
	_ "github.com/external-secrets/external-secrets/pkg/generator/grafana"
	_ "github.com/external-secrets/external-secrets/pkg/generator/mfa"
	_ "github.com/external-secrets/external-secrets/pkg/generator/password"
	_ "github.com/external-secrets/external-secrets/pkg/generator/quay"
	_ "github.com/external-secrets/external-secrets/pkg/generator/sshkey"
	_ "github.com/external-secrets/external-secrets/pkg/generator/sts"
	_ "github.com/external-secrets/external-secrets/pkg/generator/uuid"
	_ "github.com/external-secrets/external-secrets/pkg/generator/vault"
	_ "github.com/external-secrets/external-secrets/pkg/generator/webhook"
)
