# Security Best Practices

The purpose of this document is to outline a set of best practices for securing the External Secrets Operator (ESO). These practices aim to mitigate the risk of successful attacks against ESO and the Kubernetes cluster it integrates with.

## Security Functions and Features

### 1. Namespace Isolation

To maintain security boundaries, ESO ensures that namespaced resources like `SecretStore` and `ExternalSecret` are limited to their respective namespaces. The following rules apply:

1. `ExternalSecret` resources must not have cross-namespace references of `Kind=SecretStore` or `Kind=Secret` resources
2. `SecretStore` resources must not have cross-namespace references of `Kind=Secret` or others

For cluster-wide resources like `ClusterSecretStore` and `ClusterExternalSecret`, exercise caution since they have access to Secret resources across all namespaces. Minimize RBAC permissions for administrators and developers to the necessary minimum. If cluster-wide resources are not required, it is recommended to disable them.

### 2. Configure ClusterSecretStore match conditions

Utilize the ClusterSecretStore resource to define specific match conditions using `namespaceSelector` or an explicit namespaces list. This restricts the usage of the `ClusterSecretStore` to a predetermined list of namespaces or a namespace that matches a predefined label. Here's an example:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: fake
spec:
  conditions:
    - namespaceSelector:
        matchLabels:
          app: frontend
```

### 3. Selectively Disable Reconciliation of Cluster-Wide Resources

ESO allows you to selectively disable the reconciliation of cluster-wide resources such as `ClusterSecretStore`, `ClusterExternalSecret`, and `PushSecret`. You can disable the installation of CRDs in the Helm chart or disable reconciliation in the core-controller using the following options:

To disable CRD installation:

```yaml
# disable cluster-wide resources & push secret
crds:
  createClusterExternalSecret: false
  createClusterSecretStore: false
  createPushSecret: false
```

To disable reconciliation in the core-controller:

```
--enable-cluster-external-secret-reconciler
--enable-cluster-store-reconciler
```

### 4. Implement Namespace-Scoped Installation

To further enhance security, consider installing ESO into a specific namespace with restricted access to only that namespace's resources. This prevents access to cluster-wide secrets. Use the following Helm values to scope the controller to a specific namespace:

```yaml
# If set to true, create scoped RBAC roles under the scoped namespace
# and implicitly disable cluster stores and cluster external secrets
scopedRBAC: true

