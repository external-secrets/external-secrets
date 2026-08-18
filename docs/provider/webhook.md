## Generic Webhook

External Secrets Operator can integrate with simple web apis by specifying the endpoint

### Example

First, create a SecretStore with a webhook backend.  We'll use a static user/password `test`:

```yaml
{% raw %}
apiVersion: external-secrets.io/v1
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
apiVersion: external-secrets.io/v1
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
#### Push secret

To push a secret, create the following store:

```yaml
{% raw %}
apiVersion: external-secrets.io/v1
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
  refreshInterval: 1h0m0s # Refresh interval for which push secret will reconcile
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

#### Service account tokens

Instead of referencing a `Secret`, an entry in `secrets` can reference a Kubernetes `ServiceAccount`.
The operator then requests a short-lived token for that service account via the Kubernetes TokenRequest
API on every webhook call and exposes it to templates under the `token` key:

```yaml
{% raw %}
apiVersion: external-secrets.io/v1
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
        Authorization: Bearer {{ .auth.token }}
      secrets:
      - name: auth
        serviceAccountRef:
          name: webhook-sa
{%- endraw %}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: webhook-sa
  labels:
    external-secrets.io/type: webhook # Needed to allow webhook to use this service account
```

**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` in all `serviceAccountRef` references, just like for `secretRef`. The optional `audiences` field is passed to the TokenRequest API.

#### Authentication

Webhook also supports using NTLM for authorization:

```yaml
{% raw %}
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: webhook-backend
spec:
  provider:
    webhook:
      url: "http://httpbin.org/get?parameter={{ .remoteRef.key }}"
      result:
        jsonPath: "$.args.parameter"
      auth:
        ntlm:
            usernameSecret:
              name: webhook-credentials
              key: username
              namespace: externalsecrets
            passwordSecret:
              name: webhook-credentials
              key: password
              namespace: externalsecrets
{%- endraw %}
---
apiVersion: v1
kind: Secret
metadata:
  name: webhook-credentials
  namespace: externalsecrets
  labels:
    external-secrets.io/type: webhook # Also required for auth.ntlm secrets
data:
  username: dGVzdA== # "test"
  password: dGVzdA== # "test"
```




!!! note
      If a webhook endpoint for a given `ExternalSecret` returns a 404 status code, the secret is considered to have been deleted.  This will trigger the `deletionPolicy` set on the `ExternalSecret`.

### Templating

Generic WebHook provider uses the templating engine to generate the API call.  It can be used in the url, headers, body and result.jsonPath fields.

The provider inserts the secret to be retrieved in the object named `remoteRef`.

In addition, secrets can be added as named objects, for example to use in authorization headers.
Each secret has a `name` property which determines the name of the object in the templating engine.

### All Parameters

```yaml
apiVersion: external-secrets.io/v1
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
      # List of secrets to expose to the templating engine.
      # Each entry must set exactly one of secretRef or serviceAccountRef.
      secrets:
      # Use this name to refer to this secret in templating, above
      - name: <name>
        secretRef:
          namespace: <namespace> # Only used in ClusterSecretStores
          name: <name>
      # A service account token, exposed to templates as `token`
      - name: <name>
        serviceAccountRef:
          namespace: <namespace> # Only used in ClusterSecretStores
          name: <name>
          audiences: [<audience>] # Optional token audiences
      # Add CAs here for the TLS handshake
      caBundle: <base64 encoded cabundle>
      caProvider:
        type: Secret or ConfigMap
        name: <name of secret or configmap>
        namespace: <namespace> # Only used in ClusterSecretStores
        key: <key inside secret>
