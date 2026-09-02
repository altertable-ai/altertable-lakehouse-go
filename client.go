package altertable

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL         = "https://api.altertable.ai"
	defaultConnectTimeout  = 5 * time.Second
	defaultResponseTimeout = 60 * time.Second
	defaultUserAgent       = "altertable-lakehouse-go/" + Version
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type RetryPolicy struct {
	MaxAttempts int
	Backoff     func(attempt int) time.Duration
}

type Config struct {
	BaseURL         string
	Username        string
	Password        string
	BasicAuthToken  string
	Timeout         time.Duration
	ConnectTimeout  time.Duration
	RetryPolicy     RetryPolicy
	UserAgentSuffix string
	HTTPClient      HTTPDoer
}

type Client struct {
	baseURL     string
	httpClient  HTTPDoer
	authHeader  string
	userAgent   string
	retryPolicy RetryPolicy
}

type operationSpec struct {
	Name        string
	Method      string
	Path        string
	ContentType string
	Accept      string
	RawBody     io.Reader
}

func NewClient(cfg Config) (*Client, error) {
	authHeader, err := resolveAuthHeader(cfg)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	userAgent := defaultUserAgent
	if cfg.UserAgentSuffix != "" {
		userAgent += " " + cfg.UserAgentSuffix
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		connectTimeout := cfg.ConnectTimeout
		if connectTimeout <= 0 {
			connectTimeout = defaultConnectTimeout
		}
		responseTimeout := cfg.Timeout
		if responseTimeout <= 0 {
			responseTimeout = defaultResponseTimeout
		}

		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   connectTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = &http.Client{
			Timeout:   responseTimeout,
			Transport: transport,
		}
	}

	retryPolicy := cfg.RetryPolicy
	if retryPolicy.MaxAttempts <= 0 {
		retryPolicy.MaxAttempts = 1
	}
	if retryPolicy.Backoff == nil {
		retryPolicy.Backoff = func(attempt int) time.Duration {
			return time.Duration(attempt-1) * 100 * time.Millisecond
		}
	}

	return &Client{
		baseURL:     baseURL,
		httpClient:  httpClient,
		authHeader:  authHeader,
		userAgent:   userAgent,
		retryPolicy: retryPolicy,
	}, nil
}

func (c *Client) Append(ctx context.Context, params AppendParams, req AppendRequest) (*AppendResponse, error) {
	resp := &AppendResponse{}
	err := c.doJSON(ctx, operationSpec{Name: "append", Method: http.MethodPost, Path: "/append"}, params.values(), req, resp)
	return resp, err
}

func (c *Client) Validate(ctx context.Context, req ValidateRequest) (*ValidateResponse, error) {
	resp := &ValidateResponse{}
	err := c.doJSON(ctx, operationSpec{Name: "validate", Method: http.MethodPost, Path: "/validate"}, nil, req, resp)
	return resp, err
}

func (c *Client) Autocomplete(ctx context.Context, req AutocompleteRequest) (*AutocompleteResponse, error) {
	resp := &AutocompleteResponse{}
	err := c.doJSON(ctx, operationSpec{Name: "autocomplete", Method: http.MethodPost, Path: "/autocomplete"}, nil, req, resp)
	return resp, err
}

func (c *Client) Upsert(ctx context.Context, params UpsertParams, content io.Reader) error {
	body, err := io.ReadAll(content)
	if err != nil {
		return &SerializationError{apiErrorBase: apiErrorBase{Message: "failed to read upsert content", Cause: err}}
	}

	_, err = c.do(ctx, operationSpec{Name: "upsert", Method: http.MethodPost, Path: "/upsert", ContentType: params.ContentType, RawBody: bytes.NewReader(body)}, params.values(), nil)
	return err
}

