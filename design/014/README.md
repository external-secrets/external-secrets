# Out of Process Providers

This design document is split into separate sub-documents. Each sub-document analyzes different approaches and discusses trade-offs.

## Overview

- [API.md](./API.md) - API Design for Out-of-Process Providers
- [mTLS.md](./mTLS.md) - mTLS & Service Discovery for Out-of-Process Providers
- [ADAPTER.md](./ADAPTER.md) - Out-of-Process Provider Adapter Pattern
- [PROVIDER_AUTOGEN.md](./PROVIDER_AUTOGEN.md) - Provider Code Generation
- [TESTING.md](./TESTING.md) - V2 Provider Testing Strategy
- [THREAT_MODEL.md](./THREAT_MODEL.md) - Threat Model for Out-of-Process Providers

## Scope

This document covers the technical aspects of out-of-process providers. 

Multi-repository structure, user migration and graduation criteria are out of scope and will be covered in a separate design document.

## Problem Statements

For detailed discussion, see: [recording](https://zoom.us/rec/share/fioR9a-blopYRqALdvphBU2rXN-To3BAuMOZK1FdpxUIIe31Qam3oWDxGWFHKTT9.UmkUa-3W5abZww8-), [meeting notes](https://zoom-lfx.platform.linuxfoundation.org/meeting/95703312466-1761822000000/summaries?password=47fac69c-506a-408d-a424-ff3c1c69a0dc), [drawings](https://excalidraw.com/#room=bf94104fddcd05c32c20,BTAv5naTXFc-6m0275D0_g), and [this previous PR](https://github.com/external-secrets/external-secrets/pull/4792). There's also an [old issue](https://github.com/external-secrets/external-secrets/issues/696) with more research and history, although outdated.

#### Risk of Supply-Chain Attacks

ESO bundles many dependencies, creating a large attack surface. We need to reduce this surface area to improve security.

#### Users want to maintain their own providers without merging into core ESO

ESO provides limited support for providers built and maintained outside the ESO domain. Enterprises with custom APIs or workflows for managing secrets cannot integrate them effectively. Currently, only the webhook provider supports such integrations, but it lacks features like `GetAllSecrets()` and advanced authentication methods such as OAuth flows.

#### Independence of Provider Maintainers

Provider maintainers need the freedom to change their APIs. This is not possible today because everything is bundled in ESO. The GA SecretStore CRD cannot be changed in breaking ways.

#### The ESO core team does not want to maintain everything

With approximately 40 providers bundled in ESO, maintaining them has become a burden. We want to shift responsibility to the community for provider maintenance.

## Requirements

#### A Provider must run in a separate Pod

Each provider must run in a separate Pod to enable network isolation. When users run multiple providers, each can have its own network policy. This enforces the principle of least privilege at the network level. Core ESO does not need access to anything outside the cluster. All secrets flow through provider pods.

Core ESO and providers must run with different RBAC permissions. We discourage sidecar deployments. ESO must provide a secure-by-default deployment that allows users to apply NetworkPolicies post-deployment and ensures that ESO Core and providers run with separate RBAC permissions to enforce least privilege.

#### A Provider must be built with a minimum set of dependencies

A provider binary should typically import only the ESO `/api` package and the provider-specific SDK. This keeps providers lightweight, easy to audit, and easy to keep up to date.

#### ESO Core must be built without provider dependencies

ESO Core must be built without provider dependencies. This reduces the attack surface by removing unneeded dependencies.

#### Provider code must stay coherent

We must not fork or branch the existing provider source code for the out-of-process provider implementation. Provider code must remain coherent to avoid the maintenance burden of keeping multiple versions in sync.
