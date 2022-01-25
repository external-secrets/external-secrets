terraform {
  required_version = ">= 0.13"

   backend "s3" {
     bucket = "eso-e2e-aws-tfstate"
     key    = "aws-tfstate"
     region = "eu-west-1"
   }

  required_providers {}
}
