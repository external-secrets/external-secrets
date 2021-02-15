!!! note "Coming soon"
    This is in the works.

## Multi Tenancy options
External Secrets Operator provides different modes of operation to fulfill ogranizational needs. This Guide outlines the flexibility of ESO and should give you a first impression of how you could employ this operator in your organization.

For a multi-tenant deployment you should first examine

1. what roles (i.e. *Application Developers*, *Cluster Admins*, ...) do you have in your organization,
2. what responsibilities do they have and
3. how does that map to Kubernetes RBAC roles.

Further, you should examine how your external API provider manages access-control for your secrets. Can you limit access by secret names (e.g. `db/dev/*`)? Or only on a bucket (a bunch of secrets) level?

**Note:** Please keep in mind that not all external APIs provide fine-grained access management for secrets.

### Shared Managed ClusterSecretStore

### Managed SecretStore per namespace

### SecretStore self-service
