# Application root module
module "logging" {
  source = "../../shared-components/logging"
  
  vpc_id = "vpc-12345"
}

module "utils" {
  source = "../../utils/common"
  
  environment = "production"
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t3.micro"
  
  tags = module.utils.common_tags
}