
The Azure Access Token generator mints a short-lived Microsoft Entra ID access token for a
configurable Entra **resource**. It is the generic building block for "give me an Entra
token for resource X" -- for example Azure DevOps, Azure Resource Manager, Azure Storage,
or any custom Entra application.

All supported authentication methods are app-only (client-credentials) flows, so the token
is always requested with the `<resource>/.default` scope. ESO rotates it on the
`refreshInterval` of the consuming `ExternalSecret`.

## Output Keys and Values

| Key   | Description                          |
| ----- | ------------------------------------ |
| token | the Microsoft Entra ID access token  |

## Resource

`spec.resource` is **required** and selects what the token grants access to. It is the Entra
resource id (a GUID or a resource URI). Examples:

| Resource                | `spec.resource`                        |
| ----------------------- | -------------------------------------- |
| Azure DevOps            | `499b84ac-1321-427f-aa17-267ca6975798` |
| Azure Resource Manager  | `https://management.azure.com`         |
| Azure Storage           | `https://storage.azure.com`            |

The token is requested with the `<resource>/.default` scope.

## Authentication

You must choose one out of three authentication mechanisms:

- service principal (client secret **or** client certificate)
- managed identity
- workload identity

The generated token inherits the permissions of the identity that mints it.

## Azure DevOps

Azure DevOps is the primary motivating use-case (see
[external-secrets#5113](https://github.com/external-secrets/external-secrets/issues/5113)).
Service principals and managed identities
[cannot mint Azure DevOps Personal Access Tokens (PATs)](https://learn.microsoft.com/en-us/azure/devops/integrate/get-started/authentication/service-principal-managed-identity),
but the Entra access token minted here works for git checkout, package feeds
(maven/npm/NuGet) and self-hosted agent registration. Set
`spec.resource: 499b84ac-1321-427f-aa17-267ca6975798` and remap the token to `AZP_TOKEN` in
the consuming `ExternalSecret` (see example below).

## Example Manifest

```yaml
{% include 'generator-azure.yaml' %}
```

Example `ExternalSecret` that references the generator and remaps the token to `AZP_TOKEN`,
the variable the Microsoft self-hosted Azure DevOps agent expects:

```yaml
{% include 'generator-azure-example.yaml' %}
```
