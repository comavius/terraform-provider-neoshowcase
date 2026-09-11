terraform {
  required_version = ">= 1.11.0"

  required_providers {
    neoshowcase = {
      source = "comavius/neoshowcase"
    }
  }
}

provider "neoshowcase" {
  endpoint = "http://127.0.0.1:1"
}

data "neoshowcase_current_user" "validation" {}

data "neoshowcase_system_info" "validation" {}

resource "neoshowcase_repository" "public" {
  name = "public"
  url  = "https://example.com/public.git"

  auth = {
    method = "none"
  }

  additional_owner_ids = ["additional-owner"]
}

resource "neoshowcase_repository" "private" {
  name = "private"
  url  = "https://example.com/private.git"

  auth = {
    method              = "basic"
    username            = "git"
    password_wo         = var.repository_password
    password_wo_version = 1
  }
}

resource "neoshowcase_application" "static" {
  name          = "static"
  repository_id = neoshowcase_repository.public.id
  ref_name      = "main"

  build = {
    type          = "static_buildpack"
    artifact_path = "."
    context       = ""
  }

  environment_variables = {
    SITE_TITLE = {
      value_wo         = "NeoShowcase"
      value_wo_version = 1
    }
  }
}

variable "repository_password" {
  type      = string
  sensitive = true
  default   = "validation-only"
}
