# Security Best Practices

The purpose of this document is to provide a set of best practices for securing External Secrets Operator. These best practices are designed to reduce the risk of a successful attack against the operator or the Kubernetes cluster it integrates with.

## Pod Security

The External Secrets Operator Pods have been configured to adhere to [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/) `restricted` profile which establish a great security posture by following current Pod hardening best practices such as the [NSA Kubernetes Hardening Guide](https://media.defense.gov/2022/Aug/29/2003066362/-1/-1/0/CTR_KUBERNETES_HARDENING_GUIDANCE_1.2_20220829.PDF).

These measures provide a secure and robust environment for the External Secrets Operator to operate within. They have been set as default since `v0.8.2` and should not be changed to conform with the principle of least privilege.

## RBAC

The External Secrets Operator runs with highly privileged access to your Kubernetes cluster. Due to it's purpose it is able to read and write to all secrets across all namespaces.

Make sure to restrict access to ESO resources like `ExternalSecret`, `SecretStore` etc. where appropriate. This is particularly important for cluster-scoped resources like `ClusterExternalSecret` and `ClusterSecretStore`. If an attacker is able to tamper with these resources he could potentially get unauthorized access to secrets or may be able to exfiltrate data out of your system.

In most scenarios the External Secrets Operator runs cluster-wide. If this is not desired and you want to run it per-namespace, you can scope it to a particular namespace, see `scopedRBAC` and `scopedNamespace` in the helm chart.

A short checklist for you to walk through:

* restrict access to execute a shell `pods/exec` in External Secrets Operator Pod
* restrict access to `(Cluster)ExternalSecret` and `(Cluster)SecretStore`
* restrict access to [aggregated ClusterRoles](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#aggregated-clusterroles) `view/edit/admin` where needed
* run ESO with scoped RBAC/Namespace if needed


## Network Policy

Network Traffic from/to the operator should be restricted using `NetworkPolicies` or similar. By default, External Secrets Operator does not provide Network Policies for you.


!!! danger "Data Exfiltration Risk"

    If not configured properly ESO may be used to exfiltrate data out of your cluster.
    It is advised to create tight NetworkPolicies and use a policy engine such as kyverno to prevent data exfiltration.


You should restrict access in **egress** direction:

* controller
    * kube-apiserver
    * secret provider (AWS, GCP, ...), where possible use private endpoints.
* webhook
    * kube-apiserver
* cert-controller
    * kube-apiserver

Further, you also should restrict **ingress** traffic to ESO Pods:

* controller
    * `:8080` from you monitoring agent
* cert-controller:
    * `:8080` from you monitoring agent
    * `:8081` from kubelet (healthz/readyz)
* webhook:
    * `:10250` from the kube-apiserver
    * `:8080` from you monitoring agent
    * `:8081` from kubelet (healthz/readyz)

## Policy Engine Best Practices

You should use a policy engine like [kyverno](http://kyverno.io/) or [OPA Gatekeeper](https://github.com/open-policy-agent/gatekeeper) to restrict changes made to ESO resources like `SecretStore` and `ExternalSecret`.

!!! danger "Data Exfiltration Risk"

    ESO could be used to exfiltrate data out of your cluster. You should disable all providers you don't need.
    Further, you should implement `NetworkPolicies` to restrict network access to known entities (see above), to prevent data exfiltration.


Here a couple of recommendations for you to consider:

* explicitly deny usage of the providers you don't need
* restrict access to secrets with/without a particular prefix in `.spec.data[].remoteRef.key`
* restrict usage of a `(Cluster)SecretStore` reference in an ExternalSecret


!!! note "Provider Validation Example Policy"

    The following policy validates the usage of the `provider` field in the SecretStore manifest.

    ```yaml
    {% filter indent(width=4) %}
{% include 'kyverno-policy-secretstore.yaml' %}
    {% endfilter %}
    ```

## Regular Patches

Regularly patch and update all software components of ESO and the cluster to ensure that known vulnerabilities are addressed. Use automated patching and updating tools to ensure that all components are kept up-to-date.

We provide regular updates to ESO, see [Stability and Support](../../introduction/stability-support.md).

## Verify Artefacts

### Verify Container Images

External Secrets Operator container images are signed using Cosign and the keyless signing feature. To verify the container image, follow the steps below.

```sh
$ crane digest ghcr.io/external-secrets/external-secrets:v0.8.1
sha256:36e606279dbebac51b4b9300b9fa85e8c08c1c673ba3ecc38af1402a0b035554

$ COSIGN_EXPERIMENTAL=1 cosign verify ghcr.io/external-secrets/external-secrets@sha256:36e606279dbebac51b4b9300b9fa85e8c08c1c673ba3ecc38af1402a0b035554 | jq

# ...
[
  {
    "critical": {
      "identity": {
        "docker-reference": "ghcr.io/external-secrets/external-secrets"
      },
      "image": {
        "docker-manifest-digest": "sha256:36e606279dbebac51b4b9300b9fa85e8c08c1c673ba3ecc38af1402a0b035554"
      },
      "type": "cosign container image signature"
    },
    "optional": {
      "1.3.6.1.4.1.57264.1.1": "https://token.actions.githubusercontent.com",
      "1.3.6.1.4.1.57264.1.2": "workflow_dispatch",
      "1.3.6.1.4.1.57264.1.3": "a0d2aef2e35c259c9ee75d65f7587e6ed71ef2ad",
      "1.3.6.1.4.1.57264.1.4": "Create Release",
      "1.3.6.1.4.1.57264.1.5": "external-secrets/external-secrets",
      "1.3.6.1.4.1.57264.1.6": "refs/heads/main",
      "Bundle": {
        # ...
      },
      "GITHUB_ACTOR": "gusfcarvalho",
      "Issuer": "https://token.actions.githubusercontent.com",
      "Subject": "https://github.com/external-secrets/external-secrets/.github/workflows/release.yml@refs/heads/main",
      "githubWorkflowName": "Create Release",
      "githubWorkflowRef": "refs/heads/main",
      "githubWorkflowRepository": "external-secrets/external-secrets",
      "githubWorkflowSha": "a0d2aef2e35c259c9ee75d65f7587e6ed71ef2ad",
      "githubWorkflowTrigger": "workflow_dispatch"
    }
  }
]
```

Note that the important fields to verify in the output are `optional.Issuer` and `optional.Subject`. If Issuer and Subject do not match the values shown above, the image is not legit and should not be used.


### Verify Provenance

External Secrets Operator creates and attests to the provenance of its builds using the [SLSA standard](https://slsa.dev/provenance/v0.1). The attested provenance may be verified using the cosign tool.

```sh
$ COSIGN_EXPERIMENTAL=1 cosign verify-attestation --type slsaprovenance ghcr.io/external-secrets/external-secrets:v0.8.1 | jq .payload -r | base64 --decode | jq

Verification for ghcr.io/external-secrets/external-secrets:v0.8.1 --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - Any certificates were verified against the Fulcio roots.
Certificate subject:  https://github.com/external-secrets/external-secrets/.github/workflows/release.yml@refs/heads/main
Certificate issuer URL:  https://token.actions.githubusercontent.com
GitHub Workflow Trigger: workflow_dispatch
GitHub Workflow SHA: a0d2aef2e35c259c9ee75d65f7587e6ed71ef2ad
GitHub Workflow Name: Create Release
GitHub Workflow Trigger external-secrets/external-secrets
GitHub Workflow Ref: refs/heads/main
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "https://slsa.dev/provenance/v0.2",
  "subject": [
    {
      "name": "ghcr.io/external-secrets/external-secrets",
      "digest": {
        "sha256": "36e606279dbebac51b4b9300b9fa85e8c08c1c673ba3ecc38af1402a0b035554"
      }
    }
  ],
  "predicate": {
    "builder": {
      "id": "https://github.com/external-secrets/external-secrets/Attestations/GitHubHostedActions@v1"
    },
    "buildType": "https://github.com/Attestations/GitHubActionsWorkflow@v1",
    "invocation": {
      "configSource": {
        "uri": "git+https://github.com/external-secrets/external-secrets",
        "digest": {
          "sha1": "a0d2aef2e35c259c9ee75d65f7587e6ed71ef2ad"
        },
        "entryPoint": "Create Release"
      },
      "parameters": {
        "version": "v0.8.1"
      }
    },
    [...]
  }
}
```

### Fetching SBOM

An SBOM (Software Bill of Materials) in Software Package Data Exchange (SPDX) JSON format is attached to every External Secrets Operator image.  To download and verify the SBOM for a specific version, install Cosign and run:

```sh
$ crane digest ghcr.io/external-secrets/external-secrets:v0.8.1
sha256:36e606279dbebac51b4b9300b9fa85e8c08c1c673ba3ecc38af1402a0b035554

$ COSIGN_EXPERIMENTAL=1 cosign verify-attestation --type spdx ghcr.io/external-secrets/external-secrets@sha256:36e606279dbebac51b4b9300b9fa85e8c08c1c673ba3ecc38af1402a0b035554 | jq '.payload |= @base64d | .payload | fromjson' | jq '.predicate.Data | fromjson'

[...]
{
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "ghcr.io/external-secrets/external-secrets@sha256-36e606279dbebac51b4b9300b9fa85e8c08c1c673ba3ecc38af1402a0b035554",
  "spdxVersion": "SPDX-2.2",
  "creationInfo": {
    "created": "2023-03-17T23:17:01.568002344Z",
    "creators": [
      "Organization: Anchore, Inc",
      "Tool: syft-0.40.1"
    ],
    "licenseListVersion": "3.16"
  },
  "dataLicense": "CC0-1.0",
  "documentNamespace": "https://anchore.com/syft/image/ghcr.io/external-secrets/external-secrets@sha256-36e606279dbebac51b4b9300b9fa85e8c08c1c673ba3ecc38af1402a0b035554-83484ebb-b469-45fa-8fcc-9290c4ea4f6f",
  "packages": [
    [...]
    {
      "SPDXID": "SPDXRef-c809070b0beb099e",
      "name": "tzdata",
      "licenseConcluded": "NONE",
      "downloadLocation": "NOASSERTION",
      "externalRefs": [
        {
          "referenceCategory": "SECURITY",
          "referenceLocator": "cpe:2.3:a:tzdata:tzdata:2021a-1\\+deb11u8:*:*:*:*:*:*:*",
          "referenceType": "cpe23Type"
        },
        {
          "referenceCategory": "PACKAGE_MANAGER",
          "referenceLocator": "pkg:deb/debian/tzdata@2021a-1+deb11u8?arch=all&distro=debian-11",
          "referenceType": "purl"
        }
      ],
      "filesAnalyzed": false,
      "licenseDeclared": "NONE",
      "originator": "Person: GNU Libc Maintainers <debian-glibc@lists.debian.org>",
      "sourceInfo": "acquired package info from DPKG DB: /var/lib/dpkg/status.d/tzdata, /usr/share/doc/tzdata/copyright",
      "versionInfo": "2021a-1+deb11u8"
    }
  ]
}
```
