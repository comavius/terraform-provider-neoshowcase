terraform {
  required_version = ">= 1.11.0"

  required_providers {
    neoshowcase = {
      source = "comavius/neoshowcase"
    }
  }
}

provider "neoshowcase" {
  endpoint = "https://showcase.example.com"
}