// Upload sends raw CSV, JSON, or Parquet content to a table. The service infers the format from
// UploadParams.ContentType when supplied, or the payload bytes when it is omitted.
func (c *Client) Upload(ctx context.Context, params UploadParams, content io.Reader) error {
	body, err := io.ReadAll(content)
	if err != nil {
		return &SerializationError{apiErrorBase: apiErrorBase{Message: "failed to read upload content", Cause: err}}
	}

	_, err = c.do(ctx, operationSpec{Name: "upload", Method: http.MethodPost, Path: "/upload", ContentType: params.ContentType, RawBody: bytes.NewReader(body)}, params.values(), nil)
	return err
}

func (c *Client) GetTask(ctx context.Context, taskID string) (*TaskResponse, error) {
	resp := &TaskResponse{}
	err := c.doJSON(ctx, operationSpec{Name: "getTask", Method: http.MethodGet, Path: "/tasks/" + taskID}, nil, nil, resp)
	return resp, err
}

func (c *Client) GetQuery(ctx context.Context, queryID string) (*QueryLogResponse, error) {
	resp := &QueryLogResponse{}
	err := c.doJSON(ctx, operationSpec{Name: "getQuery", Method: http.MethodGet, Path: "/query/" + queryID}, nil, nil, resp)
	return resp, err
}

func (c *Client) CancelQuery(ctx context.Context, queryID, sessionID string) (*CancelQueryResponse, error) {
	resp := &CancelQueryResponse{}
	err := c.doJSON(ctx, operationSpec{Name: "cancelQuery", Method: http.MethodDelete, Path: "/query/" + queryID}, url.Values{"session_id": []string{sessionID}}, nil, resp)
	return resp, err
}

func (c *Client) Query(ctx context.Context, req QueryRequest) (*QueryStreamResult, error) {
	if req.ComputeSize != nil && *req.ComputeSize == ComputeSizeAuto && req.SessionID != nil && *req.SessionID != "" {
		return nil, &ConfigurationError{apiErrorBase: apiErrorBase{Message: "compute_size AUTO cannot be combined with session_id", Operation: "query"}}
	}
	resp, err := c.do(ctx, operationSpec{Name: "query", Method: http.MethodPost, Path: "/query", ContentType: "application/json", Accept: "application/x-ndjson"}, nil, req)
	if err != nil {
		return nil, err
	}

	result, err := parseQueryStream(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, err
	}
	result.close = resp.Body.Close
	return result, nil
}

func (c *Client) QueryAll(ctx context.Context, req QueryRequest) (*QueryAllResult, error) {
	stream, err := c.Query(ctx, req)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	rows := make([][]any, 0)
	for {
		row, err := stream.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		rows = append(rows, row)
	}

	return &QueryAllResult{Metadata: stream.Metadata, Columns: stream.Columns, Rows: rows}, nil
}

func (c *Client) doJSON(ctx context.Context, spec operationSpec, query url.Values, requestBody any, responseBody any) error {
	resp, err := c.do(ctx, withJSON(spec), query, requestBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if responseBody == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
		return &SerializationError{apiErrorBase: apiErrorBase{Message: "failed to decode response body", Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: err}}
	}
	return nil
}

func withJSON(spec operationSpec) operationSpec {
	if spec.ContentType == "" {
		spec.ContentType = "application/json"
	}
	return spec
}

func (c *Client) do(ctx context.Context, spec operationSpec, query url.Values, requestBody any) (*http.Response, error) {
	var bodyBytes []byte
	var err error
	if spec.RawBody == nil && requestBody != nil {
		bodyBytes, err = json.Marshal(requestBody)
		if err != nil {
			if errors.Is(err, ErrInvalidAppendRequest) {
				return nil, &ConfigurationError{apiErrorBase: apiErrorBase{Message: err.Error(), Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: err}}
			}
			return nil, &SerializationError{apiErrorBase: apiErrorBase{Message: "failed to encode request body", Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: err}}
		}
	}

	for attempt := 1; attempt <= c.retryPolicy.MaxAttempts; attempt++ {
		resp, err := c.doOnce(ctx, spec, query, requestBody, bodyBytes)
		if err == nil {
			return resp, nil
		}
		var apiErr *ApiError
		if errors.As(err, &apiErr) && !apiErr.Retriable() {
			return nil, err
		}
		var netErr interface{ Retriable() bool }
		if errors.As(err, &netErr) && !netErr.Retriable() {
			return nil, err
		}
		if attempt == c.retryPolicy.MaxAttempts {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, &TimeoutError{apiErrorBase: apiErrorBase{Message: "request cancelled", Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: ctx.Err(), RetriableFlag: false}}
		case <-time.After(c.retryPolicy.Backoff(attempt + 1)):
		}
	}
	return nil, &ApiError{apiErrorBase: apiErrorBase{Message: "request attempts exhausted", Operation: spec.Name, Method: spec.Method, Path: spec.Path}}
}

