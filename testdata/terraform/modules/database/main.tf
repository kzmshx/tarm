variable "vpc_id" {
  type = string
}

module "common" {
  source = "../common"
}

resource "null_resource" "database" {
  provisioner "local-exec" {
    command = "echo 'Database module'"
  }
}