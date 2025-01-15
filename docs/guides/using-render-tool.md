# Using the Render tool

The tool can be found under `cmd/render`. This command can be used to test templates for `PushSecret` and `ExternalSecret`.

To run `render` simply execute `make build` in the `cmd/render` folder. This will result in a binary under `cmd/render/bin`.

Once the build succeeds, the command can be used as such:
```console
./bin/render --source-templated-object template-test/push-secret.yaml --source-secret-data-file template-test/secret.yaml
```

Where template-test looks like this:

```
❯ tree template-test/                                                                                                                                                                                                                   (base)
template-test/
├── push-secret.yaml
└── secret.yaml

1 directory, 2 files
```

`PushSecret` is simply the following:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: example-push-secret-with-template
spec:
  refreshInterval: 10s
  secretStoreRefs:
    - name: secret-store-name
      kind: SecretStore
  selector:
    secret:
      name: git-sync-secret
  template:
    engineVersion: v2
    data:
      token: "{{ .token | toString | upper }} was templated"
  data:
    - match:
        secretKey: token
        remoteRef:
          remoteKey: git-sync-secret-copy-templated
          property: token
```

And secret data is:

```yaml
token: dG9rZW4=
```

Therefor if there is a PushSecret or an ExternalSecret object that the user would like to test the template for,
simply put it into a file along with the data it's using, and run this command.

The output will be something like this:

```console
➜ ./bin/render --source-templated-object template-test/push-secret.yaml --source-secret-data-file template-test/secret.yaml                                                                                                          
data:
  token: VE9LRU4gd2FzIHRlbXBsYXRlZA==
metadata:
  creationTimestamp: null

➜ echo -n "VE9LRU4gd2FzIHRlbXBsYXRlZA==" | base64 -d                                                                                                                                                                                    
TOKEN was templated⏎
```

Further options can be used to provide templates from a ConfigMap or a Secret:
```
➜ ./bin/render --source-templated-object template-test/push-secret.yaml \
    --source-secret-data-file template-test/secret.yaml \
    --template-from-config-map template-test/template-config-map.yaml \
    --template-from-secret template-test/template-secret.yaml
```
