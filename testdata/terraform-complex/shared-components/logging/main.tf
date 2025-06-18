# Reusable module in non-standard location
variable "vpc_id" {
  description = "VPC ID for logging resources"
  type        = string
}

# Depends on utility module
module "utils" {
  source = "../../utils/common"
  
  environment = "shared"
}

resource "aws_cloudwatch_log_group" "main" {
  name = "/aws/vpc/${var.vpc_id}"
  
  tags = module.utils.common_tags
}