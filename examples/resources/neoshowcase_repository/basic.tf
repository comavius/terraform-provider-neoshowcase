resource "neoshowcase_repository" "private" {
  name = "private-example"
  url  = "https://git.example.com/team/private-example.git"

  auth = {
    method              = "basic"
    username            = "terraform"
    password_wo         = var.repository_password
    password_wo_version = 1
  }
}

variable "repository_password" {
  type      = string
  sensitive = true
}
