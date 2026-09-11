resource "neoshowcase_application" "example" {
  name          = "example"
  repository_id = neoshowcase_repository.example.id
  ref_name      = "main"
  running       = true

  build = {
    type          = "runtime_buildpack"
    context       = "."
    entrypoint    = "./server"
    use_mariadb   = true
    use_mongodb   = false
  }

  urls = [{
    fqdn           = "example.showcase.example.com"
    path_prefix    = "/"
    strip_prefix   = false
    https          = true
    h2c            = false
    http_port      = 3000
    authentication = "off"
  }]

  environment_variables = {
    APP_SECRET = {
      value_wo         = var.application_secret
      value_wo_version = 1
    }
  }

  additional_owner_ids = []
}

variable "application_secret" {
  type      = string
  sensitive = true
}
