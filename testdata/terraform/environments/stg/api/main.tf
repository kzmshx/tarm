module "network" {
  source   = "../../../modules/network"
  vpc_cidr = "10.1.0.0/16"
}

module "database" {
  source = "../../../modules/database"
  vpc_id = module.network.vpc_id
}

module "auth" {
  source     = "../../../modules/auth"
  enable_mfa = true
}

terraform {
  backend "s3" {
    bucket = "terraform-state-stg"
    key    = "api/terraform.tfstate"
  }
}