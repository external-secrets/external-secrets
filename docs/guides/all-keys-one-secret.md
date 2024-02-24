# All Keys, One Secret

To get multiple key-values from an external secret, not having to worry about how many, or what these keys are, we have to use the dataFrom field of the ExternalSecret resource, instead of the data field. We will give an example here with the gcp provider (should work with other providers in the same way).

Please follow the authentication and SecretStore steps of the [Google Cloud Secrets Manager guide](../provider/google-secrets-manager.md) to setup access to your google cloud account first.

Then create a secret in Google Cloud Secret Manager that contains a JSON string with multiple key values like this:

![secret-value](../pictures/screenshot_json_string_gcp_secret_value.png)

Let's call this secret all-keys-example-secret on Google Cloud.
 

### Creating dataFrom external secret

Now, when creating our ExternalSecret resource, instead of using the data field, we use the dataFrom field:

```yaml
{% include 'gcpsm-data-from-external-secret.yaml' %}
```
<br>
Here, "example" is the name of Kubernetes secrets which will be created.
<br>
when we use 

```
  dataFrom:
  - extract:
      key: all-keys-example-secret
```
We get all the secrets present in the key-value pair in the secret manager(external secret store of AWS or GCP or Azure) in the form of key values and also we can pass either all or a few key-values as environment variables.

We can pass a few secrets as env variables as below:
```
        env:
          - name: key1
            valueFrom:
              secretKeyRef:
                name: example
                key: username

          - name: key2
            valueFrom:
              secretKeyRef:
                name: example
                key: surname
```
 
Here, <br>
\<key1\> and \<key2> are the names of keys that will be created and passed as env variables.
<br>
\<example\>: example is the name of your external secret created by you.
<br>
\<username\> and \<password>: is the particular key in the secrets manager whose value you want to pass.
<br>
To check both values we can run:
```
kubectl get secret secret-to-be-created -n <namespace> -o jsonpath='{.data.username}' | base64 -d
kubectl get secret secret-to-be-created -n <namespace> -o jsonpath='{.data.surname}' | base64 -d
```

<br>
Also, if you have a large number of secrets and you want to pass all of them as enviromnent variables, then either you can replicate the above steps in your deployment file for all the keys or you can use the envFrom block as below:
<br>

```
    spec:
      containers:
      - command:
        - mkdir abc.sh
        envFrom:
        - secretRef:
            name: example
```
