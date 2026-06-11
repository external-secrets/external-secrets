# Documentation Migration Guide: Diataxis Model Assessment

This document provides a comprehensive analysis of migrating the External Secrets Operator (ESO) documentation to the [Diataxis](https://diataxis.fr/) documentation framework.

## Table of Contents

1. [Complete Page Assessment](#complete-page-assessment)
2. [Diataxis Suitability Analysis](#diataxis-suitability-analysis)
3. [Migration Approaches](#migration-approaches)
4. [Time Estimation](#time-estimation)

---

## Complete Page Assessment

### Legend

**Diataxis Fit:**
- Perfect - Already matches a Diataxis quadrant
- Adapt - Needs restructuring but content is good
- Split - Should be split into multiple pages
- Rewrite - Needs significant rewriting

**Content Type:**
- **T** = Tutorial (learning-oriented)
- **H** = How-to (task-oriented)
- **E** = Explanation (understanding-oriented)
- **R** = Reference (information-oriented)
- **M** = Mixed (multiple types interleaved)

---

### Introduction Section (8 files)

| File | Diataxis Fit | Type | Adaptation Needed |
|------|--------------|------|-------------------|
| `introduction/overview.md` | Perfect | E | None - excellent architecture explanation. Might be worth splitting. |
| `introduction/getting-started.md` | Adapt | T/H | Split: Tutorial for first-time setup, How-to for subsequent installs |
| `introduction/faq.md` | Adapt | R/H | Keep as reference, extract common tasks to how-to guides |
| `introduction/glossary.md` | Perfect | R | None - clean reference format |
| `introduction/prerequisites.md` | Adapt | T | Rename to tutorial, add learning objectives. Maybe use contributing guide instead.|
| `introduction/deprecation-policy.md` | Perfect | R | None - policy reference is appropriate, but should be refactored (see example PR) |
| `introduction/stability-support.md` | Perfect | R | None - version matrix is pure reference. Those could be improved, optimized for search, and more readable, but the content is clear.|

---

### API Documentation (27 files)

#### Core API (7 files)

| File | Diataxis Fit | Type | Adaptation Needed |
|------|--------------|------|-------------------|
| `api/spec.md` | Perfect | R | None - OpenAPI spec is pure reference |
| `api/externalsecret.md` | Split | M | Split: Reference (fields) + How-to (refresh, templating) |
| `api/secretstore.md` | Adapt | R/H | Extract provider setup to how-to, keep API reference |
| `api/clustersecretstore.md` | Adapt | R/H | Similar to secretstore.md |
| `api/clusterexternalsecret.md` | Adapt | R | Add more field descriptions |
| `api/clusterpushsecret.md` | Adapt | R | Add more field descriptions |
| `api/pushsecret.md` | Split | M | Split: Reference (API) + How-to (sync to provider) |

#### API Support (4 files)

| File | Diataxis Fit | Type | Adaptation Needed |
|------|--------------|------|-------------------|
| `api/components.md` | Perfect | E | None - architecture explanation is clear. Maybe merge with overview?|
| `api/metrics.md` | Split | R/H | Split: Reference (metric list) + How-to (Grafana setup, alerting). Will depend on metric refactor|
| `api/selectable-fields.md` | Perfect | R | None - field reference. Though the examples could probably be split into how-to, with clearer goals. |
| `api/controller-options.md` | Perfect | R | None - configuration reference |

#### Generators (16 files)

All of these could be moved into each _provider_ implementation.
They almost all need adapting, because they are a mix between Reference and How-to, so they address different needs of different persona.

| File | Diataxis Fit | Type | Adaptation Needed |
|------|--------------|------|-------------------|
| `api/generator/index.md` | Adapt | R | Add overview explanation of generators |
| `api/generator/acr.md` | Adapt | H/R | Restructure: Prerequisites → Steps → Reference |
| `api/generator/ecr.md` | Adapt | H/R | Same as acr.md |
| `api/generator/gcr.md` | Adapt | H/R | Same as acr.md |
| `api/generator/quay.md` | Adapt | H/R | Same as acr.md |
| `api/generator/cloudsmith.md` | Adapt | H/R | Same as acr.md |
| `api/generator/github.md` | Adapt | H/R | Same as acr.md |
| `api/generator/vault.md` | Adapt | H/R | Same as acr.md |
| `api/generator/password.md` | Perfect | R | None - specification reference |
| `api/generator/uuid.md` | Perfect | R | None - specification reference |
| `api/generator/sshkey.md` | Perfect | R | None - specification reference |
| `api/generator/mfa.md` | Perfect | R | None - specification reference |
| `api/generator/sts.md` | Adapt | H/R | Same as acr.md |
| `api/generator/cluster.md` | Adapt | R | Add examples |
| `api/generator/fake.md` | Perfect | R | None - test utility reference |
| `api/generator/webhook.md` | Adapt | H/R | Same as acr.md |

---

### Guides Section (18 files)

Most of the guides should either be How-tos or Explanation. 
We should avoid landing pages, as we will have each section's introduction for that.

Some pages here are still reference to explain the machinery and could be moved.

| File | Diataxis Fit | Type | Adaptation Needed |
|------|--------------|------|-------------------|
| `guides/introduction.md` | Adapt | R | Convert to navigation page with categories |
| `guides/templating.md` | Split | T/R | Split: Tutorial (basics) + Reference (functions) + How-to (patterns) |
| `guides/templating-v1.md` | Rewrite | R | Mark deprecated, add migration path |
| `guides/pushsecrets.md` | Perfect | H | None - clear task-oriented guide. Though it could be split in two. |
| `guides/generator.md` | Split | E/H | Split: Explanation (concepts) + How-to (usage) |
| `guides/multi-tenancy.md` | Perfect | E | None - excellent pattern explanation |
| `guides/security-best-practices.md` | Perfect | H | None - clear security checklist. Hard to read though, maybe worth a separate section? |
| `guides/getallsecrets.md` | Perfect | H | None - task-focused with examples |
| `guides/all-keys-one-secret.md` | Perfect | H | None - specific task guide |
| `guides/common-k8s-secret-types.md` | Perfect | H | None - pattern cookbook |
| `guides/targeting-custom-resources.md` | Adapt | H | Add alpha warning prominently |
| `guides/ownership-deletion-policy.md` | Split | R/H | Split: Reference (policy matrix) + How-to (choosing policy) |
| `guides/controller-class.md` | Adapt | H | Add use case explanation |
| `guides/decoding-strategy.md` | Perfect | R/H | None - clear with examples |
| `guides/datafrom-rewrite.md` | Adapt | H | Add more context on when to use |
| `guides/threat-model.md` | Perfect | E | None - comprehensive security analysis |
| `guides/using-esoctl-tool.md` | Perfect | H | None - tool usage guide |
| `guides/disable-cluster-features.md` | Perfect | H | None - simple config guide |
| `guides/v1beta1.md` | Rewrite | R | Needs migration guide format |
| `guides/using-latest-image.md` | Adapt | H | Add warning about instability |

---

### Provider Documentation

I am not yet sure how to standardize structure here.
I know it's necessary, I feel we need a _reference_ per provider.

| File | Diataxis Fit | Type | Adaptation Needed |
|------|--------------|------|-------------------|
| `provider/aws-secrets-manager.md` | Split | M | Split: How-to (setup) + Reference (IAM policies) + Troubleshooting |
| `provider/aws-parameter-store.md` | Split | M | Same as aws-secrets-manager.md |
| `provider/azure-key-vault.md` | Split | M | Same pattern - needs auth methods separation |
| `provider/google-secrets-manager.md` | Split | M | Same pattern - WIF vs SA key separation |
| `provider/hashicorp-vault.md` | Split | M | Same pattern - multiple auth methods |
| `provider/ibm-secrets-manager.md` | Adapt | H/R | Standardize structure |
| `provider/1password-automation.md` | Adapt | H | Add prerequisites section |
| `provider/1password-sdk.md` | Adapt | H | Add prerequisites section |
| `provider/akeyless.md` | Adapt | H/R | Standardize structure |
| `provider/conjur.md` | Adapt | H/R | Standardize structure |
| `provider/keeper-security.md` | Adapt | H/R | Standardize structure |
| `provider/passbolt.md` | Adapt | H/R | Standardize structure |
| `provider/bitwarden-secrets-manager.md` | Adapt | H/R | Standardize structure |
| `provider/infisical.md` | Adapt | H/R | Standardize structure |
| `provider/delinea.md` | Adapt | H/R | Standardize structure |
| `provider/secretserver.md` | Adapt | H/R | Standardize structure |
| `provider/beyondtrust.md` | Adapt | H/R | Standardize structure |
| `provider/pulumi.md` | Adapt | H/R | Standardize structure |
| `provider/webhook.md` | Split | M | Complex - needs tutorial + reference |
| `provider/kubernetes.md` | Adapt | H/R | Standardize structure |
| `provider/gitlab-variables.md` | Adapt | H/R | Standardize structure |
| `provider/github.md` | Adapt | H/R | Standardize structure |
| `provider/doppler.md` | Adapt | H/R | Standardize structure |
| `provider/barbican.md` | Adapt | H/R | Standardize structure |
| `provider/ngrok.md` | Adapt | H/R | Standardize structure |
| `provider/chef.md` | Adapt | H/R | Standardize structure |
| `provider/fortanix.md` | Adapt | H/R | Standardize structure |
| `provider/scaleway.md` | Adapt | H/R | Standardize structure |
| `provider/alibaba.md` | Rewrite | H | Mark deprecated clearly, add migration |
| `provider/oracle-vault.md` | Adapt | H/R | Standardize structure |
| `provider/yandex-lockbox.md` | Adapt | H/R | Standardize structure |
| `provider/yandex-certificate-manager.md` | Adapt | H/R | Standardize structure |
| `provider/volcengine.md` | Adapt | H/R | Standardize structure |
| `provider/fake.md` | Perfect | R | None - test utility reference |
| `provider/onboardbase.md` | Adapt | H/R | Standardize structure |
| `provider/previder.md` | Adapt | H/R | Standardize structure |
| `provider/cloudru.md` | Adapt | H/R | Standardize structure |
| `provider/cloak.md` | Rewrite | H | Unclear status, needs verification |
| `provider/device42.md` | Rewrite | H | Mark unmaintained, add alternatives |
| `provider/senhasegura-dsm.md` | Adapt | H/R | Standardize structure |
| `provider/openbao.md` | Adapt | H/R | Standardize structure |

---

### Contributing Section (8 files)

See all the changes in contributing guide PR for example.

---

### Examples Section (4 files)

If possible, migrate some of the work to a provider specific sub-page.

| File | Diataxis Fit | Type | Adaptation Needed |
|------|--------------|------|-------------------|
| `examples/gitops-using-fluxcd.md` | Perfect | T | None - excellent end-to-end tutorial |
| `examples/bitwarden.md` | Perfect | T | None - complete integration tutorial |
| `examples/jenkins-kubernetes-credentials.md` | Adapt | T | Update and verify still works |
| `examples/anchore-engine-credentials.md` | Adapt | T | Update and verify still works |

---

### Root Level Files

They can be moved to our "Marketing" landing page.

| File | Diataxis Fit | Type | Adaptation Needed |
|------|--------------|------|-------------------|
| `index.md` | adapt | - | make it our marketing landing page |
| `eso-blogs.md` | perfect | - | add it to our new landing page |
| `eso-demos.md` | perfect | - | add it to our new landing page |
| `eso-talks.md` | perfect | - | add it to our new landing page |
| `eso-tools.md` | perfect | - | add it to our new landing page |
| `provider-passworddepot.md` | rewrite | R | move to provider/ directory |

---

## Diataxis suitability analysis

### why diataxis is a good fit for eso documentation

| strength | evidence |
|----------|----------|
| **clear user personas** | eso has distinct users: newcomers (need tutorials), operators (need how-tos), architects (need explanations), developers (need reference) |
| **task-oriented nature** | most eso usage is task-based: "connect to aws", "rotate secrets", "set up multi-tenancy" |
| **complex api surface** | 40+ providers, multiple crds - needs clear reference separation |
| **existing structure** | already partially organized (guides/, api/, provider/) - migration not starting from scratch |
| **active community** | clear contribution structure supports maintaining separated docs |

### challenges with pure diataxis

| challenge | mitigation |
|-----------|------------|
| **provider docs are inherently mixed** | use a standardized template with clear sections |
| **some content spans quadrants** | accept some mixing within pages, use clear headers |
| **navigation complexity** | implement good cross-linking and a clear landing page |
| **40+ providers = potential file explosion** | standardized templates rather than strict splitting |

---

## migration approaches

### approach 1: simplified diataxis

**philosophy:** four main sections but less strict separation within pages. pragmatic over pure.

```text
docs/
├── learn/           # tutorials + explanations combined
│   ├── getting-started.md
│   ├── architecture.md
│   ├── security-model.md
│   └── examples/
├── guides/          # how-to guides (task-focused)
│   ├── authentication/
│   ├── templating/
│   ├── multi-tenancy/
│   └── troubleshooting/
├── reference/       # api docs, provider reference, configuration
│   ├── api/
│   ├── providers/
│   └── configuration/
```

**characteristics:**
- combines tutorials and explanations into "learn" (reduces navigation)
- allows mixed content within provider pages
- fewer directories to navigate
- easier for contributors to place new content

| pros | cons |
|------|------|
| simpler structure | less precise content categorization |
| fewer files to maintain | "learn" section may grow large |
| easier for contributors | harder to find specific content types |
| quick to implement | not "true" diataxis |

My opinion is that learn will become a big mess that will mix the _concepts_ learning, _follow a lesson_ or _learn by doing_.

**effort:** 4-5 weeks

---

### approach 2: strict diataxis

**philosophy:** pure four-quadrant separation. every page belongs to exactly one quadrant.

```text
docs/
├── tutorials/           # learning-oriented (step-by-step lessons)
│   ├── getting-started.md
│   ├── first-external-secret.md
│   ├── first-pushsecret.md
│   ├── gitops-integration.md
│   └── providers/
│       ├── aws-tutorial.md
│       ├── vault-tutorial.md
│       └── ...
├── how-to/              # task-oriented (goal-focused recipes)
│   ├── authentication/
│   │   ├── aws-iam-role.md
│   │   ├── aws-static-credentials.md
│   │   ├── vault-approle.md
│   │   └── ...
│   ├── templating/
│   ├── multi-tenancy/
│   ├── monitoring/
│   └── troubleshooting/
├── explanation/         # understanding-oriented (concepts)
│   ├── architecture.md
│   ├── security-model.md
│   ├── lifecycle-policies.md
│   ├── provider-capabilities.md
│   └── threat-model.md
├── reference/           # information-oriented (facts)
│   ├── api/
│   │   ├── externalsecret.md
│   │   ├── secretstore.md
│   │   └── ...
│   ├── providers/
│   │   ├── aws-secrets-manager.md    # config reference only
│   │   ├── vault.md
│   │   └── ...
│   ├── generators/
│   ├── cli/
│   └── configuration/
```

**characteristics:**
- every page fits exactly one quadrant
- provider docs split into: tutorial (tutorials/providers/), how-to (how-to/authentication/), reference (reference/providers/)
- clear navigation by content type
- requires significant restructuring

| pros | cons |
|------|------|
| pure diataxis benefits | many more files (~160+ for providers alone) |
| very clear navigation | harder for contributors |
| easy to find content type | content duplication risk |
| good for large teams | higher maintenance burden |

All the providers will be split into multiple pages. It will help, but it is getting more complex to write, at the expense of one top folder.

**effort:** 10-12 weeks

---

### approach 3: mixed (recommended)

**philosophy:** diataxis for core docs, standardized template for providers. best of both worlds.

```text
docs/
├── tutorials/           # learning-oriented (diataxis)
│   ├── getting-started.md
│   ├── first-external-secret.md
│   └── gitops-integration.md
├── how-to/              # task-oriented (diataxis)
│   ├── authentication/
│   ├── templating/
│   ├── multi-tenancy/
│   └── troubleshooting/
├── explanation/         # understanding-oriented (diataxis)
│   ├── architecture.md
│   ├── security-model.md
│   └── lifecycle-policies.md
├── reference/           # information-oriented (diataxis)
│   ├── api/
│   │   ├── externalsecret.md
│   │   ├── secretstore.md
│   │   ├── generators/      # generators under api reference
│   │   └── ...
│   ├── cli/
│   └── configuration/
├── providers/           # standardized template (not strict diataxis)
│   ├── _template.md     # template for contributors
│   ├── aws-secrets-manager.md
│   ├── azure-key-vault.md
│   ├── hashicorp-vault.md
│   └── ...
```

This is an LLM generated template: We need to do it, and see what is best over time.
If possible, the _reference_ of a provider should be generated from the features.

**provider template structure:**
```markdown
# provider name

> **status:** maintained | not maintained | deprecated
> **capabilities:** read | write | readwrite

## overview
[brief explanation - what and when to use]

## prerequisites
[requirements and provider-side setup links]

## authentication
### method 1: [e.g., iam role]
[step-by-step with examples]

### method 2: [e.g., static credentials]
[step-by-step with examples]

## configuration reference
[field table with types and descriptions]

## examples
[working yaml examples]

## troubleshooting
[common issues and solutions]

## see also
[related guides and external docs]
```

**characteristics:**
- core docs follow strict diataxis (tutorials, how-to, explanation, reference)
- providers use standardized template with clear internal sections
- generators live under reference/api/ (not top-level, if possible in each provider)
- single file per provider (manageable)
- clear contributor guidance via template

| pros | cons |
|------|------|
| diataxis benefits for core docs | providers not "pure" diataxis |
| manageable provider docs (41 files, not 160+) | requires template discipline |
| clear navigation | mixed philosophy |
| scalable for new providers | |
| good contributor experience | |

**effort:** 6-8 weeks

---
## time estimation

### for mixed approach (recommended)

#### phase 1: foundation (week 1-2)

| task | effort | dependencies |
|------|--------|--------------|
| create diataxis landing page structure | 2 days | none |
| define provider documentation template | 1 day | none |
| set up navigation/sidebar restructure | 2 days | landing page |
| create style guide for contributors | 1 day | templates |
| set up redirects infrastructure | 1 day | none |

#### phase 2: core documentation (week 3-4)

| task | effort | dependencies |
|------|--------|--------------|
| create tutorials/ section | 2 days | foundation |
| reorganize how-to/ section | 2 days | foundation |
| create explanation/ section | 2 days | foundation |
| reorganize reference/ section (including generators) | 3 days | foundation |

#### phase 3: provider documentation (week 5-7)

| task | effort | dependencies |
|------|--------|--------------|
| migrate major providers (aws, gcp, azure, vault) | 4 days | template |
| migrate enterprise providers (11 files) | 3 days | template |
| migrate specialized providers (11 files) | 2 days | template |
| migrate remaining providers (15 files) | 2 days | template |
| mark deprecated/unmaintained providers | 1 day | all providers |

#### phase 4: polish and validation (week 8)

| task | effort | dependencies |
|------|--------|--------------|
| cross-link audit | 2 days | all content |
| navigation testing | 1 day | cross-links |
| community review | 2 days | navigation |
| final adjustments | 1 day | review |

---

## conclusion

This proposal recommends to adopt the **mixed approach** to diataxis which combines:
- diataxis structure for core documentation (tutorials, how-to, explanation, reference)
- standardized templates for providers
- generators at first citizen components like other api, under reference/api/ (not top-level)
- clear navigation with cross-linking

this approach provides:
- clear content organization
- manageable maintenance burden
- good contributor experience
- scalable for future providers
- 6-8 week realistic timeline
