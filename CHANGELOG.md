# Changelog

## [Unreleased]

### Added

- Add typed upload parameters with `create_append` support.
- Add query options for SQL dialect, output format, and expanded compute sizes.

### Changed

- Send composite upsert keys and cursor fields without the retired upsert mode parameter.
- Surface backend-emitted NDJSON query errors as typed errors with stream line context.

## [0.4.1](https://github.com/altertable-ai/altertable-lakehouse-go/compare/github.com/altertable-ai/altertable-lakehouse-go-v0.4.0...github.com/altertable-ai/altertable-lakehouse-go-v0.4.1) (2026-06-30)


### Bug Fixes

* **api:** expose upsert API ([#13](https://github.com/altertable-ai/altertable-lakehouse-go/issues/13)) ([777417b](https://github.com/altertable-ai/altertable-lakehouse-go/commit/777417b0c933f882dd229938c677353bc2275b8c))

## [0.4.0](https://github.com/altertable-ai/altertable-lakehouse-go/compare/github.com/altertable-ai/altertable-lakehouse-go-v0.3.0...github.com/altertable-ai/altertable-lakehouse-go-v0.4.0) (2026-05-17)


### Features

* **autocomplete:** re-enable autocomplete ([#10](https://github.com/altertable-ai/altertable-lakehouse-go/issues/10)) ([8cb1415](https://github.com/altertable-ai/altertable-lakehouse-go/commit/8cb1415a13cb10febb0fcf3bd4f57b7c9c1488a5))

## [0.3.0](https://github.com/altertable-ai/altertable-lakehouse-go/compare/github.com/altertable-ai/altertable-lakehouse-go-v0.2.0...github.com/altertable-ai/altertable-lakehouse-go-v0.3.0) (2026-05-14)


### Features

* **client:** add v0.11.0 lakehouse endpoints ([#7](https://github.com/altertable-ai/altertable-lakehouse-go/issues/7)) ([e0a16f6](https://github.com/altertable-ai/altertable-lakehouse-go/commit/e0a16f64cb2813395b5eb8fea6d9352e7b1addc2))

## [0.2.0](https://github.com/altertable-ai/altertable-lakehouse-go/compare/github.com/altertable-ai/altertable-lakehouse-go-v0.1.0...github.com/altertable-ai/altertable-lakehouse-go-v0.2.0) (2026-05-14)


### Features

* bootstrap lakehouse go sdk ([#2](https://github.com/altertable-ai/altertable-lakehouse-go/issues/2)) ([77a5189](https://github.com/altertable-ai/altertable-lakehouse-go/commit/77a5189972d23f69e012a549a8e0890bbe1b3eea))


### Bug Fixes

* keep default user agent in sync with release version ([#5](https://github.com/altertable-ai/altertable-lakehouse-go/issues/5)) ([65617b1](https://github.com/altertable-ai/altertable-lakehouse-go/commit/65617b1a1c9d6ec2be84079325624a4fc6ac5e64))

## 0.1.0

- initial Go SDK bootstrap for the Altertable Lakehouse API
- add typed client for append, query, queryAll, getQuery, cancelQuery, upsert, and validate
- add typed error hierarchy, configurable transport, and mock-backed tests
