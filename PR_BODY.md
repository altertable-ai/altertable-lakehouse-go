## Summary

Initial bootstrap of the Go Lakehouse SDK against `altertable-client-specs` `v0.10.0`.

## Impact analysis

- Adds the initial public Go client surface: `NewClient`, `Append`, `Query`, `QueryAll`, `GetQuery`, `CancelQuery`, `Upsert`, and `Validate`.
- Introduces typed models for lakehouse requests/responses, query metadata, query logs, upsert configuration, and stats/progress payloads.
- Adds typed error handling, auth resolution, retry/timeout configuration, and NDJSON parsing.
- Adds both unit coverage and mock-backed integration coverage for all required endpoints.
- Docs affected: README usage/configuration/examples, changelog bootstrap entry.
- Cross-SDK check: compared against `altertable-lakehouse-python` and `altertable-lakehouse-ruby`; this PR keeps the same endpoint surface and auth patterns while using Go-idiomatic types and method names.

## Validation

- `ALTERTABLE_MOCK_PORT=34643 go test ./...`

## Notes

- The mock server currently emits query columns as a JSON string array on `/query`; the Go client accepts that form and object-based column schemas.
- `upsert` integration coverage currently exercises the request path against the mock and asserts the expected `400` response for a non-existent catalog, since the mock does not ship a writable fixture catalog by default.
