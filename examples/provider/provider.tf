terraform {
  required_providers {
    plexsphere = {
      source = "plexsphere/plexsphere"
    }
  }
}

provider "plexsphere" {
  endpoint = "https://api.plexsphere.com"
  # token is read from the PLEXSPHERE_TOKEN environment variable.
}
