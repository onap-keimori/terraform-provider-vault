# Generating Resources and Data Sources

This code is part of a new code generation package. The process for generating code is:
- Ensure `testdata/openapi.json` includes the endpoints for which you want to generate code.
- If it doesn't:
  - Make a new Vault dev binary from the `vault-enterprise` repo.
  - Export a Vault license that includes the `transform` secrets engine: `export VAULT_LICENSE=foo`.
  - In the Vault or Vault Enterprise repo, run `bash scripts/gen_openapi.sh`.
  - Move the resulting file to `testdata/openapi.json`.
- From the home directory of `terraform-provider-vault`, run:
```
make generate
make fmt
```
- If you note any changes, you may need to hand-add code that implements 
[best practices](https://www.terraform.io/docs/extend/best-practices/deprecations.html)
for deprecations.
- Hand-test the code while comparing it to Vault's API docs until you're satisfied that
the generated code is correct. This is an important piece of the QA process.
- Hand-write unit tests for the code.
- Hand-add the new resource or data source to `vault/provider.go`.

This process is designed this way to ensure high quality in the generated code.
