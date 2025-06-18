# Team-based root module structure
module "shared_logging" {
  source = "../../shared-components/logging"
  
  vpc_id = "vpc-team-a"
}

# Cross-team dependency
module "shared_utils" {
  source = "../../utils/common"
  
  environment = "team-a-project-x"
}

resource "aws_s3_bucket" "project_data" {
  bucket = "team-a-project-x-data"
  
  tags = module.shared_utils.common_tags
}