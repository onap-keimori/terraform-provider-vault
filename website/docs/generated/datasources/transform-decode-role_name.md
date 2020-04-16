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
* `batch_input` - (Optional) Specifies a list of items to be decoded in a single batch. If this parameter is set, the top-level parameters &#39;value&#39;, &#39;transformation&#39; and &#39;tweak&#39; will be ignored. Each batch item within the list can specify these parameters instead.
* `role_name` - (Required) The name of the role.
* `transformation` - (Optional) The transformation to perform. If no value is provided and the role contains a single transformation, this value will be inferred from the role.
* `tweak` - (Optional) The tweak value to use. Only applicable for FPE transformations
* `value` - (Optional) The value in which to decode.
