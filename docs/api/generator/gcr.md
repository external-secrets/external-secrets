GCRAccessToken creates a GCP Access token that can be used to authenticate with GCR in order to pull OCI images.

You must specify the `spec.projectID` in which GCR is located.

## Output Keys and Values

| Key        | Description                                                               |
| ---------- | ------------------------------------------------------------------------- |
| username   | username for the `docker login` command.                                  |
| password   | password for the `docker login` command.                                  |
| expiry     | time when token expires in UNIX time (seconds since January 1, 1970 UTC). |

## Authentication

### Workload Identity

Use `spec.auth.workloadIdentity` to point to a Service Account that has Workload Identity enabled.
For details see [GCP Secret Manager](../provider/google-secrets-manager.md#authentication).


### GCP Service Account

Use `spec.auth.secretRef` to point to a Secret that contains a GCP Service Account.
For details see [GCP Secret Manager](../provider/google-secrets-manager.md#authentication).

## Example Manifest

```yaml
{% include 'generator-gcr.yaml' %}
```
