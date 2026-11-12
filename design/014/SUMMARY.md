## Summary

Separate Out of Tree Providers via dedicated Custom Resources for them.
Providers are deployed separately from core ESO, and reconcile a dedicated CRD group
for that provider. New Core-Related CRDs are added for ESO to know where a provider is,
and how to communicate with it.

Minimal manifests for the simple case are in [`examples/`](./examples/): a
`ClusterProviderClass`, a provider-owned `SecretsManager` configuration and a
`SecretStore` that points at both. `ExternalSecret` and `PushSecret` do not
change.

By moving provider configuration to a GVK Reference under `providerRef`, this allows
providers to not even be on the same Kubernetes Cluster as ESO - as long as they can
translate a `providerRef` GVK into some internal config that might live in memory,
in a database, in another kubernetes cluster, etc.

### Today: providers compiled into the controller

```mermaid
flowchart LR
    ES["ExternalSecret"]
    SS["SecretStore<br/>spec.provider.aws (inline)"]
    subgraph eso["ESO controller pod: one binary, every provider SDK"]
      CTRL["reconciler"]
      CM["client manager"]
      PROV["AWS provider (in process)"]
      end
    CRED["credential Secret / ServiceAccount"]
    TGT["target Secret"]
  AWS[("AWS Secrets Manager")]
  ES --> CTRL
  CTRL --> SS
  CTRL --> CM --> PROV
  PROV -- "reads" --> CRED
  PROV -- "SDK call" --> AWS
  CTRL -- "formats + templates" --> TGT
```

### Afterwards: providers as separate processes behind a gRPC contract

```mermaid
flowchart LR
  ES["ExternalSecret"] --> CORE["ESO controller"]
  CORE -- "reads" --> SS["SecretStore<br/>runtimeRef + providerRef"]
  SS -. "runtimeRef" .-> PC["ClusterProviderClass<br/>endpoint, tls, auth"]
  CORE == "gRPC over mTLS<br/>provider_ref + RequestContext" ==> PROV["provider-aws pod"]
  PROV -- "loads" --> CR["SecretsManager CR<br/>(provider-owned CRD)"]
  PROV -- "AWS SDK" --> AWS[("AWS Secrets Manager")]
  AWS -- "values" --> PROV
  PROV == "values" ==> CORE
  CORE -- "formats + templates" --> TGT["target Secret"]
```

### Who creates what, and what happens

```mermaid
sequenceDiagram
  autonumber
  actor Admin as Platform admin
  actor Tenant as Tenant (team-a)
  actor Dev as App developer (team-a)
  participant K8s as Kubernetes API
  participant Core as ESO controller
  participant Prov as provider-aws pod
  participant AWS as AWS Secrets Manager

  Admin->>K8s: helm install provider-aws: CRD, Deployment, Service, empty server Secret + Role, ClusterProviderClass aws
  Core->>Prov: Health + Describe over mTLS
  Prov-->>Core: serves SecretsManager, capabilities, protocol v1alpha1
  Core->>K8s: ClusterProviderClass aws status: Ready=True

  Tenant->>K8s: SecretsManager aws-prod in team-a (region, role, serviceAccountRef app-secrets)
  Tenant->>K8s: SecretStore aws-prod in team-a (runtimeRef aws, providerRef aws-prod)
  Core->>K8s: resolve class and scope rules (CEL already accepted the objects)
  Core->>Prov: Validate(provider_ref team-a/aws-prod, context credential_namespace=team-a)
  Prov->>K8s: get SecretsManager team-a/aws-prod, token for team-a/app-secrets
  Prov->>AWS: probe credentials
  Prov-->>Core: VALID, Resolved credential_namespaces=[team-a]
  Core->>K8s: SecretStore aws-prod status: Ready=True, capabilities ReadWrite

  Dev->>K8s: ExternalSecret db in team-a (secretStoreRef aws-prod)
  Core->>Prov: GetSecret(key /team-a/prod/db, provider_ref, context request=ExternalSecret db, caller_namespace=team-a)
  Prov->>Prov: optional authorization on RequestContext, cached client for (CR uid, generation, team-a)
  Prov->>AWS: GetSecretValue
  AWS-->>Prov: value
  Prov-->>Core: value, Resolved
  Core->>Core: verify Resolved, format, template
  Core->>K8s: create or update target Secret db in team-a
```
