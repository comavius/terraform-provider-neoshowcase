# Quick start

## Requirements

- Terraform 1.11 or later
- An authenticated NeoShowcase browser session

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
}
```

The following environment variables can be used instead of provider attributes:

- `NEOSHOWCASE_ENDPOINT`
- `NEOSHOWCASE_SESSION_COOKIE`

When neither the provider attribute nor `NEOSHOWCASE_ENDPOINT` is set, the endpoint defaults to `https://ns.trap.jp`.

`NEOSHOWCASE_SESSION_COOKIE` is required. Copy the complete value of the `Cookie` request header from an authenticated NeoShowcase browser session; omit the `Cookie:` prefix and attributes from a `Set-Cookie` response header.

## Supported resources and data sources

The provider currently supports:

- Resource: `neoshowcase_repository`
- Resource: `neoshowcase_application` (including environment variables)
- Data source: `neoshowcase_current_user`
- Data source: `neoshowcase_system_info`

Configuration examples are available under [`examples`](../examples).

### Repository ownership

The user resolved from the authenticated session is always included as the authoritative repository owner. `additional_owner_ids` is the exact set of other owners managed by Terraform; do not include the authenticated user in it.

When changing the authenticated user, first grant the new user ownership outside this provider or include it as an additional owner in a preceding apply. The first refresh under the new identity intentionally fails if that user is not already an owner, preventing Terraform from locking itself out.

### Repository passwords

Repository passwords use Terraform 1.11 write-only attributes and are never stored in state. Set `auth.password_wo` for BASIC authentication and increment `auth.password_wo_version` whenever the password should be sent again.

Importing an existing BASIC repository requires supplying a password on the first change to its authentication settings because NeoShowcase does not return credentials.

### Applications and environment variables

`neoshowcase_application` manages build configuration, access URLs, port forwarding rules, owners, running state, and the complete set of user-defined environment variables. Environment variables are nested in the application rather than represented by separate resources.

Environment variable values use the write-only `value_wo` attribute and are never stored in Terraform state. Increment `value_wo_version` to send a new value. Keys beginning with `NS_` are reserved for NeoShowcase and rejected by the provider. NeoShowcase-generated system environment variables are excluded from the managed map; their keys are exposed through `system_environment_variable_keys` without exposing their values.

Terraform detects added and removed environment-variable keys. Because write-only values cannot be retained for comparison, a value changed outside Terraform cannot be detected; increment `value_wo_version` to explicitly reconcile a value.

The repository ID and the `use_mariadb` and `use_mongodb` settings cannot be changed in place and therefore replace the application. The provider creates an application in the stopped state, configures its owners and environment variables, and only then starts it when `running = true`.

When a create or update requires a new build for a running application, the provider waits up to 10 minutes for that build. It checks immediately and then every 11 seconds, matching `neoshowcase-cli`; only a `SUCCEEDED` build completes the Terraform operation successfully.

## Planned resources

- `neoshowcase_user_key`

The authenticated user is treated as the authoritative owner of applications, as it is for repositories. Other owners are managed separately through `additional_owner_ids`.
