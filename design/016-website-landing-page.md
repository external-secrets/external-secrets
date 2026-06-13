```yaml
---
title: Simple Landing Page for external-secrets.io
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2026-01-05
status: draft
---
```

# Simple Landing Page for external-secrets.io

## Summary

Create a simple, single-page landing site for external-secrets.io inspired by [kured.dev](https://kured.dev/) or cilium's. The landing page will provide a clear value proposition, key features, and direct users to the documentation. The existing mkdocs documentation remains at `/latest/` (and versioned paths).

## Motivation

The current external-secrets.io immediately redirects to `/latest/` documentation. While comprehensive, this approach:

- Lacks a clear "front door" for new visitors
- Doesn't communicate the value proposition quickly
- Misses opportunity to showcase project maturity (CNCF status, adopters)
- Doesn't match the professional presence of peer CNCF projects

Projects like [kured.dev](https://kured.dev/), [fluxcd.io](https://fluxcd.io/), and [argoproj.github.io](https://argoproj.github.io/argo-cd/) give "better" first impressions while maintaining easy access to documentation.

### Goals

1. Create a simple, static landing page at `external-secrets.io/`
2. Clearly communicate what ESO does in 10 seconds
3. Highlight 3-4 key differentiators
4. Provide clear paths to documentation and community content
5. Maintain all existing documentation URLs (no breaking changes)

### Non-Goals

- Contain News/Blog (future enhancement, but planned for hosting our news/videos/...)
- Separate adopters page (inline on landing page)
- Complete docs refactor (structure or tooling)

## Proposal

### URL Structure

No change.

```text
external-secrets.io/
├── /latest/             # EXISTING: Current docs (mkdocs)
├── /v0.x.x/             # EXISTING: Versioned docs (mkdocs)
└── /main/               # EXISTING: Main branch docs (mkdocs)
```

The landing page lives at the root. Documentation remains at `/latest/` and versioned paths. No existing URLs change.

### Landing Page Structure

```
┌───────────────────────────────────────────────────────────────────────┐
│  [Logo]  External Secrets Operator   [Search] [Docs] [News] [GitHub]  │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│              External Secrets Operator                                │
│                                                                       │
│    Securely Sync secrets from external APIs into Kubernetes           │
│                                                                       │
│         [Get Started]          [View on GitHub]                       │
│                                                                       │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────┐  ┌─────────────┐       │
│  │   40+       │  │  Kubernetes │  │  Bi-dir │  │  Dynamic    │       │
│  │  Providers  │  │   Native    │  │   Sync  │  │ Generators  │       │
│  └─────────────┘  └─────────────┘  └─────────┘  └─────────────┘       │
│                                                                       │
│      [CNCF Badge] (Sandbox/Incubating/...)                            │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  Trusted by:  [Adopter logos - placeholder]                           │
│                                                                       │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  [Slack] [GitHub] [Twitter] [YouTube] [Calendar]                      │
│                                                                       │
│  © 2025 The external-secrets Authors. CNCF Project.                   │
│                                                                       │
├───────────────────────────────────────────────────────────────────────┤
```

### Key Features to Highlight

Based on ESO's capabilities, highlight these 4 differentiators:

| Feature                | Headline                        | Description                                                                                                          |
|------------------------|---------------------------------|----------------------------------------------------------------------------------------------------------------------|
| **40+ Providers**      | "Connect to Any Secret Store"   | AWS Secrets Manager, HashiCorp Vault, Azure Key Vault, GCP Secret Manager, and 40+ more providers out of the box.    |
| **Kubernetes Native**  | "Declarative Secret Management" | Define secrets as Kubernetes custom resources. ESO handles synchronization automatically using controllers and CRDs. |
| **Bidirectional Sync** | "Pull and Push Secrets"         | Sync secrets from external stores into Kubernetes, or push Kubernetes secrets to external providers.                 |
| **Dynamic Generators** | "Generate Secrets On-Demand"    | Automatically generate passwords, SSH keys, container registry credentials, and more with built-in generators.       |

Side note: I wonder if "importable" is not a benefit too. (You cannot manage CRDs with your own controller if you are not using CRDs, like for example CSI driver...)

### Technical Implementation

Use our mkdocs for the home page to keep a single toolchain.

```yaml
# mkdocs.yml
theme:
  custom_dir: overrides

# overrides/home.html - custom landing page template
```

This gives us:
- Single build system (mkdocs only)
- Search across the whole website
- Simpler CI/CD

In other words: This is better if team prefers single toolchain.

The biggest alternative is to build our docs with Hugo. It is recommended for better landing page flexibility. See alternatives section.

### Design Guidelines

#### Visual Style

- Clean, minimal design (like kured.dev)
- ESO brand colors from existing logo
- Mobile-responsive (single column on mobile)

#### Developer experience

- No JavaScript frameworks knowledge required

### User Stories

**As a new visitor**, I want to understand what ESO does in 10 seconds, so I can decide if it's relevant to my needs.

**As an evaluator**, I want to see key features and who uses ESO, so I can assess project maturity.

**As a user**, I want quick access to documentation, so I can start implementing. If possible, search for my provider and get all the relevant content.

**As a developer**, I want to quickly know what I need to do to contribute.

### API

No API changes. This is a website enhancement.

### Behavior

- `external-secrets.io/latest/` continues to work (documentation)
- All existing documentation URLs unchanged

### Drawbacks

- Less flexible for marketing pages
- mkdocs Material home page is still docs-oriented
- Harder to achieve a "kured.dev" simplicity

### Acceptance Criteria

- [ ] All key features displayed
- [ ] Links to docs work (`/latest/`, `/latest/introduction/getting-started/`)
- [ ] Links to community (Slack, GitHub, Twitter) work
- [ ] Mobile responsive (tested on 375px width)
- [ ] Page weight < 200KB
- [ ] CNCF badge displayed
- [ ] Adopter logos section (placeholder, content TBD)
- [ ] CI/CD builds and deploys both landing page and docs

## Alternatives

### Alternative: Hugo Home Page

Use Hugo instead of Material's [custom home page](https://squidfunk.github.io/mkdocs-material/setup/setting-up-navigation/#custom-home-page)

We will miss the search from mkdocs.

#### Repository Structure

Add a `/website/` directory to the existing repository:

```text
external-secrets/
├── website/                    # NEW
│   ├── hugo.toml               # Hugo configuration
│   ├── content/
│   │   └── _index.md           # Landing page content
│   ├── layouts/
│   │   └── index.html          # Landing page template
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css       # Minimal custom CSS
│   │   └── images/
│   │       ├── logo.svg
│   │       └── adopters/       # Adopter logos
│   └── themes/                 # Hugo theme (or custom)
├── docs/                       # EXISTING: mkdocs source
├── hack/api-docs/mkdocs.yml    # EXISTING: mkdocs config
└── ...
```

Alternatively, we can do the website on another git repo to not pollute our main content. 

#### Hugo Configuration

```toml
# website/hugo.toml
baseURL = "https://external-secrets.io/"
languageCode = "en-us"
title = "External Secrets Operator"

[params]
  description = "Kubernetes operator that integrates external secret management systems"
  github = "https://github.com/external-secrets/external-secrets"
  slack = "https://kubernetes.slack.com/messages/external-secrets"
  twitter = "https://twitter.com/ExtSecretsOptr"
  docs = "/latest/"

# Minimal theme - single page doesn't need much
[module]
  # Use a minimal theme or custom layouts
```

**Pros:**
- Works out of the box
- Easy and fast to build

**Cons:**
- **Two build systems**: Hugo + mkdocs adds complexity. At the same time, Hugo build is minimal (single page), well-documented in CI
- **Maintenance**: Another thing to keep updated. Landing page content is stable, rarely changes. Only need to bump hugo in our tooling.

### Variant: Separate Repository

Create a new `external-secrets/website` repository for the landing page.

**Pros:**
- Complete separation of concerns
- Independent release cycle

**Cons:**
- Two repositories to maintain
- Harder to keep in sync (ADOPTERS)
- More complex deployment coordination

**Decision:** Rejected - single repository is simpler for a single landing page, especially when using mkdocs.
If using hugo, that makes things simpler.

### Variant: Subdomain for Docs

Use `docs.external-secrets.io` for documentation, `external-secrets.io` for landing.

**Pros:**
- Clean URL separation
- Independent deployments

**Cons:**
- Requires DNS changes
- Breaks existing documentation URLs (major)
- More infrastructure to manage

**Decision:** Rejected - breaking existing URLs is not friendly.
However, if using hugo, it makes the separation very clear: docs.{domain} is through material, the rest is in hugo.

## Compatibility with Other Designs

This proposal is independent of other designs. It affects only the website, not the operator code.

## Content Placeholders

The following content needs to be filled in before launch:

| Placeholder   | Owner       | Notes                                                 |
|---------------|-------------|-------------------------------------------------------|
| CNCF badge    | Maintainers | Need to update when going to Incubating or Graduated. |
| Adopter logos | Community   | Collect from known users                              |
| Social links  | Maintainers | Ensure our link checkers always work.                 |
| Analytics ID  | Maintainers | Reuse existing or create new?                         |
