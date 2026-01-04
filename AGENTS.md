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

## Repository Overview

This repository contains a Go codebase with supporting infrastructure definitions.

## Code Structure

- `cmd/`: Entry points for binaries.
- `internal/`: Application packages and business logic.
  - `internal/ai`: AI client interfaces and the OpenAI-backed implementation for text, JSON, and TTS generation, including token usage tracking.
  - `internal/config`: Configuration defaults, file/env/flag merging, and validation for script, audio, and publish workflows.
  - `internal/paths`: Output path builder helpers and overwrite checks for generated episode artifacts.
  - `internal/podcast`: Episode schema, prompt building, safety checks, section generation helpers, and topic selection with history tracking.
  - `internal/storage`: S3 uploader and key helpers for publishing episode artifacts.
- `terraform/`: Infrastructure as code.
- `Makefile`: Common developer tasks.

## Documentation Maintenance

- Any time the code structure changes (e.g., new top-level directories or major layout updates), update this file to reflect the new structure.