# Specify the namespace where external secrets should be reconciled
scopedNamespace: my-namespace
```

## Pod Security

The Pods of the External Secrets Operator have been configured to meet the [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/), specifically the restricted profile. This configuration ensures a strong security posture by implementing recommended best practices for hardening Pods, including those outlined in the [NSA Kubernetes Hardening Guide](https://media.defense.gov/2022/Aug/29/2003066362/-1/-1/0/CTR_KUBERNETES_HARDENING_GUIDANCE_1.2_20220829.PDF).

By adhering to these standards, the External Secrets Operator benefits from a secure and resilient operating environment. The restricted profile has been set as the default configuration since version `v0.8.2`, and it is recommended to maintain this setting to align with the principle of least privilege.

## Role-Based Access Control (RBAC)

The External Secrets Operator operates with elevated privileges within your Kubernetes cluster, allowing it to read and write to all secrets across all namespaces. It is crucial to properly restrict access to ESO resources such as `ExternalSecret` and `SecretStore` where necessary. This is particularly important for cluster-scoped resources like `ClusterExternalSecret` and `ClusterSecretStore`. Unauthorized tampering with these resources by an attacker could lead to unauthorized access to secrets or potential data exfiltration from your system.

In most scenarios, the External Secrets Operator is deployed cluster-wide. However, if you prefer to run it on a per-namespace basis, you can scope it to a specific namespace using the `scopedRBAC` and `scopedNamespace` options in the helm chart.

To ensure a secure RBAC configuration, consider the following checklist:

* Restrict access to execute shell commands (pods/exec) within the External Secrets Operator Pod.
* Restrict access to (Cluster)ExternalSecret and (Cluster)SecretStore resources.
* Limit access to aggregated ClusterRoles (view/edit/admin) as needed.
* If necessary, deploy ESO with scoped RBAC or within a specific namespace.

By carefully managing RBAC permissions and scoping the External Secrets Operator appropriately, you can enhance the security of your Kubernetes cluster.

## Network Traffic and Security

To ensure a secure network environment, it is recommended to restrict network traffic to and from the External Secrets Operator using `NetworkPolicies` or similar mechanisms. By default, the External Secrets Operator does not include pre-defined Network Policies.

To implement network restrictions effectively, consider the following steps:

* Define and apply appropriate NetworkPolicies to limit inbound and outbound traffic for the External Secrets Operator.
* Specify a "deny all" policy by default and selectively permit necessary communication based on your specific requirements.
* Restrict access to only the required endpoints and protocols for the External Secrets Operator, such as communication with the Kubernetes API server or external secret providers.
* Regularly review and update the Network Policies to align with changes in your network infrastructure and security requirements.

It is the responsibility of the user to define and configure Network Policies tailored to their specific environment and security needs. By implementing proper network restrictions, you can enhance the overall security posture of the External Secrets Operator within your Kubernetes cluster.

!!! danger "Data Exfiltration Risk"

    If not configured properly ESO may be used to exfiltrate data out of your cluster.
    It is advised to create tight NetworkPolicies and use a policy engine such as kyverno to prevent data exfiltration.


### Outbound Traffic Restrictions

#### Core Controller

Restrict outbound traffic from the core controller component to the following destinations:

* `kube-apiserver`: The Kubernetes API server.
* Secret provider (e.g., AWS, GCP): Whenever possible, use private endpoints to establish secure and private communication.

#### Webhook

* Restrict outbound traffic from the webhook component to the `kube-apiserver`.

#### Cert Controller

* Restrict outbound traffic from the cert controller component to the `kube-apiserver`.


### Inbound Traffic Restrictions

#### Core Controller

* Restrict inbound traffic to the core controller component by allowing communication on port `8080` from your monitoring agent.

#### Cert Controller

* Restrict inbound traffic to the cert controller component by allowing communication on port `8080` from your monitoring agent.
* Additionally, permit inbound traffic on port `8081` from the kubelet for health check endpoints (healthz/readyz).

#### Webhook

Restrict inbound traffic to the webhook component as follows:

* Allow communication on port `10250` from the kube-apiserver.
* Allow communication on port `8080` from your monitoring agent.
* Permit inbound traffic on port `8081` from the kubelet for health check endpoints (healthz/readyz).

## Policy Engine Best Practices

To enhance the security and enforce specific policies for External Secrets Operator (ESO) resources such as SecretStore and ExternalSecret, it is recommended to utilize a policy engine like [Kyverno](http://kyverno.io/) or [OPA Gatekeeper](https://github.com/open-policy-agent/gatekeeper). These policy engines provide a way to define and enforce custom policies that restrict changes made to ESO resources.

!!! danger "Data Exfiltration Risk"

    ESO could be used to exfiltrate data out of your cluster. You should disable all providers you don't need.
    Further, you should implement `NetworkPolicies` to restrict network access to known entities (see above), to prevent data exfiltration.

Here are some recommendations to consider when configuring your policies:

1. **Explicitly Deny Unused Providers**: Create policies that explicitly deny the usage of secret providers that are not required in your environment. This prevents unauthorized access to unnecessary providers and reduces the attack surface.
2. **Restrict Access to Secrets**: Implement policies that restrict access to secrets based on specific conditions. For example, you can define policies to allow access to secrets only if they have a particular prefix in the `.spec.data[].remoteRef.key` field. This helps ensure that only authorized entities can access sensitive information.
3. **Restrict `ClusterSecretStore` References**: Define policies to restrict the usage of ClusterSecretStore references within ExternalSecret resources. This ensures that the resources are properly scoped and prevent potential unauthorized access to secrets across namespaces.

By leveraging a policy engine, you can implement these recommendations and enforce custom policies that align with your organization's security requirements. Please refer to the documentation of the chosen policy engine (e.g., Kyverno or OPA Gatekeeper) for detailed instructions on how to define and enforce policies for ESO resources.


!!! note "Provider Validation Example Policy"

    The following policy validates the usage of the `provider` field in the SecretStore manifest.

    ```yaml
    {% filter indent(width=4) %}
{% include 'kyverno-policy-secretstore.yaml' %}
    {% endfilter %}
    ```

## Regular Patches

To maintain a secure environment, it is crucial to regularly patch and update all software components of External Secrets Operator and the underlying cluster. By doing so, known vulnerabilities can be addressed, and the overall system's security can be improved. Here are some recommended practices for ensuring timely updates:

1. **Automated Patching and Updating**: Utilize automated patching and updating tools to streamline the process of keeping software components up-to-date
2. **Regular Update ESO**: Stay informed about the latest updates and releases provided for ESO. The development team regularly releases updates to improve stability, performance, and security. Please refer to the [Stability and Support](../introduction/stability-support.md) documentation for more information on the available updates
3. **Cluster-wide Updates**: Apart from ESO, ensure that all other software components within your cluster, such as the operating system, container runtime, and Kubernetes itself, are regularly patched and updated.

By adhering to a regular patching and updating schedule, you can proactively mitigate security risks associated with known vulnerabilities and ensure the overall stability and security of your ESO deployment.

## Verify Artefacts

### Verify Container Images

The container images of External Secrets Operator are signed using Cosign and the keyless signing feature. To ensure the authenticity and integrity of the container image, you can follow the steps outlined below:


```sh
# Retrieve Image Signature
$ crane digest ghcr.io/external-secrets/external-secrets:v0.8.1
sha256:36e606279dbebac51b4b9300b9fa85e8c08c1c673ba3ecc38af1402a0b035554

# verify signature
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

In the output of the verification process, pay close attention to the `optional.Issuer` and `optional.Subject` fields. These fields contain important information about the image's authenticity. Verify that the values of Issuer and Subject match the expected values for the ESO container image. If they do not match, it indicates that the image is not legitimate and should not be used.

By following these steps and confirming that the Issuer and Subject fields align with the expected values for the ESO container image, you can ensure that the image has not been tampered with and is safe to use.


### Verifying Provenance

The External Secrets Operator employs the [SLSA](https://slsa.dev/provenance/v0.1) (Supply Chain Levels for Software Artifacts) standard to create and attest to the provenance of its builds. Provenance verification is essential to ensure the integrity and trustworthiness of the software supply chain. This outlines the process of verifying the attested provenance of External Secrets Operator builds using the cosign tool.

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

Every External Secrets Operator image is accompanied by an SBOM (Software Bill of Materials) in SPDX JSON format. The SBOM provides detailed information about the software components and dependencies used in the image. This technical documentation explains the process of downloading and verifying the SBOM for a specific version of External Secrets Operator using the Cosign tool.

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
