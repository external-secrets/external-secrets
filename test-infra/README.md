# Test Infra


## Setup cloud provider
``` bash
# prepare your AWS credentials
export AWS_PROFILE=xyz

terraform init
terraform plan
terraform apply
```


## External-Secrets Operator | Cloud Integration

AWS: Place a file with the following content in `./testfiles/aws/.credentials`

```
AWS_ACCESS_KEY_ID=XXXXXX
AWS_SECRET_ACCESS_KEY=YYYYYYY
```
