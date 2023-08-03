# The road to external-secrets GA

The following external-secret custom resource APIs are considered stable:

* `ExternalSecret`
* `ClusterExternalSecret`
* `SecretStore`
* `ClusterSecretStore`

These CRDs are currently at `v1beta1` and are considered production ready. Going forward, breaking changes to these APIs will be accompanied by a conversion mechanism.

We have identified the following areas of work. This is subject to change while we gather feedback. We have a [GitHub Project Board](https://github.com/orgs/external-secrets/projects/2/views/1) where we organize issues and milestones on a high level.


* Conformance testing
    * ✓ end to end testing with ArgoCD and Flux
    * ✓ end to end testing for all project maintained providers
* API enhancements
    * consolidate provider fields
    * ✓ dataFrom key rewrites
    * provider versioning strategy
    * ✓ pushing secrets to a provider
* Documentation Improvements
    * Troubleshooting Guides
    * ✓ FAQ
    * ✓ review multi tenancy docs
    * ✓ security model for infosec teams
    * ✓ security best practices guide
    * ✓ provider specific guides
* Observability
    * ✓ Provide Grafana Dashboard and Prometheus alerts
    * ✓ add provider-level metrics
* Pentest
* ✓ SBOM
