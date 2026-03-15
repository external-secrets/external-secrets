# Deployment files

The 'static deployment manifests' are generated automatically
from the [official helm chart](../charts/external-secrets).

When a new release of external-secrets is cut, these manifests will be
automatically generated and published as an asset **attached to the GitHub release**.

## How can I generate my own manifests?

If you want to build a copy of your own manifests for testing purposes, you
can do so using Helm and Make.

To build the manifests, run:

```bash
make manifests
```

This will generate the static deployment manifests at
`bin/deploy/manifests/external-secrets.yaml`.