func (c *Client) doOnce(ctx context.Context, spec operationSpec, query url.Values, requestBody any, bodyBytes []byte) (*http.Response, error) {
	endpoint, err := url.Parse(c.baseURL + spec.Path)
	if err != nil {
		return nil, &ConfigurationError{apiErrorBase: apiErrorBase{Message: "invalid base URL", Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: err}}
	}
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}

	var body io.Reader
	if spec.RawBody != nil {
		body = spec.RawBody
		if seeker, ok := spec.RawBody.(io.Seeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}
	} else if bodyBytes != nil {
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, spec.Method, endpoint.String(), body)
	if err != nil {
		return nil, &ConfigurationError{apiErrorBase: apiErrorBase{Message: "failed to construct request", Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: err}}
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("User-Agent", c.userAgent)
	if spec.ContentType != "" {
		req.Header.Set("Content-Type", spec.ContentType)
	}
	if spec.Accept != "" {
		req.Header.Set("Accept", spec.Accept)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, mapTransportError(spec, err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	return nil, mapStatusError(spec, resp, payload)
}

func resolveAuthHeader(cfg Config) (string, error) {
	token := strings.TrimSpace(cfg.BasicAuthToken)
	switch {
	case token != "":
		return "Basic " + token, nil
	case cfg.Username != "" || cfg.Password != "":
		if cfg.Username == "" || cfg.Password == "" {
			return "", &ConfigurationError{apiErrorBase: apiErrorBase{Message: "username and password must be provided together", Operation: "configuration"}}
		}
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)), nil
	}

	envToken := strings.TrimSpace(os.Getenv("ALTERTABLE_BASIC_AUTH_TOKEN"))
	if envToken != "" {
		return "Basic " + envToken, nil
	}
	username := strings.TrimSpace(os.Getenv("ALTERTABLE_USERNAME"))
	password := os.Getenv("ALTERTABLE_PASSWORD")
	if username != "" || password != "" {
		if username == "" || password == "" {
			return "", &ConfigurationError{apiErrorBase: apiErrorBase{Message: "ALTERTABLE_USERNAME and ALTERTABLE_PASSWORD must both be set", Operation: "configuration"}}
		}
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password)), nil
	}

	return "", &ConfigurationError{apiErrorBase: apiErrorBase{Message: "no Altertable credentials configured", Operation: "configuration"}}
}

