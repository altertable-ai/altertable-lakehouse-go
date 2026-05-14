# altertable-lakehouse-go

Go client for the Altertable Lakehouse API on Go 1.21+.

## Install

```bash
go get github.com/altertable-ai/altertable-lakehouse-go
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	altertable "github.com/altertable-ai/altertable-lakehouse-go"
)

func main() {
	client, err := altertable.NewClient(altertable.Config{
		Username: "username",
		Password: "password",
	})
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.QueryAll(context.Background(), altertable.QueryRequest{
		Statement: "SELECT 1",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Columns)
	fmt.Println(result.Rows)
}
```

## API reference

### Initialization

#### `altertable.NewClient(config Config) (*Client, error)`

Creates a client with Basic Auth, timeout, retry, and transport configuration.

```go
client, err := altertable.NewClient(altertable.Config{
	BaseURL:        "https://api.altertable.ai",
	Username:       "username",
	Password:       "password",
	Timeout:        60 * time.Second,
	ConnectTimeout: 5 * time.Second,
	RetryPolicy: altertable.RetryPolicy{
		MaxAttempts: 2,
	},
	UserAgentSuffix: "my-app/1.2.3",
})
```

### Querying

#### `client.Query(ctx, request) (*QueryStreamResult, error)`

Runs a SQL query and returns metadata, columns, and a row iterator backed by the NDJSON response.

```go
stream, err := client.Query(ctx, altertable.QueryRequest{Statement: "SELECT 1"})
if err != nil {
	log.Fatal(err)
}
defer stream.Close()

for {
	row, err := stream.Next()
	if err != nil {
		break
	}
	fmt.Println(row)
}
```

#### `client.QueryAll(ctx, request) (*QueryAllResult, error)`

Runs a SQL query and accumulates all rows before returning.

```go
result, err := client.QueryAll(ctx, altertable.QueryRequest{Statement: "SELECT 1"})
fmt.Println(result.Rows, err)
```

#### `client.GetQuery(ctx, queryID) (*QueryLogResponse, error)`

Fetches the query log for a query ID.

```go
queryLog, err := client.GetQuery(ctx, "123e4567-e89b-12d3-a456-426614174000")
fmt.Println(queryLog, err)
```

#### `client.CancelQuery(ctx, queryID, sessionID) (*CancelQueryResponse, error)`

Cancels a running query for the given session.

```go
cancelResp, err := client.CancelQuery(ctx, queryID, sessionID)
fmt.Println(cancelResp, err)
```

### Data ingestion

#### `client.Append(ctx, params, request) (*AppendResponse, error)`

Appends one payload or a batch of payloads to a table.

```go
sync := true
appendResp, err := client.Append(ctx, altertable.AppendParams{
	Catalog: "catalog",
	Schema:  "public",
	Table:   "events",
	Sync:    &sync,
}, altertable.AppendRequest{
	Single: altertable.AppendPayload{"id": 1, "name": "Alice"},
})
fmt.Println(appendResp, err)
```

#### `client.GetTask(ctx, taskID) (*TaskResponse, error)`

Fetches async append task status.

```go
taskResp, err := client.GetTask(ctx, "123e4567-e89b-12d3-a456-426614174000")
fmt.Println(taskResp, err)
```

#### `client.Upload(ctx, params, content) error`

Uploads CSV, JSON, or Parquet content to a table.

```go
err = client.Upload(ctx, altertable.UploadParams{
	Catalog: "catalog",
	Schema:  "public",
	Table:   "events",
	Format:  altertable.UploadFormatCSV,
	Mode:    altertable.UploadModeCreate,
}, strings.NewReader("id,name\n1,Alice\n"))
fmt.Println(err)
```

### Validation

#### `client.Validate(ctx, request) (*ValidateResponse, error)`

Validates a SQL statement without executing it.

```go
validateResp, err := client.Validate(ctx, altertable.ValidateRequest{Statement: "SELECT 1"})
fmt.Println(validateResp, err)
```

#### `client.Autocomplete(ctx, request) (*AutocompleteResponse, error)`

Requests SQL autocomplete suggestions.

```go
autocompleteResp, err := client.Autocomplete(ctx, altertable.AutocompleteRequest{Statement: "SEL"})
fmt.Println(autocompleteResp, err)
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `BaseURL` | `string` | `https://api.altertable.ai` | Base URL for the Lakehouse API. |
| `Username` | `string` | none | Basic Auth username. Use with `Password`. |
| `Password` | `string` | none | Basic Auth password. Use with `Username`. |
| `BasicAuthToken` | `string` | none | Pre-encoded Basic Auth token, without the `Basic ` prefix. |
| `Timeout` | `time.Duration` | `60s` | End-to-end HTTP response timeout. |
| `ConnectTimeout` | `time.Duration` | `5s` | TCP connect timeout for the default transport. |
| `RetryPolicy.MaxAttempts` | `int` | `1` | Total request attempts, including the first call. |
| `RetryPolicy.Backoff` | `func(int) time.Duration` | linear `100ms` steps | Backoff function for retries. |
| `UserAgentSuffix` | `string` | none | Appended to the default SDK user agent. |
| `HTTPClient` | `HTTPDoer` | default `*http.Client` | Custom HTTP client implementation. |
| `ALTERTABLE_USERNAME` | environment variable | none | Username used when constructor credentials are omitted. |
| `ALTERTABLE_PASSWORD` | environment variable | none | Password paired with `ALTERTABLE_USERNAME`. |
| `ALTERTABLE_BASIC_AUTH_TOKEN` | environment variable | none | Pre-encoded Basic Auth token discovered from the environment. |

## Development

Requires Go 1.21 or newer.

```bash
go test ./...
gofmt -w .
ALTERTABLE_MOCK_PORT=15000 go test ./...
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).
