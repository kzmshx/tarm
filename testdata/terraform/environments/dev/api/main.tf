module "network" {
  source   = "../../../modules/network"
  vpc_cidr = "10.0.0.0/16"
}

module "database" {
  source = "../../../modules/database"
  vpc_id = module.network.vpc_id
}

terraform {
  backend "s3" {
    bucket = "terraform-state-dev"
    key    = "api/terraform.tfstate"
  }
}