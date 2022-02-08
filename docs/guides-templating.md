# Advanced Templating v2

With External Secrets Operator you can transform the data from the external secret provider before it is stored as `Kind=Secret`. You can do this with the `Spec.Target.Template`. Each data value is interpreted as a [golang template](https://golang.org/pkg/text/template/).

## Examples

You can use templates to inject your secrets into a configuration file that you mount into your pod:
``` yaml
{% include 'multiline-template-v2-external-secret.yaml' %}
```

You can also use pre-defined functions to extract data from your secrets. Here: extract key/cert from a pkcs12 archive and store it as PEM.
``` yaml
{% include 'pkcs12-template-v2-external-secret.yaml' %}
```

### TemplateFrom

You do not have to define your templates inline in an ExternalSecret but you can pull `ConfigMaps` or other Secrets that contain a template. Consider the following example:

``` yaml
{% include 'template-v2-from-secret.yaml' %}
```

## Helper functions
!!! info inline end

    Note: we removed `env` and `expandenv` from sprig functions for security reasons.

We provide a couple of convenience functions that help you transform your secrets. This is useful when dealing with pkcs12 or jwk encoded secrets.

In addition to that you can use over 200+ [sprig functions](http://masterminds.github.io/sprig/). If you feel a function is missing or might be valuable feel free to open an issue and submit a [pull request](contributing-process.md#submitting-a-pull-request).

<br/>

| Function       | Description                                                                | Input                            | Output        |
| -------------- | -------------------------------------------------------------------------- | -------------------------------- | ------------- |
| pkcs12key      | extracts the private key from a pkcs12 archive                             | `string`                         | `string`      |
| pkcs12keyPass  | extracts the private key from a pkcs12 archive using the provided password | password `string`, data `string` | `string`      |
| pkcs12cert     | extracts the certificate from a pkcs12 archive                             | `string`                         | `string`      |
| pkcs12certPass | extracts the certificate from a pkcs12 archive using the provided password | password `string`, data `string` | `string`      |
| pemPrivateKey  | PEM encodes the provided bytes as private key                              | `string`                         | `string`      |
| pemCertificate | PEM encodes the provided bytes as certificate                              | `string`                         | `string`      |
| jwkPublicKeyPem | takes an json-serialized JWK as `string` and returns an PEM block of type `PUBLIC KEY` that contains the public key ([see here](https://golang.org/pkg/crypto/x509/#MarshalPKIXPublicKey)) for details | `string`                         | `string`      |
| jwkPrivateKeyPem | takes an json-serialized JWK as `string` and returns an PEM block of type `PRIVATE KEY` that contains the private key in PKCS #8 format ([see here](https://golang.org/pkg/crypto/x509/#MarshalPKCS8PrivateKey)) for details | `string`                         | `string`      |

## Migrating from v1

You have to opt-in to use the new engine version by specifying `template.engineVersion=v2`:
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

        #
        mycert: "{{ .mysecret | pkcs12cert | pemCertificate }}"
{% endraw %}
```

##### Functions removed/replaced

* `base64encode` was renamed to `b64dec`.
* `base64decode` was renamed to `b64dec`. Any errors that occurr during decoding are silenced.
* `fromJSON` was renamed to `toJson`. Any errors that occurr during unmarshalling are silenced.
* `toJSON` was renamed to `toJson`. Any errors that occurr during marshalling are silenced.
* `toString` implementation was replaced by the `sprig` implementation and should be api-compatible.
* `toBytes` was removed.
