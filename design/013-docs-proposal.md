---
title: Complete Docs refactor proposal
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2026-03-10
status: draft
---

# Docs refactor proposal

## Summary

ESO documentation fails in (at least) three recurring ways: users cannot find setup instructions, cannot determine feature support status, and cannot follow a clear path through the site. This proposal defines five sequential steps to fix it. Steps 1 and 3 are optional accelerators; Steps 2, 4, and 5 are required.

## Motivation

ESO receives three categories of recurring support questions that documentation should answer without human intervention:

- Setup: users cannot find or follow instructions for feature X
- Support status: the feature matrix is ambiguous; users cannot determine what is alpha, beta, or stable
- Navigation: users do not know where to go next, what to read, or how to write a contribution

Undocumented decisions and an undefined content architecture also block automated content generation.

### Goals

By the end of this proposal:

- external-secrets.io serves content per persona: adopters, operators, contributors, security reviewers
- Feature support status per provider is unambiguous and machine-generated from source
- The site supports multi-project navigation across ESO, reloader, and the CLI
- Maintenance cost per release is reduced: no manual feature matrix updates, no manual structure decisions

### Non-Goals

- Documentation for sub-projects (CLI, reloader) beyond navigation and cross-linking
- Content rewrites outside the ESO main documentation

## Proposal

### Step 1 (Optional): New multi-project website

We create a single landing site for external-secrets.io modeled after kured.dev. The landing page states the value proposition, lists key features, and routes users to per-project documentation.

Top-level structure:
- Sub-project docs (ESO, reloader, CLI) as first-class sections
- Releases as a top-level page
- Community links shared across all projects

#### Actual implementation

1. Create a dedicated repository for the new website (separate from the main ESO repo).
2. Implement Hugo with Docsy as the static site generator, matching the CNCF ecosystem baseline.
3. Customize Docsy's build process, JS, and templates to support coherent search across sub-project versions without an external search service.
4. Port existing ESO documentation to the new site structure, verifying all pages render correctly.
5. Update CI pipelines to build, test, and publish the new site on every commit.
6. Redirect `external-secrets.io` to the new site.
7. Adapt our documentation process: Deprecate our old docs/ folder in our main repo, adapt the new website build pipeline to generate the previously generated content from the ESO repo.
8. Adapt the release process documentation and github workflows for the new repository.

Demo: https://github.com/evrardj-roche/external-secrets-website

#### Acceptance criteria

The new site is live at https://external-secrets.io, replacing the current landing page, with CI building and publishing it automatically on release.

#### Side-effects

The release management pipeline must be updated to build and publish the new site. If Step 1 is implemented, this pipeline update must be completed before Step 2 begins.

### Step 2: Agree on a documentation architecture in a community meeting

The current content architecture has no defined personas, no enforced structure per section, and provider documentation that has grown unreadably long (based on provider).
This is already visible today and may become more apparent if Step 1 is implemented. Step 2 fixes the foundation and can be pursued independently of Step 1.

