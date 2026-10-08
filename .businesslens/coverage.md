---
scope: The operator's v1 and v1alpha1 resources, their controllers and admission checks, the generators and their credential cleanup, the controller options, the metrics endpoint, the esoctl template command and the shipped RBAC roles.
method: Static inspection of source, CRD types, the Helm chart and documentation; nothing was built or run.
covered:
  - description: ExternalSecret, ClusterExternalSecret and store API types with their admission validators.
    paths: [apis/externalsecrets/v1/]
  - description: PushSecret and ClusterPushSecret API types.
    paths: [apis/externalsecrets/v1alpha1/]
  - description: Generator, ClusterGenerator and generator state API types.
    paths: [apis/generators/v1alpha1/]
  - description: Controllers for stores, ExternalSecrets, ClusterExternalSecrets, PushSecrets, ClusterPushSecrets and generator state, with their shared helpers.
    paths: [pkg/controllers/secretstore/, pkg/controllers/externalsecret/, pkg/controllers/clusterexternalsecret/, pkg/controllers/clusterpushsecret/, pkg/controllers/pushsecret/, pkg/controllers/generatorstate/, pkg/controllers/templating/, pkg/controllers/util/, pkg/controllers/common/]
  - description: Generator implementations and their registration.
    paths: [generators/v1/, pkg/register/generators.go]
  - description: Shared runtime for reference resolution, templating, decoding, key finding, validation and generator state.
    paths: [runtime/esutils/, runtime/template/, runtime/decoding/, runtime/find/, runtime/statemanager/]
  - description: Prometheus metrics the controller exposes.
    paths: [runtime/metrics/, pkg/controllers/metrics/]
  - description: Controller entry point, its options and the admission webhook server.
    paths: [main.go, cmd/controller/root.go, cmd/controller/webhook.go]
  - description: The esoctl template command.
    paths: [cmd/esoctl/template.go, cmd/esoctl/main.go, cmd/esoctl/root.go]
  - description: The view and edit roles the Helm chart ships.
    paths: [deploy/charts/external-secrets/templates/rbac.yaml]
exclusions:
  - description: Webhook certificate management and CRD conversion certificate injection.
    paths: [cmd/controller/certcontroller.go, pkg/controllers/crds/, pkg/controllers/webhookconfig/]
  - description: Contributor scaffolding for new generators.
    paths: [cmd/esoctl/bootstrap.go, cmd/esoctl/generator/]
  - description: Packaging, installation manifests and the Grafana dashboard, other than the shipped roles.
    paths: [deploy/charts/external-secrets/templates/deployment.yaml, deploy/charts/external-secrets/files/, deploy/manifests/, config/]
  - description: Documentation site, design notes and images.
    paths: [docs/, design/, assets/, hack/, overrides/]
  - description: Test suites, end-to-end environments and CI.
    paths: [e2e/, tests/, terraform/, .github/, pkg/controllers/commontest/, runtime/testing/]
unmapped:
  - description: Per-provider clients, their authentication methods and which operations each provider supports.
    paths: [providers/v1/, pkg/register/]
  - description: The deprecated v1beta1 API version.
    paths: [apis/externalsecrets/v1beta1/]
  - description: Provider client caching, OIDC token exchange and provider feature flags.
    paths: [runtime/cache/, runtime/oidc/, runtime/feature/]
limitations: []
---

# Coverage
