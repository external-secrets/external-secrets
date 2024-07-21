# ESO Blogs

A list of blogs written by people all over the community. Feel free to let us know if you are writing about ESO at some place! We would be happy to mention you here!

## [Comparing External Secrets Operator with Secret Storage CSI as Kubernetes External Secrets is Deprecated](https://mixi-developers.mixi.co.jp/compare-eso-with-secret-csi-402bf37f20bc)

@riddle writes about choosing ESO when comparing with Secret Store CSI Driver in their specific use case. They show us the relevant differences between the projects when looking at their scenario and requirements while integrating with ArgoCD. [Comparing External Secrets Operator with Secret Storage CSI as Kubernetes External Secrets is Deprecated](https://mixi-developers.mixi.co.jp/compare-eso-with-secret-csi-402bf37f20bc)

## [Tutorial: Getting Started with External Secrets Operator on Kubernetes using AWS Secrets Manager](https://ptuladhar3.medium.com/getting-started-with-external-secrets-operator-on-kubernetes-using-aws-secrets-manager-6dc403d9630c)

Puru writes about getting started using ESO with AWS Secrets Manager. He uses illustrations to explain ESO to new users and get's you to quickly start using ESO, as article is easy to follow along. [Getting Started with External Secrets Operator on Kubernetes using AWS Secrets Manager](https://ptuladhar3.medium.com/getting-started-with-external-secrets-operator-on-kubernetes-using-aws-secrets-manager-6dc403d9630c)

## [Tutorial: How to Set External-Secrets with Azure KeyVault](https://blog.container-solutions.com/tutorial-external-secrets-with-azure-keyvault)

Gustavo writes about how to setup ESO with Azure Key Vault and adds an guide on how to make it a bit more secure with OPA (Open Policy Agent). [How to Set External-Secrets with Azure KeyVault](https://blog.container-solutions.com/tutorial-external-secrets-with-azure-keyvault)

## [Tutorial: How to Set External-Secrets with GCP Secret Manager](https://blog.container-solutions.com/tutorial-how-to-set-external-secrets-with-gcp-secret-manager)

Gustavo writes about how to setup ESO with GCP Secret Manager. He also shows you how to make a simple multi tenant setup with a ClusterSecretStore. [How to Set External-Secrets with GCP Secret Manager](https://blog.container-solutions.com/tutorial-how-to-set-external-secrets-with-gcp-secret-manager)

## [Tutorial: How to Set External-Secrets with Hashicorp Vault](https://blog.container-solutions.com/tutorialexternal-secrets-with-hashicorp-vault)

Gustavo writes about how to setup ESO with Hashicorp Vault. He also shows you how to make this scale with multiple replicas of the operator and leader election enabled to lead balance handling synchronization work. [How to Set External-Secrets with Hashicorp Vault](https://blog.container-solutions.com/tutorialexternal-secrets-with-hashicorp-vault)

## [Tutorial: How to Set External-Secrets with AWS](https://blog.container-solutions.com/tutorial-how-to-set-external-secrets-with-aws)

Gustavo writes about how to setup ESO with AWS Secrets Manager. He also shows you how to limit access and give granular permissions with better policies and roles for your service accounts to use. [How to Set External-Secrets with AWS](https://blog.container-solutions.com/tutorial-how-to-set-external-secrets-with-aws)

## [Tutorial: How to Set External-Secrets with IBM Secrets Manager](https://0x58.medium.com/ibm-cloud-secrets-manager-and-the-external-secrets-operator-1c94234993b6)

In this multi-articles series, Xavier writes about how to setup ESO with IBM Secrets Manager using the web user-interface. Xavier also shares how it is integrated into his pipeline scripts. [How to Set External-Secrets with IBM Secrets Manager](https://0x58.medium.com/ibm-cloud-secrets-manager-and-the-external-secrets-operator-1c94234993b6)


## [Kubernetes Hardening Tutorial Part 2: Network](https://blog.gitguardian.com/kubernetes-tutorial-part-2-network/)

Tiexin Guo Writes about Kubernetes hardening in this series of blogs. He mentions ESO as one of the convenient options when dealing with secrets in Kubernetes, and how to use it with AWS Secret Manager using AWS credentials. [Kubernetes Hardening Tutorial Part 2: Network](https://blog.gitguardian.com/kubernetes-tutorial-part-2-network/)


## [Tutorial: How to manage secrets in OpenShift using Vault and External Secrets Operator](https://youtu.be/N7njTq6TSx8)

Balkrishna Pandey published a video tutorial and a [blog post](https://goglides.io/how-to-manage-secrets-in-openshift-using-vault-and-external-secrets/1164/) on integrating HashiCorp Vault and External Secret Operator (ESO) to manage application secrets on OpenShift Cluster. In this blog, he demonstrates the strength of the `ClusterSecretStore` functionality, a cluster scoped SecretStore and is global to the Cluster that all `ExternalSecrets` can reference from all namespaces.

## [Tutorial: Leverage AWS secrets stores from EKS Fargate with External Secrets Operator](https://aws.amazon.com/blogs/containers/leverage-aws-secrets-stores-from-eks-fargate-with-external-secrets-operator/)

In this AWS Containers blog post, Ryan writes about how to leverage External Secret Operator with an EKS Fargate cluster using IAM Roles for Service Accounts (IRSA). This setup supports the requirements of Fargate based workloads. [Leverage AWS secrets stores from EKS Fargate with External Secrets Operator](https://aws.amazon.com/blogs/containers/leverage-aws-secrets-stores-from-eks-fargate-with-external-secrets-operator/)

## [Cloud Native Secret Management with External Secrets Operator](https://eminalemdar.medium.com/cloud-native-secret-management-with-external-secrets-operator-2912f41f9c49)

Emin writes about what problems ESO can solve and how to setup ESO on an Amazon EKS Cluster with integrations for AWS Secrets Manager using IAM Roles for Service Accounts (IRSA). In this blog post, there is also a GitHub repository with example codes for everyone to follow this demonstration.

## [External Secrets Operator Integration with HashiCorp Vault](https://eminalemdar.medium.com/external-secrets-operator-integration-with-hashicorp-vault-aff3f956237b)

Emin writes about integration between External Secrets Operator and HashiCorp Vault with a demonstration on installing ESO and Vault on a Kubernetes Cluster and configuration of the permissions and other integration parts.

## [Reversing the Workflow with External Secrets Operator’s Push Secret Feature](https://medium.com/@eminalemdar/reversing-the-workflow-with-external-secrets-operators-push-secret-feature-f2a64f3db748)

Emin writes about the Push Secret feature of ESO and how this new feature reverse the workflow of ESO by pushing Kubernetes secrets to external secret management providers.

## [GCP Secret Manager with self-hosted Kubernetes](https://medium.com/@jjlakis/gcp-secret-manager-with-self-hosted-kubernetes-db35d01d65f0)

Jacek writes about bringing GCP secrets to on-premises cluster through External Secrets Operator intergration with workload identity.

## [Injecting AWS Secrets in a Kubernetes Cluster with External Secrets Operator](https://blog.devops.dev/injecting-external-secrets-in-a-kubernetes-cluster-1e9bbe0f0d5b)

Ali writes about integrating AWS Secrets Manager and Parameter Store secrets within an EKS Cluster using ESO. He shows a quick setup of the operator, and how to fetch secrets in a repeatable fashion. The guide is bundled with cool illustrations and code snippets that describe the ESO architecture and injection process

## [Encoding & Decoding Kubernetes Secrets — ESO Advanced Templating](https://blog.devops.dev/encoding-decoding-kubernetes-secrets-externalsecrets-operator-826b9680df63)

Here, Ali briefly introduces templates within ESO and describes some use cases where templating can be crucial. Code snippets are included where needed too.