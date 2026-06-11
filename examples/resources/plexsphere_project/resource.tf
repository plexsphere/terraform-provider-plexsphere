resource "plexsphere_project" "payments" {
  domain_id = "0190a0c0-0000-7000-8000-000000000000" # parent domain (immutable)
  name      = "Payments"
  slug      = "payments" # immutable; changing it forces a replace

  description    = "Payments platform project"
  sub_range_cidr = "10.10.0.0/20"
}

# Look up an existing project by id.
data "plexsphere_project" "existing" {
  id = "0190a0c0-1111-7000-8000-000000000000"
}
