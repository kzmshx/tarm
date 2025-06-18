# Utility module
variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

output "common_tags" {
  value = {
    Environment = var.environment
    ManagedBy   = "terraform"
    Project     = "tarm-test"
  }
}