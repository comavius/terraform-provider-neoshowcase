# Terraform Provider for NeoShowcase

This repository contains a Terraform provider for
[NeoShowcase](https://github.com/traPtitech/NeoShowcase).

> [!NOTE]
> The provider is under active development. The currently registered data
> sources are `neoshowcase_current_user` and `neoshowcase_system_info`.
> Managed resources will be registered after their complete lifecycle and
> acceptance tests are implemented.

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

## Planned managed resources

- `neoshowcase_repository`
- `neoshowcase_application`
- `neoshowcase_environment_variable`
- `neoshowcase_user_key`

The provider user is treated as the authoritative owner of repositories and
applications. Other owners are managed separately through
`additional_owner_ids`.
