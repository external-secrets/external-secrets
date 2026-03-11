# Delinea Secret-Server/Platform

For detailed information about configuring  Kubernetes ESO with Secret Server and the Delinea Platform, see the https://docs.delinea.com/online-help/integrations/external-secrets/kubernetes-eso-secret-server.htm

### Creating a SecretStore

You need a username, password and a fully qualified Secret-Server/Platform tenant URL to authenticate
i.e. `https://yourTenantName.secretservercloud.com` or `https://yourtenantname.delinea.app`.

Both username and password can be specified either directly in your `SecretStore` yaml config, or by referencing a kubernetes secret.

Both `username` and `password` can either be specified directly via the `value` field (example below)
>spec.provider.secretserver.username.value: "yourusername"<br />
spec.provider.secretserver.password.value: "yourpassword" <br />

Or you can reference a kubernetes secret (password example below).

**Note:** Use `https://yourtenantname.secretservercloud.com` for Secret Server or `https://yourtenantname.delinea.app` for Platform.
```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secret-server-store
spec:
  provider:
    secretserver:
      serverURL: "https://yourtenantname.secretservercloud.com"  # or "https://yourtenantname.delinea.app" for Platform
      username:
        value: "yourusername"
      password:
        secretRef:
          name: <NAME_OF_K8S_SECRET>
          key: <KEY_IN_K8S_SECRET>
```

### Referencing Secrets

Secrets may be referenced by:
>Secret ID<br />
Secret Name<br />
Secret Path (/FolderName/SecretName)<br />

Please note if using the secret name or path,
the name field must not contain spaces or control characters.<br />
If multiple secrets are found, *`only the first found secret will be returned`*.

Please note: `Retrieving a specific version of a secret is not yet supported.`

Note that because all Secret-Server/Platform secrets are JSON objects, you must specify the `remoteRef.property`
in your ExternalSecret configuration.<br />
You can access nested values or arrays using [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md).

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
    name: secret-server-external-secret
spec:
    refreshInterval: 1h0m0s
    secretStoreRef:
        kind: SecretStore
        name: secret-server-store
    data:
      - secretKey: SecretServerValue #<SECRET_VALUE_RETURNED_HERE>
        remoteRef:
          key: "52622" #<SECRET_ID>
          property: "array.0.value" #<GJSON_PROPERTY> * an empty property will return the entire secret
```

### Working with Plain Text ItemValue Fields

While Secret-Server/Platform always returns secrets in JSON format with an `Items` array structure, individual field values (stored in `ItemValue`) may contain plain text, passwords, URLs, or other non-JSON content.

When retrieving fields that contain plain text values, you can reference them directly by their `FieldName` or `Slug` without needing additional JSON parsing within the `ItemValue`.

#### Example with Plain Text Password Field

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
    name: secret-server-external-secret
spec:
    refreshInterval: 1h0m0s
    secretStoreRef:
      kind: SecretStore
      name: secret-server-store
    data:
      - secretKey: password
        remoteRef:
          key: "52622"      # Secret ID
          property: "password"  # FieldName or Slug of the password field
```

In this example, if the secret contains an Item with `FieldName: "Password"` or `Slug: "password"`, the plain text value stored in `ItemValue` is retrieved directly and stored under the key `password` in the Kubernetes Secret.

This approach works for any field type (text, password, URL, etc.) where the `ItemValue` contains simple content rather than nested JSON structures.

### Support for Fetching Secrets by Path

In addition to retrieving secrets by ID or Name, the Secret-Server/Platform provider now supports fetching secrets by **path**.
This allows you to specify a secret’s folder hierarchy and name in the format:
>/FolderName/SecretName

#### Example

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: secret-server-external-secret
spec:
  refreshInterval: 15s
  secretStoreRef:
    kind: SecretStore
    name: secret-server-store
  data:
    - secretKey: SecretServerValue  # Key in the Kubernetes Secret
      remoteRef:
        key: "/secretFolder/secretname"  # Path format: /<Folder>/<SecretName>
        property: ""                    # Optional: use gjson syntax to extract a specific field
