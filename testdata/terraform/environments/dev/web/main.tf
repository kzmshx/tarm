module "network" {
  source   = "../../../modules/network"
  vpc_cidr = "10.0.0.0/16"
}

module "auth" {
  source     = "../../../modules/auth"
  enable_mfa = false
}

terraform {
  backend "s3" {
    bucket = "terraform-state-dev"
    key    = "web/terraform.tfstate"
  }
}