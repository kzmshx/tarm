module "module_b" {
  source = "../module-b"
  
  value = "from-a"
}

output "from_a" {
  value = "Hello from A"
}