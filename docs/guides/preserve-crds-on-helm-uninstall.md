# Preserving CRDs when uninstalling the Helm chart

The External Secrets Operator chart templates its CRDs like any other chart
resource. This means Helm owns their lifecycle: when you run
`helm uninstall`, Helm deletes the CRDs along with the rest of the release.

Deleting a CRD also deletes every custom resource of that type in the
cluster — every `ExternalSecret`, `SecretStore`, `ClusterSecretStore`,
`PushSecret`, and generator resource — including resources in namespaces the
release itself does not touch. Depending on your policies, the Kubernetes
`Secret` objects those resources manage can be removed as well.

!!! note
    This is standard Helm behavior for *templated* CRDs, on both Helm 3 and
    Helm 4. It differs from charts that ship static CRD files in a `crds/`
    directory: Helm never deletes those, but it also never upgrades them.
    ESO templates its CRDs precisely so that upgrades keep them current.

If losing the CRDs and their resources on uninstall is not acceptable for
your environment, you have two options.

## Option 1: Keep the CRDs on uninstall

Helm skips resources annotated with `helm.sh/resource-policy: keep` when a
release is uninstalled. The chart exposes annotations for its CRDs through
the `crds.annotations` value:

```yaml
# values.yaml
crds:
  annotations:
    helm.sh/resource-policy: keep
```

or, on the command line:

```console
helm install external-secrets external-secrets/external-secrets \
  --set 'crds.annotations.helm\.sh/resource-policy=keep'
```

The annotation must be present on the CRDs *before* you uninstall, so set it
when you install or upgrade the release — adding it afterwards is too late
if the release is already gone.

With the annotation in place, `helm uninstall` removes the operator but
leaves the CRDs, their custom resources, and the synced `Secret` objects in
the cluster.

## Option 2: Manage the CRDs separately

If you want the CRDs to have a lifecycle that is fully independent of the
chart, install the chart without them and manage the CRDs with your own
tooling (for example, by applying the CRD manifests from the release assets,
or through your GitOps setup):

```console
helm install external-secrets external-secrets/external-secrets \
  --set installCRDs=false
```

In this setup `helm uninstall` can never delete the CRDs, because Helm never
managed them in the first place. You are then responsible for keeping the
CRDs in sync with the operator version you run.

!!! warning
    Preserving CRDs is a deliberate trade-off. Environments that rely on
    `helm uninstall` for a clean slate — for example ephemeral CI clusters
    that reinstall ESO between test runs — will keep the old CRDs and
    custom resources behind, so the next install does not start clean.
    Choose the option that matches how you operate the cluster.
