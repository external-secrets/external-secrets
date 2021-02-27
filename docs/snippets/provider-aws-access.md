## AWS Authentication

Access to AWS providers can be granted in various ways:

* [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html): IAM roles for service accounts.
* Per pod IAM authentication: [kiam](https://github.com/uswitch/kiam) or [kube2iam](https://github.com/jtblin/kube2iam).
* Directly provide AWS credentials to the External Secrets Operator pod by using environment variables.

Additionally, before fetching a secret from a store, ESO is able to assume role (as a proxy so to speak). It is advisable to use multiple roles in a multi-tenant environment.


You can limit the range of roles which can be assumed by this particular namespace by using annotations on the namespace resource. The annotation value is evaluated as a regular expression.

!!! bug "Not implemented"
    This is currently **not** implemented. Feel free to contribute.

``` yaml
kind: Namespace
metadata:
  name: iam-example
  annotations:
    # annotation key is configurable
    iam.amazonaws.com/permitted: "arn:aws:iam::123456789012:role/foo.*"
```
