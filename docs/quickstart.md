# Quick start

## Requirements

- Terraform 1.11 or later
- A NeoShowcase endpoint that accepts authentication from a trusted reverse-proxy header

## Provider configuration

Declare the provider using its Terraform Registry source address:

```terraform
terraform {
  required_providers {
    neoshowcase = {
      source = "comavius/neoshowcase"
    }
  }
}

provider "neoshowcase" {
  endpoint = "https://showcase.example.com"
  user     = "terraform"
}
```

The following environment variables can be used instead of provider attributes:

- `NEOSHOWCASE_ENDPOINT`
- `NEOSHOWCASE_USER`
- `NEOSHOWCASE_AUTH_HEADER`

NeoShowcase authenticates API requests using a trusted reverse-proxy header. The default header is `X-Showcase-User`.

## Supported resources and data sources

The provider currently supports:

- Resource: `neoshowcase_repository`
- Data source: `neoshowcase_current_user`
- Data source: `neoshowcase_system_info`

Configuration examples are available under [`examples`](../examples).

### Repository ownership

The user resolved by the provider's `user` setting is always included as the authoritative repository owner. `additional_owner_ids` is the exact set of other owners managed by Terraform; do not include the provider user in it.

When changing the provider user, first grant the new user ownership outside this provider or include it as an additional owner in a preceding apply. The first refresh under the new identity intentionally fails if that user is not already an owner, preventing Terraform from locking itself out.

### Repository passwords

Repository passwords use Terraform 1.11 write-only attributes and are never stored in state. Set `auth.password_wo` for BASIC authentication and increment `auth.password_wo_version` whenever the password should be sent again.

Importing an existing BASIC repository requires supplying a password on the first change to its authentication settings because NeoShowcase does not return credentials.

## Planned resources

- `neoshowcase_application`
- `neoshowcase_environment_variable`
- `neoshowcase_user_key`

The provider user will be treated as the authoritative owner of applications, as it is for repositories. Other owners will be managed separately through `additional_owner_ids`.
