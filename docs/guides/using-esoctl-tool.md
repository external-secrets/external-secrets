# Using the esoctl tool

The tool can be found under `cmd/esoctl`.


## Debugging templates

 The `template` command can be used to test templates for `PushSecret` and `ExternalSecret`.

To run render simply execute `make build` in the `cmd/esoctl` folder. This will result in a binary under `cmd/esoctl/bin`.

Once the build succeeds, the command can be used as such:

```
bin/esoctl template --source-templated-object template-test/push-secret.yaml --source-secret-data-file template-test/secret.yaml
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
{% include 'esoctl-tool-push-secret-snippet.yaml' %}
```

And secret data is:

```yaml
token: dG9rZW4=
```

Therefore if there is a PushSecret or an ExternalSecret object that the user would like to test the template for,
simply put it into a file along with the data it's using, and run this command.

The output will be something like this:

```
bin/esoctl template --source-templated-object template-test/push-secret.yaml --source-secret-data-file template-test/secret.yaml
data:
  token: VE9LRU4gd2FzIHRlbXBsYXRlZA==
metadata:
  creationTimestamp: null

echo -n "VE9LRU4gd2FzIHRlbXBsYXRlZA==" | base64 -d
TOKEN was templated⏎
```

Further options can be used to provide templates from a ConfigMap or a Secret:
```
bin/esoctl template --source-templated-object template-test/push-secret.yaml \
  --source-secret-data-file template-test/secret.yaml \
  --template-from-config-map template-test/template-config-map.yaml \
  --template-from-secret template-test/template-secret.yaml
```

## Bootstrapping generator code

The `bootstrap generator` command can be used to create a new generator.

When running it, it will automatically:

- Bootstrap a new generator CRD
- Bootstrap a new generator implementation
- Update the register file with the new generator
- Update Cluster Generators to include the new generator
- Update needed dependencies  (go.mod, resolver file, etc)

To run, simply execute:
```
bin/esoctl bootstrap generator --name GeneratorName --description "A description of this generator" --package generatorname
```

### Example
```
bin/esoctl bootstrap generator --name MyAwesomeGenerator --description "An awesome generator I want to add to ESO :)"

✓ Created CRD: /home/gusfcarvalho/Documents/repos/external-secrets/apis/generators/v1alpha1/types_myawesomegenerator.go
✓ Created implementation: /home/gusfcarvalho/Documents/repos/external-secrets/generators/v1/myawesomegenerator/myawesomegenerator.go
✓ Created test file: /home/gusfcarvalho/Documents/repos/external-secrets/generators/v1/myawesomegenerator/myawesomegenerator_test.go
✓ Created go.mod: /home/gusfcarvalho/Documents/repos/external-secrets/generators/v1/myawesomegenerator/go.mod
✓ Created go.sum: /home/gusfcarvalho/Documents/repos/external-secrets/generators/v1/myawesomegenerator/go.sum
✓ Updated register file: /home/gusfcarvalho/Documents/repos/external-secrets/pkg/register/generators.go
✓ Updated types_cluster.go
✓ Updated main go.mod
✓ Updated resolver file: /home/gusfcarvalho/Documents/repos/external-secrets/runtime/esutils/resolvers/generator.go
✓ Updated register.go
✓ Successfully bootstrapped generator: MyAwesomeGenerator

Next steps:
1. Review and customize: apis/generators/v1alpha1/types_myawesomegenerator.go
2. Implement the generator logic in: generators/v1/myawesomegenerator/myawesomegenerator.go
3. Run: go mod tidy
4. Run: make generate
5. Run: make manifests
6. Add tests for your generator
```

You should also expect the following `git diff` with specific changes:

