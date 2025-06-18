# Invalid Terraform syntax
module "test" {
  source = "./nonexistent"
  
  # Missing closing brace
  variable = {
    name = "test"
    # Missing }

# Invalid block without closing
resource "aws_instance" "test" {
  ami = "ami-12345"
  # Missing closing brace