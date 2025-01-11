# Using the Render tool

ESO has a command called `render-template`. This command can be used to test templates for `PushSecret` and `ExternalSecret`.

To run `render-template` simply execute `make build` or by `go build main.go -o bin/external-secrets` in the root folder of ESO.

Once the build succeeds, the command can be used as such:
```console
./bin/external-secrets render-template --source-templated-object template-test/push-secret.yaml --source-secret-data-file template-test/secret.yaml
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
➜ ./bin/external-secrets render-template --source-templated-object template-test/push-secret.yaml --source-secret-data-file template-test/secret.yaml                                                                                                          
data:
  token: VE9LRU4gd2FzIHRlbXBsYXRlZA==
metadata:
  creationTimestamp: null

➜ echo -n "VE9LRU4gd2FzIHRlbXBsYXRlZA==" | base64 -d                                                                                                                                                                                    
TOKEN was templated⏎
```

Further options can be used to provide templates from a ConfigMap or a Secret:
```
➜ ./bin/external-secrets render-template --source-templated-object template-test/push-secret.yaml \
    --source-secret-data-file template-test/secret.yaml \
    --template-from-config-map template-test/template-config-map.yaml \
    --template-from-secret template-test/template-secret.yaml
```
