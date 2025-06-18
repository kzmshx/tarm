module "module_a" {
  source = "../module-a"
  
  value = "from-b"
}

output "from_b" {
  value = "Hello from B"
}