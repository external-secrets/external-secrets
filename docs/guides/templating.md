# Advanced Templating v2

With External Secrets Operator you can transform the data from the external secret provider before it is stored as `Kind=Secret`. You can do this with the `Spec.Target.Template`.

Each data value is interpreted as a [Go template](https://golang.org/pkg/text/template/). Please note that referencing a non-existing key in the template will raise an error, instead of being suppressed.

!!! note

    Consider using camelcase when defining  **.'spec.data.secretkey'**, example: serviceAccountToken

    If your secret keys contain **`-` (dashes)**, you will need to reference them using **`index`** </br>
    Example: **`\{\{ index .data "service-account-token" \}\}`**

## Helm

When installing ExternalSecrets via `helm`, the template must be escaped so that `helm` will not try to render it. The most straightforward way to accomplish this would be to use backticks ([raw string constants](https://pkg.go.dev/text/template#hdr-Examples)):

```yaml
{% include 'helm-template-v2-escape-sequence.yaml' %}
```

## Examples

You can use templates to inject your secrets into a configuration file that you mount into your pod:

```yaml
{% include 'multiline-template-v2-external-secret.yaml' %}
```

Another example with two keys in the same secret:

```yaml
{% include 'multikey-template-v2-external-secret.yaml' %}
```

### MergePolicy

By default, the templating mechanism will not use any information available from the original `data` and `dataFrom` queries to the provider, and only keep the templated information. It is possible to change this behavior through the use of the `mergePolicy` field. `mergePolicy` currently accepts two values: `Replace` (the default) and `Merge`. When using `Merge`, `data` and `dataFrom` keys will also be embedded into the templated secret, having lower priority than the template outcome. See the example for more information:

```yaml
{% include 'merge-template-v2-external-secret.yaml' %}
```

### TemplateFrom

You do not have to define your templates inline in an ExternalSecret but you can pull `ConfigMaps` or other Secrets that contain a template. Consider the following example:

```yaml
{% include 'template-v2-from-secret.yaml' %}
```

`TemplateFrom` also gives you the ability to Target your template to the Secret's Annotations, Labels or the Data block. It also allows you to render the templated information as `Values` or as `KeysAndValues` through the `templateAs` configuration:

```yaml
{% include 'template-v2-scope-and-target.yaml' %}
```

Lastly, `TemplateFrom` also supports adding `Literal` blocks for quick templating. These `Literal` blocks differ from `Template.Data` as they are rendered as a a `key:value` pair (while the `Template.Data`, you can only template the value).

See an example, how to produce a `htpasswd` file that can be used by an ingress-controller (for example: https://kubernetes.github.io/ingress-nginx/examples/auth/basic/) where the contents of the `htpasswd` file needs to be presented via the `auth` key. We use the `htpasswd` function to create a `bcrytped` hash of the password.

Suppose you have multiple key-value pairs within your provider secret like

```json
{
  "user1": "password1",
  "user2": "password2",
  ...
}
```

```yaml
{% include 'template-v2-literal-example.yaml' %}
```

### Extract Keys and Certificates from PKCS#12 Archive

You can use pre-defined functions to extract data from your secrets. Here: extract keys and certificates from a PKCS#12 archive and store it as PEM.

```yaml
{% include 'pkcs12-template-v2-external-secret.yaml' %}
```

### Extract from JWK

You can extract the public or private key parts of a JWK and use them as [PKCS#8](https://pkg.go.dev/crypto/x509#ParsePKCS8PrivateKey) private key or PEM-encoded [PKIX](https://pkg.go.dev/crypto/x509#MarshalPKIXPublicKey) public key.

A JWK looks similar to this:

```json
{
  "kty": "RSA",
  "kid": "cc34c0a0-bd5a-4a3c-a50d-a2a7db7643df",
  "use": "sig",
  "n": "pjdss...",
  "e": "AQAB"
  // ...
}
```

And what you want may be a PEM-encoded public or private key portion of it. Take a look at this example on how to transform it into the desired format:

```yaml
{% include 'jwk-template-v2-external-secret.yaml' %}
```

### Filter PEM blocks

Consider you have a secret that contains both a certificate and a private key encoded in PEM format and it is your goal to use only the certificate from that secret.

```
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCvxGZOW4IXvGlh
 . . .
m8JCpbJXDfSSVxKHgK1Siw4K6pnTsIA2e/Z+Ha2fvtocERjq7VQMAJFaIZSTKo9Q
JwwY+vj0yxWjyzHUzZB33tg=
-----END PRIVATE KEY-----
-----BEGIN CERTIFICATE-----
MIIDMDCCAhigAwIBAgIQabPaXuZCQaCg+eQAVptGGDANBgkqhkiG9w0BAQsFADAV
 . . .
NtFUGA95RGN9s+pl6XY0YARPHf5O76ErC1OZtDTR5RdyQfcM+94gYZsexsXl0aQO
9YD3Wg==
-----END CERTIFICATE-----

```

You can achieve that by using the `filterPEM` function to extract a specific type of PEM block from that secret. If multiple blocks of that type (here: `CERTIFICATE`) exist, all of them are returned in the order specified. To extract a specific type of PEM block, pass the type as a string argument to the filterPEM function. Take a look at this example of how to transform a secret which contains a private key and a certificate into the desired format:

```yaml
{% include 'filterpem-template-v2-external-secret.yaml' %}
```

## Templating with PushSecret

`PushSecret` templating is much like `ExternalSecrets` templating. In-fact under the hood, it's using the same data structure.
Which means, anything described in the above should be possible with push secret as well resulting in a templated secret
created at the provider.

```yaml
{% include 'template-v2-push-secret.yaml' %}
```

## Helper functions

!!! info inline end

    Note: we removed `env` and `expandenv` from sprig functions for security reasons.

We provide a couple of convenience functions that help you transform your secrets. This is useful when dealing with PKCS#12 archives or JSON Web Keys (JWK).

In addition to that you can use over 200+ [sprig functions](http://masterminds.github.io/sprig/). If you feel a function is missing or might be valuable feel free to open an issue and submit a [pull request](../contributing/process.md#submitting-a-pull-request).

<br/>

| Function         | Description                                                                                                                                                                                                                  |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| pkcs12key        | Extracts all private keys from a PKCS#12 archive and encodes them in **PKCS#8 PEM** format.                                                                                                                                  |
| pkcs12keyPass    | Same as `pkcs12key`. Uses the provided password to decrypt the PKCS#12 archive.                                                                                                                                              |
| pkcs12cert       | Extracts all certificates from a PKCS#12 archive and orders them if possible. If disjunct or multiple leaf certs are provided they are returned as-is. <br/> Sort order: `leaf / intermediate(s) / root`.                    |
| pkcs12certPass   | Same as `pkcs12cert`. Uses the provided password to decrypt the PKCS#12 archive.                                                                                                                                             |
| pemToPkcs12      | Takes a PEM encoded certificate and key and creates a base64 encoded PKCS#12 archive.                                                                                                                                         |
| pemToPkcs12Pass  | Same as `pemToPkcs12`. Uses the provided password to encrypt the PKCS#12 archive.                                                                                                                                            |
| fullPemToPkcs12      | Takes a PEM encoded certificates chain and key and creates a base64 encoded PKCS#12 archive.                                                                                                                                         |
| fullPemToPkcs12Pass  | Same as `fullPemToPkcs12`. Uses the provided password to encrypt the PKCS#12 archive.                                                                                                                                            |
| filterPEM        | Filters PEM blocks with a specific type from a list of PEM blocks.                                                                                                                                                           |
| jwkPublicKeyPem  | Takes an json-serialized JWK and returns an PEM block of type `PUBLIC KEY` that contains the public key. [See here](https://golang.org/pkg/crypto/x509/#MarshalPKIXPublicKey) for details.                                   |
| jwkPrivateKeyPem | Takes an json-serialized JWK as `string` and returns an PEM block of type `PRIVATE KEY` that contains the private key in PKCS #8 format. [See here](https://golang.org/pkg/crypto/x509/#MarshalPKCS8PrivateKey) for details. |
| toYaml           | Takes an interface, marshals it to yaml. It returns a string, even on marshal error (empty string).                                                                                                                          |
| fromYaml         | Function converts a YAML document into a map[string]any.                                                                                                                                                             |

## Migrating from v1

If you are still using `v1alpha1`, You have to opt-in to use the new engine version by specifying `template.engineVersion=v2`:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecret
metadata:
  name: secret
spec:
  # ...
  target:
    template:
      engineVersion: v2
  # ...
```

The biggest change was that basically all function parameter types were changed from accepting/returning `[]byte` to `string`. This is relevant for you because now you don't need to specify `toString` all the time at the end of a template pipeline.

```yaml
{% raw %}
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecret
# ...
spec:
  target:
    template:
      engineVersion: v2
      data:
        # this used to be {{ .foobar | toString }}
        egg: "new: {{ .foobar }}"
{% endraw %}
```

##### Functions removed/replaced

- `base64encode` was renamed to `b64enc`.
- `base64decode` was renamed to `b64dec`. Any errors that occur during decoding are silenced.
- `fromJSON` was renamed to `fromJson`. Any errors that occur during unmarshalling are silenced.
- `toJSON` was renamed to `toJson`. Any errors that occur during marshalling are silenced.
- `pkcs12key` and `pkcs12keyPass` encode the PKCS#8 key directly into PEM format. There is no need to call `pemPrivateKey` anymore. Also, these functions do extract all private keys from the PKCS#12 archive not just the first one.
- `pkcs12cert` and `pkcs12certPass` encode the certs directly into PEM format. There is no need to call `pemCertificate` anymore. These functions now **extract all certificates** from the PKCS#12 archive not just the first one.
- `toString` implementation was replaced by the `sprig` implementation and should be api-compatible.
- `toBytes` was removed.
- `pemPrivateKey` was removed. It's now implemented within the `pkcs12*` functions.
- `pemCertificate` was removed. It's now implemented within the `pkcs12*` functions.