We adopt Diataxis (https://www.diataxis.fr) as the basis for content framework. Diataxis defines four documentation modes (tutorials, how-tos, explanation, reference) and **maps each to a reader goal**. It does not require a dedicated content architect to enforce.

The content/pages hierarchy does not change on day 1. The writing process changes immediately.
Important note: The hierarchy might not map (sensu stricto) to Diataxis, it will depend on reader goals/personas.

#### Actual implementation

1. Draft a proposed persona list (by doing a PR on this design document) and share it on the mailing list / Slack for async feedback before the community meeting.
   We already know the following persona must be present:

   * A first time user of ESO learning ropes by doing its first deployment of ESO
   * A person who wants to know how to configure feature X of ESO, regardless of the provider (exemple Configuring FluxCD for ESO, triggering secret refresh, share secrets between namespaces, ...).
   * A security person who wants to know all the internal details about ESO (Threat model ...)
   * An "ESO Developer/Contributor" who wants to know how to contribute
   * A person who wants to know definitive information about how we work (Governance, Code of Conduct, LLM Policy, ...)

   More personas can be added.
   The next PR targetting this implementation should have an exhaustive persona list and the content structure. This is not necessary as of today, as the documentation structure should not prevent the plan to be acted upon.

2. Run the idea in a community meeting to get consensus (if no consensus reached on PR) to finalize and ratify the reader personas (we can still merge/reject the PR asynchronously based on the comments of the meeting).
3. Publish the agreed persona list and content structure in `CONTRIBUTING.md` and our contributing guide.
4. Add a CI check that rejects PRs adding documentation files outside the agreed persona-scoped paths.

#### Acceptance criteria

- Community meeting minutes record the agreed persona list;
- A persona-scoped path structure is documented in this design file, in a follow-up PR;
- CI fails on PRs that add documentation outside a persona-scoped path listed in this document.

#### Side-effects

Documentation will temporarily fragment across more files as new content follows our adapted Diataxis layout while old content does not. This is expected and will be resolved in Step 3.

### Step 3 (Optional): Accelerate the adaptation of our documentation based on these new foundations

Rewriting all existing content in one go is not feasible for a single contributor (reference: https://github.com/external-secrets/external-secrets/pull/5822). This step is optional: the community decides whether to resource it.

#### Actual implementation

1. Assign ownership of each top-level documentation section to a volunteer (one owner per section).
2. Rewrite sections one at a time: Tutorials first (highest user impact), then How-tos, then Reference, then Explanation.
3. Each rewritten section is submitted as a standalone PR, reviewed against the structure agreed in Step 2, and merged independently.
4. Optionally, publish a set of LLM-based writing skills to help contributors write to **our** adapted Diataxis layout.

#### Acceptance criteria

- Inventory of all unreleased pages done;
- All existing unreleased pages are rewritten for their target persona;
- Top-level structure is reduced to: Providers / Guides / How-tos / Reference / Tutorials (or the structure defined in step 2);
- (Optional) At least one LLM writing skill is available in the contributor toolchain.

#### Side-effects

The final structure cannot be known in advance. Step 4 must not depend on it.

### Step 4: Clarify governance around features stability and their documentation

ESO contributors and users have no authoritative definition of alpha, beta, and stable for features and providers. The support matrix is manually maintained and inconsistently applied.

We define and publish a lifecycle policy for features and providers.

#### Actual implementation

1. Assign ownership to a person in charge of defining the provider/feature lifecycle (alpha / beta / stable criteria, promotion and demotion rules) in a new governance documentation section, focusing about provider/feature lifecycle.
2. The PR is shared for async review (min. 1 week) before the community meeting.
3. The lifecycle policy is ratified in a community meeting and merged, according to our governance rules.
4. Apply the new lifecycle labels retroactively to all existing providers and features via a follow-up PR.

#### Acceptance criteria

- The lifecycle policy is merged into the main branch;
- All providers are updated with their appropriate lifecycle labels.

#### Side-effects

Some providers or features will be reclassified. The reclassification must be communicated clearly to avoid unexpected user impact.

### Step 5: Automatically generate lifecycle events in our documentation, from our codebase.

With the Step 4 policy in place, the feature support matrix for each provider can be generated from code, not written by hand on our docs.

#### Actual implementation

1. Define the Go registry interface: each provider registers its name, supported features, and their maturity level.

`pkg/controllers/secretstore/common.go` can be adapted to get capabilities of each provider implementation, which will be useful for v2 anyway:

```
capabilities, found := provider.GetCapabilities(ss.GetName()) // Probably update this to use Provider type instead of name, but you get the gist.
	if !found {
		return ctrl.Result{}, fmt.Errorf("provider capabilities not found in registry")
	}
	capStatus := esapi.SecretStoreStatus{
		Capabilities: capabilities,
		Conditions:   ss.GetStatus().Conditions,
	}
```

This means changing the provider interface

`apis/externalsecrets/v1/provider.go` and remove its capabilities there.

Also, `runtime/feature/feature.go` gets extended:

```
type Feature struct {
	Flags      *pflag.FlagSet
	Initialize func()
	Maturity   Maturity
	Safety     Safety
}

type Maturity string
type Safety string

const (
	Experimental    Maturity = "experimental"
	Stable          Maturity = "stable"
	Deprecated      Maturity = "deprecated"
	UnknownMaturity Maturity = "unknown"

	Unsafe        Safety = "insecure"
	Safe          Safety = "secure"
	UnknownSafety Safety = "unknown"
)
```

This means we have a standardize feature maturity/safety behaviour, and so does the providers:

`runtime/provider/capabilities.go` :

```
package provider

// CapabilityName represents a specific operation a provider can perform
type CapabilityName string

// A series of capability Names for standard implementations.
// This can _later_ be moved to more interfaces in this package, guaranteeing an easy observation
const (
	CapabilityGetSecret              CapabilityName = "GetSecret"
	CapabilityGetSecretMap           CapabilityName = "GetSecretMap"
	CapabilityGetAllSecrets          CapabilityName = "GetAllSecrets"
	CapabilityPushSecret             CapabilityName = "PushSecret"
	CapabilityDeleteSecret           CapabilityName = "DeleteSecret"
	CapabilitySecretExists           CapabilityName = "SecretExists"
	CapabilityValidate               CapabilityName = "Validate"
	CapabilityValidateStore          CapabilityName = "ValidateStore"
	CapabilityFindByName             CapabilityName = "FindByName"
	CapabilityFindByTag              CapabilityName = "FindByTag"
	CapabilityMetadataPolicyFetch    CapabilityName = "MetadataPolicyFetch"
	CapabilityReferentAuthentication CapabilityName = "ReferentAuthentication"
	CapabilityDeletionPolicy         CapabilityName = "DeletionPolicy"
)

// Capability describes a capability with optional notes
type Capability struct {
	Name  CapabilityName `json:"name"`
	Notes string         `json:"notes,omitempty"`
}
```

and  `runtime/provider/metadata.go`:

```
package provider

import v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

// Name contains the name of the provider in the Provider (feature) registry
type Name string

// Metadata represents the complete view of provider supportability state. It must be implemented in each provider.
type Metadata struct {
	Stability    Stability    `json:"stability"`
	Capabilities []Capability `json:"capabilities,omitempty"`
	Comment      string       `json:"comment,omitempty"`
}

// Stability represents the maturity level of a provider
type Stability string

const (
	StabilityAlpha        Stability = "Alpha"
	StabilityBeta         Stability = "Beta"
	StabilityStable       Stability = "Stable"
	StabilityUnmaintained Stability = "Unmaintained"
	StabilityDeprecated   Stability = "Deprecated"
)

// MaintenanceStatus derives the MaintenanceStatus reporter to the user based on Provider Metadata, see https://github.com/external-secrets/external-secrets/issues/5494
func (m Metadata) MaintenanceStatus() v1.MaintenanceStatus {
	if m.Stability == StabilityUnmaintained {
		return v1.MaintenanceStatusNotMaintained
	}
	if m.Stability == StabilityDeprecated {
		return v1.MaintenanceStatusDeprecated
	}
	return v1.MaintenanceStatusMaintained
}

// APICapabilities derives the SecretStore capabilities (ReadOnly/WriteOnly/ReadWrite)
// from the provider's full capability list in their metadata, to be used in API.
func (m Metadata) APICapabilities() v1.SecretStoreCapabilities {
	var canRead, canWrite bool

	for _, capability := range m.Capabilities {
		switch capability.Name {
		case CapabilityGetSecret, CapabilityGetSecretMap, CapabilityGetAllSecrets:
			canRead = true
		case CapabilityPushSecret, CapabilityDeleteSecret:
			canWrite = true
		}
	}

	if canRead && canWrite {
		return v1.SecretStoreReadWrite
	}
	if canWrite {
		return v1.SecretStoreWriteOnly
	}
	return v1.SecretStoreReadOnly // As currently done, but we should think about NoCapabilitiesReported constant
}

// MetadataReporter is a way to guarantee each provider will report their metadata.
type MetadataReporter interface {
	Metadata() Metadata
}
```

See also the rest of the implementation here: https://github.com/external-secrets/external-secrets/compare/main...evrardj-roche:external-secrets:feature-flags

2. Migrate all providers to register into the central registry (one PR per provider, or batched).

For exemple, a provider implementation could look like this:

```
package akeyless

import (
	"github.com/external-secrets/external-secrets/runtime/provider"
)

var metadata = provider.Metadata{
	Stability: provider.StabilityStable,
	Capabilities: []provider.Capability{
		{Name: provider.CapabilityGetSecret},
		{Name: provider.CapabilityGetSecretMap},
		{Name: provider.CapabilityGetAllSecrets},
		{Name: provider.CapabilitySecretExists},
		{Name: provider.CapabilityValidate},
		{Name: provider.CapabilityValidateStore},
		{Name: provider.CapabilityFindByName},
		{Name: provider.CapabilityFindByTag},
		{Name: provider.CapabilityPushSecret},
		{Name: provider.CapabilityDeletionPolicy},
		{Name: provider.CapabilityReferentAuthentication},
	},
}

// Metadata returns the package-level metadata for the akeyless provider.
func Metadata() provider.Metadata {
	return metadata
}

func init() {
	provider.Register("akeyless", NewProvider(), ProviderSpec())
}
```

The registering of the provider in registry becomes:
```
package register

import (
	_ "github.com/external-secrets/external-secrets/providers/v1/akeyless"
)
```

(which further reduces our code).

As a side benefit, going through this code will standardize metadata handling across providers and provider's file structure.

3. Write a Go tool (`cmd/docgen` or similar) that reads the registry and emits support matrix markdown for each provider.

Example from my previous PoC, which can be improved over time, but gives an idea of how it could work:

(Do not remove the backticks if you are trying this locally!)

```

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"sort"
	"text/template"

	_ "github.com/external-secrets/external-secrets/pkg/register"
	runtimeprovider "github.com/external-secrets/external-secrets/runtime/provider"
)

const markdownTemplate = `---
title: Provider Capabilities Matrix
description: Auto-generated documentation of provider capabilities
---
# Provider Capabilities Matrix
This page lists all External Secrets Operator providers and their supported capabilities.
## Capabilities Matrix
| Provider | Stability | GetSecret | FindByName | FindByTag | PushSecret | DeleteSecret | ReferentAuth | MetadataPolicy | ValidateStore |
|----------|-----------|-----------|------------|-----------|------------|--------------|--------------|----------------|---------------|
{{- range $provider := .}}
| {{$provider.Name}} | {{$provider.Stability}} |{{if index $provider.DisplayedCapabilities "GetSecret"}} ✓ |{{else}}|{{end}}{{if index $provider.DisplayedCapabilities "FindByName"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "FindByTag"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "PushSecret"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "DeleteSecret"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "ReferentAuthentication"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "MetadataPolicyFetch"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "ValidateStore"}} ✓ |{{else}}|{{end}}{{- end}}
## Extra Capabilities per provider
{{range .}}
{{if .ExtraCapabilities}}
### {{.Name}} provider capabilities
{{range .ExtraCapabilities}}
- ` + "`{{.Name}}`" + `{{if .Notes}}: {{.Notes}}{{end}}
{{- end}}
{{end}}
{{end }}
`

var (
	outputFormat        = flag.String("format", "md", "Output format: json or md")
	capabilityShortList = []runtimeprovider.CapabilityName{
		runtimeprovider.CapabilityGetSecret,
		runtimeprovider.CapabilityFindByName,
		runtimeprovider.CapabilityFindByTag,
		runtimeprovider.CapabilityPushSecret,
		runtimeprovider.CapabilityDeleteSecret,
		runtimeprovider.CapabilityReferentAuthentication,
		runtimeprovider.CapabilityMetadataPolicyFetch,
		runtimeprovider.CapabilityValidateStore,
	}
)

func main() {
	flag.Parse()

	providers := runtimeprovider.List()
	matrix := make(map[runtimeprovider.Name]runtimeprovider.Metadata, len(providers))

	for providerName, providerData := range providers {
		matrix[providerName] = providerData.Metadata
	}

	var output string
	switch *outputFormat {
	case "json":
		output = generateJSON(matrix)
	case "md":
		output = generateMarkdown(matrix)
	default:
		log.Fatalf("Unknown format: %s", *outputFormat)
	}

	fmt.Print(output)
}

func generateJSON(matrix map[runtimeprovider.Name]runtimeprovider.Metadata) string {
	data, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	return string(data)
}

type providerInfo struct {
	Name                  string
	Stability             string
	DisplayedCapabilities map[string]bool
	ExtraCapabilities     []runtimeprovider.Capability
}

type providerTable []providerInfo

func (p providerTable) Len() int {
	return len(p)
}
func (p providerTable) Less(i, j int) bool {
	return p[i].Name < p[j].Name
}
func (p providerTable) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func newProviderTable(matrix map[runtimeprovider.Name]runtimeprovider.Metadata) providerTable {
	var table providerTable
	for pn, metadata := range matrix {
		line := providerInfo{}
		line.Name = string(pn)
		line.Stability = string(metadata.Stability)
		line.DisplayedCapabilities = make(map[string]bool)

		// Build a map of provider capabilities for quick lookup
		providerCapabilities := make(map[runtimeprovider.CapabilityName]bool)
		for _, capability := range metadata.Capabilities {
			providerCapabilities[capability.Name] = true
		}

		// Check which short-list capabilities this provider has
		for _, capability := range capabilityShortList {
			line.DisplayedCapabilities[string(capability)] = providerCapabilities[capability]
		}

		// Extend the info with metadata outside the short list
		for _, providerCapability := range metadata.Capabilities {
			if _, ok := line.DisplayedCapabilities[string(providerCapability.Name)]; !ok {
				line.ExtraCapabilities = append(line.ExtraCapabilities, providerCapability)
			}
		}
		table = append(table, line)
	}
	sort.Sort(table)
	return table
}

func generateMarkdown(matrix map[runtimeprovider.Name]runtimeprovider.Metadata) string {
	sortedTable := newProviderTable(matrix)

	// Execute template
	tmpl, err := template.New("markdown").Parse(markdownTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, sortedTable); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	return buf.String()
}
```

4. Integrate the tool into the docs build: CI runs it and the output is committed to our docs expected file for support matrix (actual file location will depend on previous steps).
5. Add a CI diff check that blocks PRs manually editing such file.

This step supersedes design proposal 014: https://github.com/external-secrets/external-secrets/blob/81078c9ab6a7cf2ddbd0fe5856188a120e09e87a/design/014-feature-flag-consolidation.md

#### Acceptance criteria

- Feature support documentation is fully generated; no manual edits are possible
- CI enforces this with a diff check on the generated files

#### Side-effects

None beyond the suppression of proposal 014.

### User Stories

- As a user, I need step-by-step instructions for configuring feature X without opening a Slack thread.
- As a user, I know exactly the support and maturity status of current feature/provider.
- As an advocate, I need a one-page summary of what ESO does that I can share without asking the team to write one.
- As a new contributor, I need a single linear path through environment setup, triage, and first PR without chasing links across three separate pages.

### API

No API change.

### Behavior

No ESO API/runtime behavior change. Cosmetic and user experience changes only.

### Drawbacks

None identified.

### Acceptance Criteria

See each step's acceptance criteria above.

## Alternatives

Doing nothing is the rejected alternative. The current state produces recurring support load, blocks automation, and degrades contributor onboarding quality with each added provider.
