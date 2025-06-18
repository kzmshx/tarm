variable "vpc_cidr" {
  type = string
}

resource "null_resource" "network" {
  provisioner "local-exec" {
    command = "echo 'Network module'"
  }
}

output "vpc_id" {
  value = "vpc-12345"
}