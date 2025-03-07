## Generic Webhook

External Secrets Operator can integrate with simple web apis by specifying the endpoint

### Example

First, create a SecretStore with a webhook backend.  We'll use a static user/password `test`:

```yaml
{% raw %}
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: webhook-backend
spec:
  provider:
    webhook:
      url: "http://httpbin.org/get?parameter={{ .remoteRef.key }}"
      result:
        jsonPath: "$.args.parameter"
      headers:
        Content-Type: application/json
        Authorization: Basic {{ print .auth.username ":" .auth.password | b64enc }}
      secrets:
      - name: auth
        secretRef:
          name: webhook-credentials
{%- endraw %}
---
apiVersion: v1
kind: Secret
metadata:
  name: webhook-credentials
  labels:
    external-secrets.io/type: webhook #Needed to allow webhook to use this secret
data:
  username: dGVzdA== # "test"
  password: dGVzdA== # "test"
```

NB: This is obviously not practical because it just returns the key as the result, but it shows how it works

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in all `secrets` references with the namespaces where the secrets reside.

Now create an ExternalSecret that uses the above SecretStore:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: webhook-example
spec:
  refreshInterval: "15s"
  secretStoreRef:
    name: webhook-backend
    kind: SecretStore
  target:
    name: example-sync
  data:
  - secretKey: foobar
    remoteRef:
      key: secret
---
# will create a secret with:
kind: Secret
metadata:
  name: example-sync
data:
  foobar: c2VjcmV0
```

To push a secret, create the following store:

```yaml
{% raw %}
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: webhook-backend
spec:
  provider:
    webhook:
      url: "http://httpbin.org/push?id={{ .remoteRef.remoteKey }}&secret={{ .remoteRef.secretKey }}"
      body: '{"secret-field": "{{ index .remoteRef .remoteRef.remoteKey }}"}'
      headers:
        Content-Type: application/json
        Authorization: Basic {{ print .auth.username ":" .auth.password | b64enc }}
      secrets:
      - name: auth
        secretRef:
          name: webhook-credentials
{%- endraw %}
```

Then create a push secret:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example # Customisable
spec:
  refreshInterval: 1h # Refresh interval for which push secret will reconcile
  secretStoreRefs: # A list of secret stores to push secrets to
    - name: webhook-backend
      kind: SecretStore
  selector:
    secret:
      name: test-secret
  data:
    - conversionStrategy:
      match:
        secretKey: testsecret
        remoteRef:
          remoteKey: remotekey
```
If `secretKey` is not provided, the whole secret is provided JSON encoded.

The secret will be added to the `remoteRef` object so that it is retrievable in the templating engine. The secret will be sent in the body when the body field of the provider is empty. In the rare case that the body should be empty, the provider can be configured to use `{% raw %}'{{ "" }}'{% endraw %}` for the body value.

#### Limitations

Webhook does not support authorization, other than what can be sent by generating http headers

!!! note
      If a webhook endpoint for a given `ExternalSecret` returns a 404 status code, the secret is considered to have been deleted.  This will trigger the `deletionPolicy` set on the `ExternalSecret`.

### Templating

Generic WebHook provider uses the templating engine to generate the API call.  It can be used in the url, headers, body and result.jsonPath fields.

The provider inserts the secret to be retrieved in the object named `remoteRef`.

In addition, secrets can be added as named objects, for example to use in authorization headers.
Each secret has a `name` property which determines the name of the object in the templating engine.

### All Parameters

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: statervault
spec:
  provider:
    webhook:
      # Url to call.  Use templating engine to fill in the request parameters
      url: <url>
      # http method, defaults to GET
      method: <method>
      # Timeout in duration (1s, 1m, etc)
      timeout: 1s
      result:
        # [jsonPath](https://jsonpath.com) syntax, which also can be templated
        jsonPath: <jsonPath>
      # Map of headers, can be templated
      headers:
        <Header-Name>: <header contents>
      # Body to sent as request, can be templated (optional)
      body: <body>
      # List of secrets to expose to the templating engine
      secrets:
      # Use this name to refer to this secret in templating, above
      - name: <name>
        secretRef:
          namespace: <namespace> # Only used in ClusterSecretStores
          name: <name>
      # Add CAs here for the TLS handshake
      caBundle: <base64 encoded cabundle>
      caProvider:
        type: Secret or ConfigMap
        name: <name of secret or configmap>
        namespace: <namespace> # Only used in ClusterSecretStores
        key: <key inside secret>
```

### Webhook as generators
You can also leverage webhooks as generators, following the same syntax. The only difference is that the webhook generator needs its source secrets to be labeled, as opposed to webhook secretstores. Please see the [generator-webhook](../api/generator/webhook.md) documentation for more information.
