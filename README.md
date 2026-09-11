# Terraform Provider for NeoShowcase

This repository contains a Terraform provider for
[NeoShowcase](https://github.com/traPtitech/NeoShowcase).

> [!NOTE]
> The provider is under active development. The currently registered managed
> resource is `neoshowcase_repository`; the registered data sources are
> `neoshowcase_current_user` and `neoshowcase_system_info`.

## Requirements

- Terraform 1.11 or later
- Go 1.25 or later

## Provider configuration

```terraform
terraform {
  required_providers {
    neoshowcase = {
      source = "traP-jp/neoshowcase"
    }
  }
}

provider "neoshowcase" {
  endpoint = "https://showcase.example.com"
  user     = "terraform"
}
```

The following environment variables can be used instead of provider
attributes:

- `NEOSHOWCASE_ENDPOINT`
- `NEOSHOWCASE_USER`
- `NEOSHOWCASE_AUTH_HEADER`

NeoShowcase authenticates API requests using a trusted reverse-proxy header.
The default header is `X-Showcase-User`.

## Development

```console
make proto
make test
make build
```

The protobuf API schema is pinned to the upstream revision documented in
[`proto/UPSTREAM.md`](proto/UPSTREAM.md). Generated files under
`internal/neoshowcase/gen` must not be edited manually.

## Managed resources

- `neoshowcase_repository`

### Repository ownership

The user resolved by the provider's `user` setting is always included as the
authoritative repository owner. `additional_owner_ids` is the exact set of
other owners managed by Terraform; do not include the provider user in it.

When changing the provider user, first grant the new user ownership outside
this provider or include it as an additional owner in a preceding apply. The
first refresh under the new identity intentionally fails if that user is not
already an owner, preventing Terraform from locking itself out.

Repository passwords use Terraform 1.11 write-only attributes and are never
stored in state. Set `auth.password_wo` for BASIC authentication and increment
`auth.password_wo_version` whenever the password should be sent again. Importing
an existing BASIC repository requires supplying a password on the first change
to its authentication settings because NeoShowcase does not return credentials.

## Planned managed resources

- `neoshowcase_application`
- `neoshowcase_environment_variable`
- `neoshowcase_user_key`

The provider user is treated as the authoritative owner of repositories and
applications. Other owners are managed separately through
`additional_owner_ids`.
