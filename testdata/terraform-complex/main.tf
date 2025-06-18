# Root module at repository root
terraform {
  required_version = ">= 1.0"
}

# Uses nested reusable module
module "shared_vpc" {
  source = "./stacks/shared/vpc"
}