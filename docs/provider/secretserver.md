# Delinea Secret Server

External Secrets Operator integration with [Delinea Secret Server](https://docs.delinea.com/online-help/secret-server/start.htm).

### Creating a SecretStore

You need a username, password and a fully qualified Secret Server tenant URL to authenticate
i.e. `https://yourTenantName.secretservercloud.com`.

Both username and password can be specified either directly in your `SecretStore` yaml config, or by referencing a kubernetes secret.

To acquire a username and password, refer to the  Secret Server [user management](https://docs.delinea.com/online-help/secret-server/users/creating-users/index.htm) documentation.

Both `username` and `password` can either be specified directly via the `value` field (example below)
>spec.provider.secretserver.username.value: "yourusername"<br />
spec.provider.secretserver.password.value: "yourpassword" <br />

Or you can reference a kubernetes secret (password example below).

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secret-server-store
spec:
  provider:
    secretserver:
      serverURL: "https://yourtenantname.secretservercloud.com"
      username:
        value: "yourusername"
      password:
        secretRef:
          name: <NAME_OF_K8S_SECRET>
          key: <KEY_IN_K8S_SECRET>
```

### Referencing Secrets

Secrets may be referenced by secret ID or secret name.
>Please note if using the secret name
the name field must not contain spaces or control characters.<br />
If multiple secrets are found, *`only the first found secret will be returned`*.

Please note: `Retrieving a specific version of a secret is not yet supported.`

Note that because all Secret Server secrets are JSON objects, you must specify the `remoteRef.property`
in your ExternalSecret configuration.<br />
You can access nested values or arrays using [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md).

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
    name: secret-server-external-secret
spec:
    refreshInterval: 1h
    secretStoreRef:
        kind: SecretStore
        name: secret-server-store
    data:
      - secretKey: SecretServerValue #<SECRET_VALUE_RETURNED_HERE>
        remoteRef:
          key: "52622" #<SECRET_ID>
          property: "array.0.value" #<GJSON_PROPERTY> * an empty property will return the entire secret
```

### Support for Non-JSON Secret Templates

The Secret Server provider now supports secrets that are not formatted as JSON. This enhancement allows users to retrieve and utilize secrets stored in formats such as plain text, XML, or other non-JSON structures without requiring additional parsing or transformation.​

When working with non-JSON secrets, you can omit the remoteRef.property field in your ExternalSecret configuration. The entire content of the secret will be retrieved and stored as-is in the corresponding Kubernetes Secret.​

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
    name: secret-server-external-secret
spec:
    refreshInterval: 1h
    secretStoreRef:
      kind: SecretStore
      name: secret-server-store
    data:
      - secretKey: SecretServerValue
        remoteRef:
          key: "52622"  # Secret ID
```

In this example, the secret with ID 52622 is retrieved in its entirety and stored under the key SecretServerValue in the Kubernetes Secret.​

This feature simplifies the integration process for applications that require secrets in specific formats, eliminating the need for custom parsing logic within your applications.

### Support for Fetching Secrets by Path

In addition to retrieving secrets by ID or Name, the Secret Server provider now supports fetching secrets by **path**.  
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

The path must exactly match the folder and secret name in Secret Server.
If multiple secrets with the same name exist in different folders, the path helps to uniquely identify the correct one.
You can still use property to extract values from JSON-formatted secrets or omit it to retrieve the entire secret (JSON or non-JSON).
Updated Referencing Secrets Section

Secrets may be referenced by:
>Secret ID<br />
Secret Name<br />
Secret Path (/FolderName/SecretName)<br />

Please note if using the secret name or path,
the field must not contain spaces or control characters.<br />
If multiple secrets are found, only the first found secret will be returned.

Please note: Retrieving a specific version of a secret is not yet supported.

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
