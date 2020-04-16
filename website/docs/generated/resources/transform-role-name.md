---
layout: "vault"
page_title: "Vault: vault_jwt_auth_backend resource"
sidebar_current: "docs-vault-resource-jwt-auth-backend"
description: |-
  Managing JWT/OIDC auth backends in Vault
---

# vault\_jwt\_auth\_backend

Provides a resource for managing an
[JWT auth backend within Vault](https://www.vaultproject.io/docs/auth/jwt.html).

## Example Usage

Manage JWT auth backend:

```hcl
resource "vault_jwt_auth_backend" "example" {
    description  = "Demonstration of the Terraform JWT auth backend"
    path = "jwt"
    oidc_discovery_url = "https://myco.auth0.com/"
    bound_issuer = "https://myco.auth0.com/"
}
```

Manage OIDC auth backend:

```hcl
resource "vault_jwt_auth_backend" "example" {
    description  = "Demonstration of the Terraform JWT auth backend"
    path = "oidc"
    type = "oidc"
    oidc_discovery_url = "https://myco.auth0.com/"
    oidc_client_id = "1234567890"
    oidc_client_secret = "secret123456"
    bound_issuer = "https://myco.auth0.com/"
    tune {
        listing_visibility = "unauth"
    }
}
```

## Argument Reference

The following arguments are supported:
* `path` - (Required) Path to where the back-end is mounted within Vault.
* `name` - (Required) The name of the role.
* `transformations` - (Optional) A comma separated string or slice of transformations to use.