```

### Webhook as generators
You can also leverage webhooks as generators, following the same syntax. Please see the
[generator-webhook](../api/generator/webhook.md) documentation for more information.

Note that source secrets must be labeled for both secretstores and generators, see
[Referenced secrets must be labeled](#referenced-secrets-must-be-labeled) below.

### Debugging

#### Start with the store status and events

Most webhook problems surface on the `SecretStore` itself rather than in the logs:

```sh
kubectl describe secretstore <name> -n <namespace>
# or, for a cluster-scoped store
kubectl describe clustersecretstore <name>
```

A `Ready` condition of `False` with reason `InvalidProviderConfig` means the client could
not be created or that store validation failed. The accompanying event carries the
underlying error, which is usually more specific than the condition message.

For per-secret failures, check the `ExternalSecret` instead:

```sh
kubectl describe externalsecret <name> -n <namespace>
```

#### Increase the operator log level

Run the controller with `--loglevel debug` to get the reconcile decisions for each object.
This logs what the operator did and why, but it deliberately does not log HTTP request or
response contents, see below.

#### Why there is no built-in request or response dump

The webhook provider renders `url`, `headers` and `body` through the templating engine, and
those templates typically contain credentials pulled from the referenced secrets. Dumping
requests or responses would therefore write those credentials to the operator log, where
anyone with log access could read them. That is why no trace or wire-dump option is
provided, and why one is unlikely to be added.

!!! warning
      Setting `GODEBUG=http2debug=2` on the operator does produce HTTP/2 frame dumps
      including authorization headers and full bodies, entirely unredacted. It only covers
      HTTP/2, it is not a supported debugging path, and it must not be enabled against a
      production instance.

#### Inspecting traffic with a logging proxy

The supported way to see the traffic is to put a proxy between the operator and the target
service, and read the proxy's logs. Point the webhook `url` at the proxy over plain HTTP so
the request is readable there, and let the proxy terminate TLS towards the real endpoint.
Running it as a sidecar keeps the plaintext hop inside the pod.

Add the sidecar and its config volume to the external-secrets deployment. Only the fields
relevant to the proxy are shown; keep the rest of the pod spec as it is.

```yaml
# spec.template.spec.containers
- name: debug-proxy
  image: nginx:alpine
  ports:
    - containerPort: 8080
  volumeMounts:
    - name: debug-proxy-config
      mountPath: /etc/nginx/conf.d
# spec.template.spec.volumes
- name: debug-proxy-config
  configMap:
    name: debug-proxy-config
```

The key must end in `.conf`, because the stock nginx config only includes
`/etc/nginx/conf.d/*.conf`. With any other key the sidecar starts and serves the default
nginx page instead of proxying.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: debug-proxy-config
data:
  debug-proxy.conf: |
    log_format dump escape=none '$request_method $uri $args -> $upstream_status'
                                ' authorization="$http_authorization"';

    server {
      listen 8080;
      access_log /dev/stdout dump;
      location / {
        proxy_pass https://secrets.example.com;
        proxy_set_header Host secrets.example.com;
        # nginx sends no SNI by default, which breaks endpoints that select a
        # certificate by server name.
        proxy_ssl_server_name on;
      }
    }
```

That logs the request method, path and query, the upstream status, and the `Authorization`
header. Add more `$http_<header>` variables to the `log_format` for other headers. Request
and response bodies are not captured; use `mirror` or a dedicated capture proxy if you need
them.

Then set `url: "http://localhost:8080/..."` on the store while debugging. Remember that the
proxy log now contains the same credentials the operator refuses to log, so treat it as
sensitive and remove the sidecar when you are done.

!!! warning
      nginx does not verify the upstream certificate unless `proxy_ssl_verify on` and
      `proxy_ssl_trusted_certificate` are set, so this proxy is deliberately more permissive
      than the operator. Do not use it to diagnose TLS trust problems: a `caProvider` or
      certificate error will appear to go away as soon as the proxy is in the path.

#### Referenced secrets must be labeled

Every Secret the webhook reads must carry the label `external-secrets.io/type: webhook`.
That covers `spec.provider.webhook.secrets` on a `SecretStore` or `ClusterSecretStore`,
`spec.secrets` on a `Webhook` generator, and the `usernameSecret` and `passwordSecret` of
`auth.ntlm`, which resolve through the same code path. Without it the lookup fails with:

```
secret does not contain needed label 'external-secrets.io/type: webhook'. Update secret label to use it with webhook
```

The same applies to every ServiceAccount referenced via `serviceAccountRef`; without the
label, token creation fails with an analogous error.

#### Template values are two levels deep

Secrets listed under `secrets` are exposed as `.<name>.<keyInSecret>`, not `.<name>`. For:

```yaml
secrets:
  - name: creds
    secretRef:
      name: webhook-credentials
```

the values are `{{ .creds.username }}` and `{{ .creds.password }}`. Referring to
`{{ .creds }}` renders a Go map rather than a value. Note also that every key of the
referenced secret is exposed; a `key` field on the `secretRef` does not narrow it.
For `serviceAccountRef` entries the only exposed key is `token`, i.e. `{{ .<name>.token }}`.

#### A templated `url` is validated before templating

Store validation performs a reachability check against `spec.provider.webhook.url` as
written, before the templating engine runs. A url whose host comes from a template
therefore fails validation with a message such as:

```
error accessing external store: dial tcp :443: connect: connection refused
```

even when the templated request itself would succeed. Keep the host literal and template
only the path or query string.
