# Rewriting Keys in DataFrom

When calling out an ExternalSecret with `dataFrom.extract` or `dataFrom.find`, it is possible that you end up with a kubernetes secret that has conflicts in the key names, or that you simply want to remove a common path from the secret keys.

In order to do so, it is possible to define a set of rewrite operations using `dataFrom.rewrite`. These operations can be stacked, hence allowing complex manipulations of the secret keys.

Rewrite operations are all applied before `ConversionStrategy` is applied.

## Methods

### Regexp
This method implements rewriting through the use of regular expressions. It needs a `source` and a `target` field. The source field is where the definition of the matching regular expression goes, where the `target` field is where the replacing expression goes.

Some considerations about the implementation of Regexp Rewrite:

1. The input of a subsequent rewrite operation are the outputs of the previous rewrite.
2. If a given set of keys do not match any Rewrite operation, there will be no error. Rather, the original keys will be used.
3. If a `source` is not a compilable `regexp` expression, an error will be produced and the external secret goes into a error state.

## Examples
### Removing a common path from find operations
The following ExternalSecret:
```yaml
{% include 'datafrom-rewrite-remove-path.yaml' %}
```
Will get all the secrets matching `path/to/my/secrets/*` and then rewrite them by removing the common path away.

In this example, if we had the following secrets available in the provider:
```
path/to/my/secrets/username
path/to/my/secrets/password
```
the output kubernetes secret would be:
```yaml
apiVersion: v1
kind: Secret
type: Opaque
data:
    username: ...
    password: ...
```
### Avoiding key collisions
The following ExternalSecret:
```yaml
{% include 'datafrom-rewrite-conflict.yaml' %}

```
Will allow two secrets with the same JSON keys to be imported into a Kubernetes Secret without any conflict.
In this example, if we had the following secrets available in the provider:
```json
{
    "my-secrets-dev": {
        "password": "bar",
     },
    "my-secrets-prod": {
        "password": "safebar",
     }
}
```
the output kubernetes secret would be:
```yaml
apiVersion: v1
kind: Secret
type: Opaque
data:
    dev_password: YmFy #bar
    prod_password: c2FmZWJhcg== #safebar
```

### Remove invalid characters
The following ExternalSecret:
```yaml
{% include 'datafrom-rewrite-invalid-characters.yaml' %}

```
Will remove invalid characters from the secret key.
In this example, if we had the following secrets available in the provider:
```json
{
    "development": {
        "foo/bar": "1111",
        "foo$baz": "2222"
    }
}
```
the output kubernetes secret would be:
```yaml
apiVersion: v1
kind: Secret
type: Opaque
data:
    foo_bar: MTExMQ== #1111
    foo_baz: MjIyMg== #2222
```

## Limitations

Regexp Rewrite is based on golang `regexp`, which in turns implements `RE2` regexp language. There a a series of known limitations to this implementation, such as:

* Lack of ability to do lookaheads or lookbehinds;
* Lack of negation expressions;
* Lack of support for conditional branches;
* Lack of support for possessive repetitions.

A list of compatibility and known limitations considering other commonly used regexp frameworks (such as PCRE and PERL) are listed [here](https://github.com/google/re2/wiki/Syntax).
