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
Here, "secret-to-be-created" is the name of Kubernetes secrets that will be created.   
when we use 

```
  dataFrom:
  - extract:
      key: all-keys-example-secret
```
We get all the key-value pairs present over the remote secret store (GCP or AWS or Azure) and can pass either all or a few key-values as environment variables.    
Please note that, "all-keys-example-secret" is the name of your secret present on GCP/AWS secrets manager/Azure    
    
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
 
Here,    
\<key1\> and \<key2> are the names of keys that will be created and passed as env variables.    
\<example\>: example is the name of your external secret created by you.    
\<username\> and \<password>: is the particular key in the secrets manager whose value you want to pass.    
To check both values we can run:
```
kubectl get secret secret-to-be-created -n <namespace> -o jsonpath='{.data.username}' | base64 -d
kubectl get secret secret-to-be-created -n <namespace> -o jsonpath='{.data.surname}' | base64 -d
```

Also, if you have a large number of secrets and you want to pass all of them as enviromnent variables, then either you can replicate the above steps in your deployment file for all the keys or you can use the envFrom block as below:    

```
    spec:
      containers:
      - command:
        - mkdir abc.sh
        envFrom:
        - secretRef:
            name: example
```