```diff
diff --git a/apis/generators/v1alpha1/register.go b/apis/generators/v1alpha1/register.go
index 16c05154b..9538bcc57 100644
--- a/apis/generators/v1alpha1/register.go
+++ b/apis/generators/v1alpha1/register.go
@@ -73,6 +73,9 @@ var (
        ClusterGeneratorKind = reflect.TypeOf(ClusterGenerator{}).Name()
        // CloudsmithAccessTokenKind is the kind name for CloudsmithAccessToken resource.
        CloudsmithAccessTokenKind = reflect.TypeOf(CloudsmithAccessToken{}).Name()
+
+       // MyAwesomeGeneratorKind is the kind name for MyAwesomeGenerator resource.
+       MyAwesomeGeneratorKind = reflect.TypeOf(MyAwesomeGenerator{}).Name()
 )
 
 func init() {
@@ -109,4 +112,5 @@ func init() {
        SchemeBuilder.Register(&Webhook{}, &WebhookList{})
        SchemeBuilder.Register(&Grafana{}, &GrafanaList{})
        SchemeBuilder.Register(&MFA{}, &MFAList{})
+       SchemeBuilder.Register(&MyAwesomeGenerator{}, &MyAwesomeGeneratorList{})
 }
diff --git a/apis/generators/v1alpha1/types_cluster.go b/apis/generators/v1alpha1/types_cluster.go
index e212dab76..0245e8f1c 100644
--- a/apis/generators/v1alpha1/types_cluster.go
+++ b/apis/generators/v1alpha1/types_cluster.go
@@ -30,7 +30,7 @@ type ClusterGeneratorSpec struct {
 }
 
 // GeneratorKind represents a kind of generator.
-// +kubebuilder:validation:Enum=ACRAccessToken;CloudsmithAccessToken;ECRAuthorizationToken;Fake;GCRAccessToken;GithubAccessToken;QuayAccessToken;Password;SSHKey;STSSessionToken;UUID;VaultDynamicSecret;Webhook;Grafana
+// +kubebuilder:validation:Enum=ACRAccessToken;CloudsmithAccessToken;ECRAuthorizationToken;Fake;GCRAccessToken;GithubAccessToken;QuayAccessToken;Password;SSHKey;STSSessionToken;UUID;VaultDynamicSecret;Webhook;Grafana;MyAwesomeGenerator
 type GeneratorKind string
 
 const (
@@ -64,6 +64,8 @@ const (
        GeneratorKindMFA GeneratorKind = "MFA"
        // GeneratorKindCloudsmithAccessToken represents a Cloudsmith access token generator.
        GeneratorKindCloudsmithAccessToken GeneratorKind = "CloudsmithAccessToken"
+       // GeneratorKindMyAwesomeGenerator represents a myawesomegenerator generator.
+       GeneratorKindMyAwesomeGenerator GeneratorKind = "MyAwesomeGenerator"
 )
 
 // GeneratorSpec defines the configuration for various supported generator types.
@@ -85,6 +87,7 @@ type GeneratorSpec struct {
        WebhookSpec               *WebhookSpec               `json:"webhookSpec,omitempty"`
        GrafanaSpec               *GrafanaSpec               `json:"grafanaSpec,omitempty"`
        MFASpec                   *MFASpec                   `json:"mfaSpec,omitempty"`
+       MyAwesomeGeneratorSpec             *MyAwesomeGeneratorSpec             `json:"myawesomegeneratorSpec,omitempty"`
 }
 
 // ClusterGenerator represents a cluster-wide generator which can be referenced as part of `generatorRef` fields.
diff --git a/go.mod b/go.mod
index ff95a9558..c73ecb0c7 100644
--- a/go.mod
+++ b/go.mod
@@ -14,6 +14,7 @@ replace (
        github.com/external-secrets/external-secrets/generators/v1/github => ./generators/v1/github
        github.com/external-secrets/external-secrets/generators/v1/grafana => ./generators/v1/grafana
        github.com/external-secrets/external-secrets/generators/v1/mfa => ./generators/v1/mfa
+       github.com/external-secrets/external-secrets/generators/v1/myawesomegenerator => ./generators/v1/myawesomegenerator
        github.com/external-secrets/external-secrets/generators/v1/password => ./generators/v1/password
        github.com/external-secrets/external-secrets/generators/v1/quay => ./generators/v1/quay
        github.com/external-secrets/external-secrets/generators/v1/sshkey => ./generators/v1/sshkey
diff --git a/pkg/register/generators.go b/pkg/register/generators.go
index dd9ad55fb..6aafd4089 100644
--- a/pkg/register/generators.go
+++ b/pkg/register/generators.go
@@ -34,6 +34,7 @@ import (
        uuid "github.com/external-secrets/external-secrets/generators/v1/uuid"
        vaultgen "github.com/external-secrets/external-secrets/generators/v1/vault"
        webhookgen "github.com/external-secrets/external-secrets/generators/v1/webhook"
+       myawesomegenerator "github.com/external-secrets/external-secrets/generators/v1/myawesomegenerator"
 )
 
 func init() {
@@ -53,4 +54,5 @@ func init() {
        genv1alpha1.Register(uuid.Kind(), uuid.NewGenerator())
        genv1alpha1.Register(vaultgen.Kind(), vaultgen.NewGenerator())
        genv1alpha1.Register(webhookgen.Kind(), webhookgen.NewGenerator())
+       genv1alpha1.Register(myawesomegenerator.Kind(), myawesomegenerator.NewGenerator())
 }
diff --git a/runtime/esutils/resolvers/generator.go b/runtime/esutils/resolvers/generator.go
index 66f4b4037..938ccd6cd 100644
--- a/runtime/esutils/resolvers/generator.go
+++ b/runtime/esutils/resolvers/generator.go
@@ -302,6 +302,17 @@ func clusterGeneratorToVirtual(gen *genv1alpha1.ClusterGenerator) (client.Object
                        },
                        Spec: *gen.Spec.Generator.MFASpec,
                }, nil
+       case genv1alpha1.GeneratorKindMyAwesomeGenerator:
+               if gen.Spec.Generator.MyAwesomeGeneratorSpec == nil {
+                       return nil, fmt.Errorf("when kind is %s, MyAwesomeGeneratorSpec must be set", gen.Spec.Kind)
+               }
+               return &genv1alpha1.MyAwesomeGenerator{
+                       TypeMeta: metav1.TypeMeta{
+                               APIVersion: genv1alpha1.SchemeGroupVersion.String(),
+                               Kind:       genv1alpha1.MyAwesomeGeneratorKind,
+                       },
+                       Spec: *gen.Spec.Generator.MyAwesomeGeneratorSpec,
+               }, nil
        default:
                return nil, fmt.Errorf("unknown kind %s", gen.Spec.Kind)
        }
```

### flags
#### name
Defines the generator name. Must be `PascalCase`.

#### description
Defines the generator description (added as a golang comment)

#### package (optional)
Defines the package name for the generator. Must be `snake_case`. defaults to lowercase of `name`