func parseQueryStream(reader io.Reader) (*QueryStreamResult, error) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	lineIndex := 0
	result := &QueryStreamResult{}
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if lineIndex == 0 {
			if err := json.Unmarshal(line, &result.Metadata); err != nil {
				return nil, &ParseError{apiErrorBase: apiErrorBase{Message: "failed to parse query metadata", Cause: err}, LineIndex: lineIndex, RawContent: string(line)}
			}
			lineIndex++
			continue
		}
		if lineIndex == 1 {
			if queryErr := decodeQueryStreamError(line, lineIndex); queryErr != nil {
				return nil, queryErr
			}
			if bytes.HasPrefix(line, []byte("[{")) {
				if err := json.Unmarshal(line, &result.Columns); err != nil {
					return nil, &ParseError{apiErrorBase: apiErrorBase{Message: "failed to parse query columns", Cause: err}, LineIndex: lineIndex, RawContent: string(line)}
				}
			} else {
				var columnNames []string
				if err := json.Unmarshal(line, &columnNames); err != nil {
					return nil, &ParseError{apiErrorBase: apiErrorBase{Message: "failed to parse query columns", Cause: err}, LineIndex: lineIndex, RawContent: string(line)}
				}
				result.Columns = make([]QueryColumn, len(columnNames))
				for i, name := range columnNames {
					result.Columns[i] = QueryColumn{Name: name}
				}
			}
			lineIndex++
			continue
		}
		if queryErr := decodeQueryStreamError(line, lineIndex); queryErr != nil {
			return nil, queryErr
		}
		var row []any
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, &ParseError{apiErrorBase: apiErrorBase{Message: "failed to parse query row", Cause: err}, LineIndex: lineIndex, RawContent: string(line)}
		}
		result.Rows = append(result.Rows, row)
		lineIndex++
	}
	if err := scanner.Err(); err != nil {
		return nil, &ParseError{apiErrorBase: apiErrorBase{Message: "failed to read query stream", Cause: err}, LineIndex: lineIndex}
	}
	if lineIndex < 2 {
		return nil, &ParseError{apiErrorBase: apiErrorBase{Message: "query stream missing metadata or columns"}, LineIndex: lineIndex}
	}
	return result, nil
}

func decodeQueryStreamError(line []byte, lineIndex int) error {
	var payload struct {
		Error *string `json:"error"`
	}
	if err := json.Unmarshal(line, &payload); err != nil || payload.Error == nil {
		return nil
	}
	return &QueryError{
		apiErrorBase: apiErrorBase{Message: *payload.Error, Operation: "query"},
		LineIndex:    lineIndex,
		RawContent:   string(line),
	}
}

func mapTransportError(spec operationSpec, err error) error {
	message := err.Error()
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return &TimeoutError{apiErrorBase: apiErrorBase{Message: "request timed out", Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: err, RetriableFlag: true}}
		}
		return &NetworkError{apiErrorBase: apiErrorBase{Message: message, Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: err, RetriableFlag: true}}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &TimeoutError{apiErrorBase: apiErrorBase{Message: "request timed out", Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: err, RetriableFlag: false}}
	}
	return &NetworkError{apiErrorBase: apiErrorBase{Message: message, Operation: spec.Name, Method: spec.Method, Path: spec.Path, Cause: err, RetriableFlag: true}}
}

func mapStatusError(spec operationSpec, resp *http.Response, payload []byte) error {
	message := strings.TrimSpace(string(payload))
	if message == "" {
		message = resp.Status
	}
	base := apiErrorBase{
		Message:         message,
		Operation:       spec.Name,
		Method:          spec.Method,
		Path:            spec.Path,
		StatusCodeValue: resp.StatusCode,
		RetriableFlag:   resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests,
		RequestID:       resp.Header.Get("X-Request-Id"),
		CorrelationID:   resp.Header.Get("X-Correlation-Id"),
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &AuthError{apiErrorBase: base}
	case http.StatusBadRequest:
		return &BadRequestError{apiErrorBase: base}
	default:
		return &ApiError{apiErrorBase: base}
	}
}

func redactAuthHeader(header string) string {
	if header == "" {
		return ""
	}
	return "Basic [REDACTED]"
}

func debugConfigSummary(cfg Config) string {
	return fmt.Sprintf("baseURL=%s auth=%s timeout=%s connectTimeout=%s userAgentSuffix=%q",
		strings.TrimRight(cfg.BaseURL, "/"),
		redactAuthHeader(mustAuthHeaderForDebug(cfg)),
		cfg.Timeout,
		cfg.ConnectTimeout,
		cfg.UserAgentSuffix,
	)
}

func mustAuthHeaderForDebug(cfg Config) string {
	header, _ := resolveAuthHeader(cfg)
	return header
}
