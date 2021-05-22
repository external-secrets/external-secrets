## AWS Authentication

Access to AWS providers can be granted in various ways:

* [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html): IAM roles for service accounts.
* Per pod IAM authentication: [kiam](https://github.com/uswitch/kiam) or [kube2iam](https://github.com/jtblin/kube2iam).
* Directly provide AWS credentials to the External Secrets Operator pod by using environment variables.

Additionally, before fetching a secret from a store, ESO is able to assume role (as a proxy so to speak). It is advisable to use multiple roles in a multi-tenant environment.
