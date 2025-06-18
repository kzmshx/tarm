# Root module in deep directory
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  
  tags = {
    Name = "shared-vpc"
  }
}

# Uses reusable module from different location
module "logging" {
  source = "../../../shared-components/logging"
  
  vpc_id = aws_vpc.main.id
}