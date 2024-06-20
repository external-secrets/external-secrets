# Advanced Templating v1

!!! warning

    Templating Engine v1 is **deprecated** and will be removed in the future. Please migrate to engine v2 and take a look at our [upgrade guide](templating.md#migrating-from-v1) for changes.


With External Secrets Operator you can transform the data from the external secret provider before it is stored as `Kind=Secret`. You can do this with the `Spec.Target.Template`.

Each data value is interpreted as a [Go template](https://golang.org/pkg/text/template/). Please note that referencing a non-existing key in the template will raise an error, instead of being suppressed.

## Examples

You can use templates to inject your secrets into a configuration file that you mount into your pod:
``` yaml
{% include 'multiline-template-v1-external-secret.yaml' %}
```

You can also use pre-defined functions to extract data from your secrets. Here: extract key/cert from a pkcs12 archive and store it as PEM.
``` yaml
{% include 'pkcs12-template-v1-external-secret.yaml' %}
```

### TemplateFrom

You do not have to define your templates inline in an ExternalSecret but you can pull `ConfigMaps` or other Secrets that contain a template. Consider the following example:

``` yaml
{% include 'template-v1-from-secret.yaml' %}
```

## Helper functions
We provide a bunch of convenience functions that help you transform your secrets. A secret value is a `[]byte`.

| Function       | Description                                                                | Input                            | Output        |
| -------------- | -------------------------------------------------------------------------- | -------------------------------- | ------------- |
| pkcs12key      | extracts the private key from a pkcs12 archive                             | `[]byte`                         | `[]byte`      |
| pkcs12keyPass  | extracts the private key from a pkcs12 archive using the provided password | password `string`, data `[]byte` | `[]byte`      |
| pkcs12cert     | extracts the certificate from a pkcs12 archive                             | `[]byte`                         | `[]byte`      |
| pkcs12certPass | extracts the certificate from a pkcs12 archive using the provided password | password `string`, data `[]byte` | `[]byte`      |
| pemPrivateKey  | PEM encodes the provided bytes as private key                              | `[]byte`                         | `string`      |
| pemCertificate | PEM encodes the provided bytes as certificate                              | `[]byte`                         | `string`      |
| jwkPublicKeyPem | takes an json-serialized JWK as `[]byte` and returns an PEM block of type `PUBLIC KEY` that contains the public key ([see here](https://golang.org/pkg/crypto/x509/#MarshalPKIXPublicKey)) for details | `[]byte`                         | `string`      |
| jwkPrivateKeyPem | takes an json-serialized JWK as `[]byte` and returns an PEM block of type `PRIVATE KEY` that contains the private key in PKCS #8 format ([see here](https://golang.org/pkg/crypto/x509/#MarshalPKCS8PrivateKey)) for details | `[]byte`                         | `string`      |
| base64decode   | decodes the provided bytes as base64                                       | `[]byte`                         | `[]byte`      |
| base64encode   | encodes the provided bytes as base64                                       | `[]byte`                         | `[]byte`      |
| fromJSON       | parses the bytes as JSON so you can access individual properties           | `[]byte`                         | `any` |
| toJSON         | encodes the provided object as json string                                 | `any`                    | `string`      |
| toString       | converts bytes to string                                                   | `[]byte`                         | `string`      |
| toBytes        | converts string to bytes                                                   | `string`                         | `[]byte`      |
| upper          | converts all characters to their upper case                                | `string`                         | `string`      |
| lower          | converts all character to their lower case                                 | `string`                         | `string`      |
