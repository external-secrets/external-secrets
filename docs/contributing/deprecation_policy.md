<!-- Reference. We explain WHAT is the content of our deprecation policy, verbatim -->
<!-- If you want to contribute to this page: -->
<!-- If you are looking for WHY we have security process x, look for our general policy (policy.md) -->
<!-- If you are looking for "HOW DO I deprecate code x", just do it. look for the DEPRECATING.md page -->
# Deprecation policy

We follow the 9 rules of the [Kubernetes Deprecation Policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/) and [API Versioning Scheme](https://kubernetes.io/docs/reference/using-api/#api-versioning): alpha, beta, GA.
On top of that, we have our own deprecation policy explained here.

(Note: Please read the `DEPRECATING.md` at the base of our repository to know how-to technically deprecate a feature)

## Scope
* API Objects and fields: `.Spec`, `.Status` and `.Status.Conditions[]`
* Enums and constant values
* Controller Configuration: CLI flags & environment variables
* Metrics as defined in the [Kubernetes docs](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-metric)
* The following features or specific behavior:
    * `ExternalSecret` update mechanics

### Deprecations follow our general policy and are considered as impactful

A quick reminder: Our [general policy](policy.md#deprecations-are-considered-impactful) considers deprecations and removals are impactful.
They require a proposal.

### Deprecation notices in release management

Every deprecation warrant a major version bump.

The release notes must prominently include:
* A deprecation notice for the feature.
* The expected timeline for removal (if applicable).

TODO EVRARDJP w/ SKARLSO: Should we add a UPGRADING.md file explaining how to upgrade and force this in our policy?

### Removal timeframe

The removal must follow standard Kubernetes deprecation timelines.

## Non-Scope
Everything not listed in scope is:
* not subject to this deprecation policy
* subject to breaking changes, updates at any point in time, and deprecation **as long as it follows the Deprecation Process listed below**.

This includes, but isn't limited to:

* Any feature / specific behavior not in Scope.
* Source code imports
* Helm Charts
* Release process
* Docker Images (including multi-arch builds)
* Image Signature (including provenance, providers, keys)



