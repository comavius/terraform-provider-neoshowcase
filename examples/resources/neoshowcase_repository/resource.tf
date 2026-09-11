resource "neoshowcase_repository" "example" {
  name = "example"
  url  = "https://github.com/traPtitech/NeoShowcase.git"

  auth = {
    method = "none"
  }

  additional_owner_ids = []
}

output "repository_id" {
  value = neoshowcase_repository.example.id
}