```

#### Notes:

The path must exactly match the folder and secret name in Secret-Server/Platform.
If multiple secrets with the same name exist in different folders, the path helps to uniquely identify the correct one.
You can still use property to extract values from JSON-formatted secrets or omit it to retrieve the entire secret.

### Preparing your secret
You can either retrieve your entire secret or you can use a JSON formatted string
stored in your secret located at Items[0].ItemValue to retrieve a specific value.<br />
See example JSON secret below.

#### Examples
Using the json formatted secret below:

- Lookup a single top level property using secret ID.

>spec.data.remoteRef.key = 52622 (id of the secret)<br />
spec.data.remoteRef.property = "user" (Items.0.ItemValue user attribute)<br />
returns: marktwain@hannibal.com

- Lookup a nested property using secret name.

>spec.data.remoteRef.key = "external-secret-testing" (name of the secret)<br />
spec.data.remoteRef.property = "books.1" (Items.0.ItemValue books.1 attribute)<br />
returns: huckleberryFinn

- Lookup by secret ID (*secret name will work as well*) and return the entire secret.

>spec.data.remoteRef.key = "52622" (id of the secret)<br />
spec.data.remoteRef.property = "" <br />
returns: The entire secret in JSON format as displayed below


```JSON
{
  "Name": "external-secret-testing",
  "FolderID": 73,
  "ID": 52622,
  "SiteID": 1,
  "SecretTemplateID": 6098,
  "SecretPolicyID": -1,
  "PasswordTypeWebScriptID": -1,
  "LauncherConnectAsSecretID": -1,
  "CheckOutIntervalMinutes": -1,
  "Active": true,
  "CheckedOut": false,
  "CheckOutEnabled": false,
  "AutoChangeEnabled": false,
  "CheckOutChangePasswordEnabled": false,
  "DelayIndexing": false,
  "EnableInheritPermissions": true,
  "EnableInheritSecretPolicy": true,
  "ProxyEnabled": false,
  "RequiresComment": false,
  "SessionRecordingEnabled": false,
  "WebLauncherRequiresIncognitoMode": false,
  "Items": [
    {
      "ItemID": 280265,
      "FieldID": 439,
      "FileAttachmentID": 0,
      "FieldName": "Data",
      "Slug": "data",
      "FieldDescription": "json text field",
      "Filename": "",
      "ItemValue": "{ \"user\": \"marktwain@hannibal.com\", \"occupation\": \"author\",\"books\":[ \"tomSawyer\",\"huckleberryFinn\",\"Pudd'nhead Wilson\"] }",
      "IsFile": false,
      "IsNotes": false,
      "IsPassword": false
    }
  ]
}
```

### Referencing Secrets in multiple Items secrets

If there is more then one Item in the secret, it supports to retrieve them (all Item.\*.ItemValue) looking up by Item.\*.FieldName or Item.\*.Slug, instead of the above behaviour to use gjson only on the first item Items.0.ItemValue only.

#### Examples

Using the json formatted secret below:

- Lookup a single top level property using secret ID.

>spec.data.remoteRef.key = 4000 (id of the secret)<br />
spec.data.remoteRef.property = "Username" (Items.0.FieldName)<br />
returns: usernamevalue

- Lookup a nested property using secret name.

>spec.data.remoteRef.key = "Secretname" (name of the secret)<br />
spec.data.remoteRef.property = "password" (Items.1.slug)<br />
returns: passwordvalue

- Lookup by secret ID (*secret name will work as well*) and return the entire secret.

>spec.data.remoteRef.key = "4000" (id of the secret)<br />
returns: The entire secret in JSON format as displayed below


```JSON
{
  "Name": "Secretname",
  "FolderID": 0,
  "ID": 4000,
  "SiteID": 0,
  "SecretTemplateID": 0,
  "LauncherConnectAsSecretID": 0,
  "CheckOutIntervalMinutes": 0,
  "Active": false,
  "CheckedOut": false,
  "CheckOutEnabled": false,
  "AutoChangeEnabled": false,
  "CheckOutChangePasswordEnabled": false,
  "DelayIndexing": false,
  "EnableInheritPermissions": false,
  "EnableInheritSecretPolicy": false,
  "ProxyEnabled": false,
  "RequiresComment": false,
  "SessionRecordingEnabled": false,
  "WebLauncherRequiresIncognitoMode": false,
  "Items": [
    {
      "ItemID": 0,
      "FieldID": 0,
      "FileAttachmentID": 0,
      "FieldName": "Username",
      "Slug": "username",
      "FieldDescription": "",
      "Filename": "",
      "ItemValue": "usernamevalue",
      "IsFile": false,
      "IsNotes": false,
      "IsPassword": false
    },
    {
      "ItemID": 0,
      "FieldID": 0,
      "FileAttachmentID": 0,
      "FieldName": "Password",
      "Slug": "password",
      "FieldDescription": "",
      "Filename": "",
      "ItemValue": "passwordvalue",
      "IsFile": false,
      "IsNotes": false,
      "IsPassword": false
    }
  ]
}
```

### Pushing Secrets

The Delinea Secret-Server/Platform provider supports pushing secrets from Kubernetes back to your Secret Server instance using the `PushSecret` resource. You can both create new secrets and update existing ones.

#### Requirements for Creating New Secrets

When creating a **new** secret in Secret Server, you must provide a `folderId` and a `secretTemplateId`. These are passed as `metadata` in the `PushSecret` spec:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-secret-example
spec:
  refreshInterval: 1h
  secretStoreRefs:
    - name: secret-server-store
      kind: SecretStore
  selector:
    secret:
      name: my-k8s-secret
  data:
    - match:
        secretKey: username
        remoteRef:
          remoteKey: my-new-secret
          property: username # Maps to the 'Username' field/slug in Secret Server
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          folderId: 73 # Required for new secrets
          secretTemplateId: 6098 # Required for new secrets
    - match:
        secretKey: password
        remoteRef:
          remoteKey: my-new-secret
          property: password # Maps to the 'Password' field/slug in Secret Server
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          folderId: 73
          secretTemplateId: 6098
```

