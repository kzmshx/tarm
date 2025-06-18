# Standalone module with no local dependencies

resource "aws_s3_bucket" "standalone" {
  bucket = "standalone-bucket"
}

# External module - should be ignored
module "external_vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
  
  name = "standalone-vpc"
  cidr = "10.3.0.0/16"
}

terraform {
  backend "s3" {
    bucket = "terraform-state-standalone"
    key    = "simple/terraform.tfstate"
  }
}