variable "tags" {
  type    = map(string)
  default = {}
}

output "common_tags" {
  value = var.tags
}