# Deprecation Policy

We follow the [Kubernetes Deprecation Policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/) and [API Versioning Scheme](https://kubernetes.io/docs/reference/using-api/#api-versioning): alpha, beta, GA.

The project is currently in `beta` state. Please try the `beta` features and provide feedback. After the features exits beta, it may not be practical to make more changes.

* alpha
    * The support for a feature may be dropped at any time without notice.
    * The API may change in incompatible ways in a later software release without notice.
    * The software is recommended for use only in short-lived testing clusters, due to increased risk of bugs and lack of long-term support.

* beta
    * The software is well tested. Enabling a feature is considered safe. Features are enabled by default.
    * The support for a feature will not be dropped, though the details may change.
    * The schema and/or semantics of objects may change in incompatible ways in a subsequent beta or stable release. When this happens, migration instructions are provided. Schema changes may require deleting, editing, and re-creating API objects. The editing process may not be straightforward. The migration may require downtime for applications that rely on the feature.
    * The software is not recommended for production uses. Subsequent releases may introduce incompatible changes. If you have multiple clusters which can be upgraded independently, you may be able to relax this restriction.
* GA
    * The stable versions of features appear in released software for many subsequent versions.
    * Use it in production ;)

## API Surface

We define the following scope that is covered by our deprecation policy. We follow the [9 Rules of the Kubernetes Deprecation Policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/).

### Scope
* API Objects and fields: `.Spec`, `.Status` and `.Status.Conditions[]`
* Enums and constant values
* Controller Configuration: CLI flags & environment variables
* Metrics as defined in the [Kubernetes docs](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-metric)
* The following features or specific behavior:
    * `ExternalSecret` [update mechanics](http://localhost:8000/api-externalsecret/#update-behavior)

### Non-Scope
Everything not listed in scope is not subject to this deprecation policy and it is subject to breaking changes, updates at any point in time, and deprecation - **as long as it follows the Deprecation Process listed below**.

This includes, but insn't limited to :
* Any feature / specific behavior not in Scope.
* Source code imports
* Helm Charts
* Release process
* Docker Images (including multi-arch builds)
* Image Signature (including provenance, providers, keys)
* OLM-specific builds

## Depreaction Process:

Deprecation process is described within the [project github repository](https://github.com/external-secrets/external-secrets/blob/main/DEPRECATING.md)