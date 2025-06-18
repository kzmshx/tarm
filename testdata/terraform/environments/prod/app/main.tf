# Local module - should be tracked
module "network" {
  source   = "../../../modules/network"
  vpc_cidr = "10.2.0.0/16"
}

# Registry module - should be ignored
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
  
  name = "prod-vpc"
  cidr = "10.2.0.0/16"
}

# Git module - should be ignored  
module "security_group" {
  source = "git::https://github.com/terraform-aws-modules/terraform-aws-security-group.git?ref=v5.1.0"
  
  name = "prod-sg"
}

# S3 module - should be ignored
module "s3_module" {
  source = "s3::https://my-bucket.s3.amazonaws.com/modules/s3-module.zip"
  
  bucket_name = "prod-bucket"
}

# HTTP module - should be ignored
module "http_module" {
  source = "https://example.com/modules/http-module.zip"
  
  config = "prod"
}

terraform {
  backend "s3" {
    bucket = "terraform-state-prod"
    key    = "app/terraform.tfstate"
  }
}