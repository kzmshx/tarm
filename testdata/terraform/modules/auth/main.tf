module "common" {
  source = "../common"
}

resource "null_resource" "auth" {
  provisioner "local-exec" {
    command = "echo 'Auth module'"
  }
}

variable "enable_mfa" {
  type    = bool
  default = false
}