#### Updating Existing Secrets

When updating an existing secret, you do not strictly need the `folderId` or `secretTemplateId` metadata, as the provider will fetch the existing secret by its name or ID to update the corresponding fields.

#### Deletion Behavior

The `PushSecret` resource allows you to configure what happens to the remote secret in Secret Server when the `PushSecret` itself is deleted, via the `PushSecret.spec.deletionPolicy` field. Supported values are:
- `Retain`: (Default) The remote secret is left intact in Secret Server when the `PushSecret` is deleted.
- `Delete`: The provider will attempt to delete the remote secret from Secret Server when the `PushSecret` is removed.

When `Delete` is specified, the deletion operation is idempotent; if the secret has already been removed or cannot be found, the provider will safely ignore the error and proceed. Note that this deletion uses the exact remote key (ID, name, or path) configured in the `remoteRef` to match and remove the secret.

#### Pushing JSON Payloads

If your Kubernetes secret contains a full JSON payload that you want to push into a single text field in Secret Server (like a `Data` or `Notes` field), you can omit the `property` in the `remoteRef`. The provider will place the complete JSON representation of your secret into the **first** field of the Secret Server secret.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-secret-json-example
spec:
  refreshInterval: 1h
  secretStoreRefs:
    - name: secret-server-store
      kind: SecretStore
  selector:
    secret:
      name: my-k8s-json-secret
  data:
    - match:
        secretKey: config # The key in your k8s secret containing the JSON
        remoteRef:
          remoteKey: my-new-json-secret
          # property is omitted to store the value in the first available field
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          folderId: 73
          secretTemplateId: 6098
```
