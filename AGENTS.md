# Agent Instructions

- All changes must use idiomatic, boring Go code.
- Changes should be logically independent.
- Keep changes focused, small, and easy to review.
- Prefer clear naming, straightforward control flow, and standard library solutions.
- Avoid unnecessary abstractions or cleverness; optimize for readability and maintenance.
- For Terraform changes, follow boring, idiomatic infrastructure patterns and avoid over-engineering.
- Before committing changes:
  - tests must pass.
  - `goimports` must pass.
  - `terraform fmt` must pass for any Terraform changes.
