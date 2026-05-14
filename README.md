# altertable-lakehouse-go

Go client for the Altertable Lakehouse API.

## Installation

```bash
go get github.com/altertable-ai/altertable-lakehouse-go
```

## Features

- Basic Auth via direct credentials, pre-encoded token, or environment discovery
- Typed request/response models and typed errors
- Streamed `Query` and accumulated `QueryAll`
- Configurable timeouts, retries, and user-agent suffix
- Mock-backed integration tests

## Configuration

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

You can also provide a pre-encoded token with `BasicAuthToken`, or let the client read:

- `ALTERTABLE_USERNAME` + `ALTERTABLE_PASSWORD`
- `ALTERTABLE_BASIC_AUTH_TOKEN`

If no credentials can be resolved, `NewClient` returns `ConfigurationError`.

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "strings"

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

    ctx := context.Background()

    appendResp, err := client.Append(ctx, altertable.AppendParams{
        Catalog: "catalog",
        Schema:  "public",
        Table:   "events",
    }, altertable.AppendRequest{Single: map[string]any{"id": 1}})
    fmt.Println(appendResp, err)

    stream, err := client.Query(ctx, altertable.QueryRequest{Statement: "SELECT 1"})
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()
    fmt.Println(stream.Metadata)
    fmt.Println(stream.Columns)
    for {
        row, err := stream.Next()
        if err != nil {
            break
        }
        fmt.Println(row)
    }

    allRows, err := client.QueryAll(ctx, altertable.QueryRequest{Statement: "SELECT 1"})
    fmt.Println(allRows, err)

    queryLog, err := client.GetQuery(ctx, stream.Metadata.QueryID)
    fmt.Println(queryLog, err)

    cancelResp, err := client.CancelQuery(ctx, stream.Metadata.QueryID, stream.Metadata.SessionID)
    fmt.Println(cancelResp, err)

    uploadErr := client.Upload(ctx, altertable.UploadParams{
        Catalog: "catalog",
        Schema:  "public",
        Table:   "events",
        Format:  altertable.UploadFormatCSV,
        Mode:    altertable.UploadModeCreate,
    }, strings.NewReader("id,name\n1,Alice\n"))
    fmt.Println(uploadErr)

    validateResp, err := client.Validate(ctx, altertable.ValidateRequest{Statement: "SELECT 1"})
    fmt.Println(validateResp, err)
}
```

## Error handling

Errors are returned as concrete types:

- `AuthError`
- `BadRequestError`
- `NetworkError`
- `TimeoutError`
- `SerializationError`
- `ParseError`
- `ApiError`
- `ConfigurationError`

Each error includes operation/method/path context, and API/transport errors expose retriable classification.

## Timeouts and retries

Default transport behavior:

- connection keep-alive enabled
- connect timeout: 5 seconds
- response timeout: 60 seconds
- HTTP/2 attempted by default
- retries configurable via `RetryPolicy`

For long-running queries, increase `Timeout` or inject a custom `HTTPClient`.

## Development

```bash
go test ./...
gofmt -w .
```

For integration tests, point the client at the Altertable mock server:

```bash
ALTERTABLE_MOCK_PORT=15000 go test ./...
```
