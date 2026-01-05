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

## Table of Contents

<!-- toc -->
<!-- /toc -->

## Summary

Create a simple, single-page landing site for external-secrets.io inspired by [kured.dev](https://kured.dev/). The landing page will provide a clear value proposition, key features, and direct users to the documentation. The existing mkdocs documentation remains at `/latest/` (and versioned paths).

## Motivation

The current external-secrets.io immediately redirects to `/latest/` documentation. While comprehensive, this approach:

- Lacks a clear "front door" for new visitors
- Doesn't communicate the value proposition quickly
- Misses opportunity to showcase project maturity (CNCF status, adopters)
- Doesn't match the professional presence of peer CNCF projects

Projects like [kured.dev](https://kured.dev/), [fluxcd.io](https://fluxcd.io/), and [argoproj.github.io](https://argoproj.github.io/argo-cd/) demonstrate that a simple landing page significantly improves first impressions while maintaining easy access to documentation.

### Goals

1. Create a simple, static landing page at `external-secrets.io/`
2. Clearly communicate what ESO does in 10 seconds
3. Highlight 3-4 key differentiators
4. Provide clear paths to documentation and community
5. Maintain all existing documentation URLs (no breaking changes)

### Non-Goals

- News/Blog section (future enhancement, but planned for hosting our news/videos/...)
- Separate adopters page (inline on landing page)
- Complete website redesign
- Changing documentation structure or tooling

## Proposal

### URL Structure

```
external-secrets.io/
├── /                    # NEW: Landing page (mkdocs)
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

### Technical Implementation

Use our mkdocs for the home page to keep a single toolchain.

**Implementation:**
```yaml
# mkdocs.yml
theme:
  custom_dir: overrides

# overrides/home.html - custom landing page template
```

**Pros:**
- Single build system (mkdocs only)
- Search across the whole website
- Simpler CI/CD

In other words: This is better if team prefers single toolchain. Hugo recommended for better landing page flexibility.

#### Landing Page Template

```html
<!-- website/layouts/index.html -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>External Secrets Operator</title>
    <meta name="description" content="Kubernetes operator that integrates external secret management systems">
    <link rel="stylesheet" href="/css/style.css">
    <link rel="icon" href="/images/logo.svg">
</head>
<body>
    <!-- Navigation -->
    <nav>
        <a href="/" class="logo">
            <img src="/images/logo.svg" alt="ESO">
            <span>External Secrets</span>
        </a>
        <div class="nav-links">
            <a href="/latest/">Docs</a>
            <a href="https://github.com/external-secrets/external-secrets">GitHub</a>
        </div>
    </nav>

    <!-- Hero -->
    <section class="hero">
        <h1>External Secrets Operator</h1>
        <p class="tagline">Sync secrets from external APIs into Kubernetes</p>
        <div class="cta-buttons">
            <a href="/latest/introduction/getting-started/" class="btn primary">Get Started</a>
            <a href="https://github.com/external-secrets/external-secrets" class="btn secondary">View on GitHub</a>
        </div>
    </section>

    <!-- Features -->
    <section class="features">
        <div class="feature">
            <h3>40+ Providers</h3>
            <p>AWS, Vault, Azure, GCP, and many more out of the box.</p>
        </div>
        <div class="feature">
            <h3>Kubernetes Native</h3>
            <p>Declarative CRDs with automatic reconciliation.</p>
        </div>
        <div class="feature">
            <h3>Bidirectional Sync</h3>
            <p>Pull secrets in, or push them out with PushSecret.</p>
        </div>
        <div class="feature">
            <h3>Dynamic Generators</h3>
            <p>Generate passwords, SSH keys, and registry credentials.</p>
        </div>
    </section>

    <!-- CNCF Badge -->
    <section class="cncf">
        <p>We are a <a href="https://www.cncf.io/">Cloud Native Computing Foundation</a> project.</p>
        <img src="/images/cncf-logo.svg" alt="CNCF">
    </section>

    <!-- Adopters -->
    <section class="adopters">
        <h2>Trusted By</h2>
        <div class="adopter-logos">
            <!-- Logos added here -->
        </div>
    </section>

    <!-- Footer -->
    <footer>
        <div class="social-links">
            <a href="https://kubernetes.slack.com/messages/external-secrets">Slack</a>
            <a href="https://github.com/external-secrets/external-secrets">GitHub</a>
            <a href="https://twitter.com/ExtSecretsOptr">Twitter</a>
        </div>
        <p>&copy; 2025 The external-secrets Authors. A CNCF Project.</p>
    </footer>
</body>
</html>
```


### Design Guidelines

#### Visual Style
- Clean, minimal design (like kured.dev)
- We could include search and Dark/light mode toggle (matching docs)
- ESO brand colors from existing logo
- Mobile-responsive (single column on mobile)

#### Developer experience
- No JavaScript frameworks knowledge required

### User Stories

**As a new visitor**, I want to understand what ESO does in 10 seconds, so I can decide if it's relevant to my needs.

**As an evaluator**, I want to see key features and who uses ESO, so I can assess project maturity.

**As a developer**, I want quick access to documentation, so I can start implementing.

**As a community member**, I want to find Slack/GitHub links easily, so I can get involved.

### API

No API changes. This is a website enhancement.

### Behavior

- `external-secrets.io/` shows new landing page
- `external-secrets.io/latest/` continues to work (documentation)
- All existing documentation URLs unchanged
- Old bookmarks to `/` will see landing page instead of redirect

### Drawbacks

- Less flexible for marketing pages
- mkdocs Material home page is still docs-oriented
- Harder to achieve a "kured.dev" simplicity

### Acceptance Criteria

- [ ] Landing page accessible at `external-secrets.io/`
- [ ] All 4 key features displayed
- [ ] Links to docs work (`/latest/`, `/latest/introduction/getting-started/`)
- [ ] Links to community (Slack, GitHub, Twitter) work
- [ ] Mobile responsive (tested on 375px width)
- [ ] Page weight < 100KB
- [ ] Dark/light mode toggle works
- [ ] CNCF badge displayed
- [ ] Adopter logos section (placeholder, content TBD)
- [ ] CI/CD builds and deploys both landing page and docs

**Rollout:**
1. PR with new page
2. Review design on preview deployment
3. Merge and deploy
4. Update DNS if needed (likely no change needed)

**Monitoring:**
- Existing analytics (Google Analytics in mkdocs config) extended to landing page
- Optional: add simple analytics to landing page

## Alternatives

### Alternative 1: Hugo Home Page

Use Hugo instead of Material's [custom home page](https://squidfunk.github.io/mkdocs-material/setup/setting-up-navigation/#custom-home-page)

**Implementation:**

#### Repository Structure

Add a `/website/` directory to the existing repository:

```
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

#### Build Pipeline

Update GitHub Actions to build both Hugo and mkdocs:

```yaml
# .github/workflows/publish.yml (addition)
jobs:
  build-website:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # Build Hugo landing page
      - name: Setup Hugo
        uses: peaceiris/actions-hugo@v2
        with:
          hugo-version: 'latest'

      - name: Build Landing Page
        run: hugo --source website --destination ../public

      # Build mkdocs docs
      - name: Setup Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.x'

      - name: Install mkdocs
        run: pip install mkdocs-material mike pymdown-extensions

      - name: Build Docs
        run: mkdocs build -f hack/api-docs/mkdocs.yml

      # Combine outputs
      # Hugo output at /public/
      # mkdocs output at /public/latest/ (via mike)

      - name: Deploy
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./public
```

**Pros:**
- Works out of the box
- Easy and fast to build
- Optimized images (SVG for logos)
- Target: < 100KB total page weight
- Inline CSS for faster response

**Cons:**
- **Two build systems**: Hugo + mkdocs adds complexity. At the same time, Hugo build is minimal (single page), well-documented in CI
- **Maintenance**: Another thing to keep updated. Landing page content is stable, rarely changes. Only need to bump hugo in our tooling.

### Alternative 2: Separate Repository

Create a new `external-secrets/website` repository for the landing page.

**Pros:**
- Complete separation of concerns
- Independent release cycle

**Cons:**
- Two repositories to maintain
- Harder to keep in sync
- More complex deployment coordination

**Decision:** Rejected - single repository is simpler for a single landing page, especially when using mkdocs.
If using hugo, that makes things simpler.

### Alternative 3: Subdomain for Docs

Use `docs.external-secrets.io` for documentation, `external-secrets.io` for landing.

**Pros:**
- Clean URL separation
- Independent deployments

**Cons:**
- Requires DNS changes
- Breaks existing documentation URLs (major)
- More infrastructure to manage

**Decision:** Rejected - breaking existing URLs is unacceptable.
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
