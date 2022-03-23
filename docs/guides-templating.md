# Advanced Templating v2

With External Secrets Operator you can transform the data from the external secret provider before it is stored as `Kind=Secret`. You can do this with the `Spec.Target.Template`. Each data value is interpreted as a [golang template](https://golang.org/pkg/text/template/).

## Examples

You can use templates to inject your secrets into a configuration file that you mount into your pod:

```yaml
{% include 'multiline-template-v2-external-secret.yaml' %}
```

### TemplateFrom

You do not have to define your templates inline in an ExternalSecret but you can pull `ConfigMaps` or other Secrets that contain a template. Consider the following example:

```yaml
{% include 'template-v2-from-secret.yaml' %}
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

You can achieve that by using the `filterPEM` function to extract a specific type of PEM block from that secret. If multiple blocks of that type (here: `CERTIFICATE`) exist then all of them are returned in the order they are specified.

## Helper functions

!!! info inline end

    Note: we removed `env` and `expandenv` from sprig functions for security reasons.

We provide a couple of convenience functions that help you transform your secrets. This is useful when dealing with PKCS#12 archives or JSON Web Keys (JWK).

In addition to that you can use over 200+ [sprig functions](http://masterminds.github.io/sprig/). If you feel a function is missing or might be valuable feel free to open an issue and submit a [pull request](contributing-process.md#submitting-a-pull-request).

<br/>

| Function       | Description                                                                                                                                                                                               |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| pkcs12key      | Extracts all private keys from a PKCS#12 archive and encodes them in **PKCS#8 PEM** format.                                                                                                               |
| pkcs12keyPass  | Same as `pkcs12key`. Uses the provided password to decrypt the PKCS#12 archive.                                                                                                                           |
| pkcs12cert     | Extracts all certificates from a PKCS#12 archive and orders them if possible. If disjunct or multiple leaf certs are provided they are returned as-is. <br/> Sort order: `leaf / intermediate(s) / root`. |
| pkcs12certPass | Same as `pkcs12cert`. Uses the provided password to decrypt the PKCS#12 archive.                                                                                                                          |
| filterPEM      | Filters PEM blocks with a specific type from a list of PEM blocks.                                                                                                                                        |

| jwkPublicKeyPem | Takes an json-serialized JWK and returns an PEM block of type `PUBLIC KEY` that contains the public key. [See here](https://golang.org/pkg/crypto/x509/#MarshalPKIXPublicKey) for details. |
| jwkPrivateKeyPem | Takes an json-serialized JWK as `string` and returns an PEM block of type `PRIVATE KEY` that contains the private key in PKCS #8 format. [See here](https://golang.org/pkg/crypto/x509/#MarshalPKCS8PrivateKey) for details. |

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
- `base64decode` was renamed to `b64dec`. Any errors that occurr during decoding are silenced.
- `fromJSON` was renamed to `fromJson`. Any errors that occurr during unmarshalling are silenced.
- `toJSON` was renamed to `toJson`. Any errors that occurr during marshalling are silenced.
- `pkcs12key` and `pkcs12keyPass` encode the PKCS#8 key directly into PEM format. There is no need to call `pemPrivateKey` anymore. Also, these functions do extract all private keys from the PKCS#12 archive not just the first one.
- `pkcs12cert` and `pkcs12certPass` encode the certs directly into PEM format. There is no need to call `pemCertificate` anymore. These functions now **extract all certificates** from the PKCS#12 archive not just the first one.
- `toString` implementation was replaced by the `sprig` implementation and should be api-compatible.
- `toBytes` was removed.
- `pemPrivateKey` was removed. It's now implemented within the `pkcs12*` functions.
- `pemCertificate` was removed. It's now implemented within the `pkcs12*` functions.
