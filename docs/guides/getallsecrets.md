# Fetching information from multiple secrets

In some use cases, it might be impractical to bundle all sensitive information into a single secret, or even it is not possible to fully know a given secret name. In such cases, it is possible that a user might need to sync multiple secrets from an external provider into a single Kubernetes Secret. This is possible to be done in external-secrets with the `dataFrom.find` option.

!!! note
    The secret's contents as defined in the provider are going to be stored in the kubernetes secret as a single key. Currently, it's possible to apply a decoding Strategy during a find operation, but only at the secret level (e.g. if a secret is a JSON with some B64 encoded data within, `decodingStrategy: Auto` would not decode it)


### Fetching secrets matching a given name pattern
To fetch multiple secrets matching a name pattern from a common SecretStore you can apply the following manifest:
```yaml
{% include 'getallsecrets-find-by-name.yaml' %}
```

Suppose we contain secrets `/path/key1`, `key2/path`, and `path/to/keyring` with their respective values. The above YAML will produce the following kubernetes Secret:

```yaml
_path_key1: Cg==
key2_path: Cg==
path_to_keyring: Cg==
```
### Fetching secrets matching a set of metadata tags
To fetch multiple secrets matching a name pattern from a common SecretStore you can apply the following manifest:
```yaml
{% include 'getallsecrets-find-by-tags.yaml' %}
```
This will match any secrets containing all of the metadata labels in the `tags` parameter. At least one tag must be provided in order to allow finding secrets by metadata tags.


### Searching only in a given path
Some providers support filtering out a find operation only to a given path, instead of the root path. In order to use this feature, you can pass `find.path` to filter out these secrets into only this path, instead of the root path.

### Avoiding name conflicts
By default, kubernetes Secrets accepts only a given range of characters. `Find` operations will automatically replace any not allowed character with a `_`. So if we have a given secret `a_c` and `a/c` would lead to a naming conflict.


If you happen to have a case where a conflict is happening, you can use the `rewrite` block to apply a regexp on one of the find operations (for more information please refer to [Rewriting Keys from DataFrom](datafrom-rewrite.md)).

You can also set  `dataFrom.find.conversionStrategy: Unicode` to reduce the collistion probability. When using `Unicode`, any invalid character will be replaced by its unicode, in the form of `_UXXXX_`. In this case, the available kubernetes keys would be `a_c` and `a_U2215_c`, hence avoiding most of possible conflicts.



!!! note "PRs welcome"
    Some providers might not have the implementation needed for fetching multiple secrets. If that's your case, please feel free to contribute!
