package altertable

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"
)

type ComputeSize string

const (
	ComputeSizeXS ComputeSize = "XS"
	ComputeSizeS  ComputeSize = "S"
	ComputeSizeM  ComputeSize = "M"
	ComputeSizeL  ComputeSize = "L"
	ComputeSizeXL ComputeSize = "XL"
)

type UpsertMode string

const (
	UpsertModeCreate    UpsertMode = "create"
	UpsertModeAppend    UpsertMode = "append"
	UpsertModeUpsert    UpsertMode = "upsert"
	UpsertModeOverwrite UpsertMode = "overwrite"
)

type SessionKind string

const (
	SessionKindArrowFlightSQL   SessionKind = "ArrowFlightSQL"
	SessionKindHTTPQuery        SessionKind = "HttpQuery"
	SessionKindHTTPCancel       SessionKind = "HttpCancel"
	SessionKindHTTPValidate     SessionKind = "HttpValidate"
	SessionKindHTTPExplain      SessionKind = "HttpExplain"
	SessionKindHTTPAutocomplete SessionKind = "HttpAutocomplete"
	SessionKindPostgres         SessionKind = "Postgres"
)

type AppendErrorCode string

const (
	AppendErrorCodeInvalidData        AppendErrorCode = "invalid-data"
	AppendErrorCodeIncompatibleSchema AppendErrorCode = "incompatible-schema"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusCompleted TaskStatus = "completed"
)

type AppendPayload map[string]any

type AppendRequest struct {
	Single AppendPayload
	Batch  []AppendPayload
}

func (r AppendRequest) MarshalJSON() ([]byte, error) {
	switch {
	case r.Single != nil && r.Batch != nil:
		return nil, ErrInvalidAppendRequest
	case r.Single != nil:
		return json.Marshal(r.Single)
	case r.Batch != nil:
		return json.Marshal(r.Batch)
	default:
		return []byte("null"), nil
	}
}

type AppendParams struct {
	Catalog string
	Schema  string
	Table   string
	Sync    *bool
}

func (p AppendParams) values() url.Values {
	values := url.Values{
		"catalog": []string{p.Catalog},
		"schema":  []string{p.Schema},
		"table":   []string{p.Table},
	}
	if p.Sync != nil {
		values.Set("sync", strconv.FormatBool(*p.Sync))
	}
	return values
}

type AppendResponse struct {
	OK           bool             `json:"ok"`
	ErrorCode    *AppendErrorCode `json:"error_code"`
	ErrorMessage *string          `json:"error_message"`
	TaskID       *string          `json:"task_id"`
}

type TaskResponse struct {
	TaskID string     `json:"task_id"`
	Status TaskStatus `json:"status"`
}

type QueryRequest struct {
	Statement   string       `json:"statement"`
	Cache       *bool        `json:"cache,omitempty"`
	Catalog     *string      `json:"catalog,omitempty"`
	ComputeSize *ComputeSize `json:"compute_size,omitempty"`
	Ephemeral   *bool        `json:"ephemeral,omitempty"`
	Limit       *int         `json:"limit,omitempty"`
	Offset      *int         `json:"offset,omitempty"`
	QueryID     *string      `json:"query_id,omitempty"`
	RequestedBy *string      `json:"requested_by,omitempty"`
	Sanitize    *bool        `json:"sanitize,omitempty"`
	Schema      *string      `json:"schema,omitempty"`
	SessionID   *string      `json:"session_id,omitempty"`
	Timezone    *string      `json:"timezone,omitempty"`
	Visible     *bool        `json:"visible,omitempty"`
}

type QueryMetadata struct {
	Statement         string            `json:"statement"`
	RowsLimit         *int              `json:"rows_limit"`
	RowsOffset        *int              `json:"rows_offset"`
	InitTimeMS        *int64            `json:"init_time_ms"`
	ConnectionsErrors map[string]string `json:"connections_errors"`
	SessionID         string            `json:"session_id"`
	QueryID           string            `json:"query_id"`
	WorkerSlug        *string           `json:"worker_slug"`
}

type QueryColumn struct {
	Name string
	Type string
}

type QueryStreamResult struct {
	Metadata QueryMetadata
	Columns  []QueryColumn
	Rows     [][]any
	index    int
	close    func() error
}

func (r *QueryStreamResult) Next() ([]any, error) {
	if r.index >= len(r.Rows) {
		return nil, io.EOF
	}
	row := r.Rows[r.index]
	r.index++
	return row, nil
}

func (r *QueryStreamResult) Close() error {
	if r.close != nil {
		return r.close()
	}
	return nil
}

type QueryAllResult struct {
	Metadata QueryMetadata
	Columns  []QueryColumn
	Rows     [][]any
}

type CachingStats struct {
	DataHits         int64 `json:"data_hits"`
	DataMisses       int64 `json:"data_misses"`
	DataBytesHits    int64 `json:"data_bytes_hits"`
	DataBytesMisses  int64 `json:"data_bytes_misses"`
	FilehandleHits   int64 `json:"filehandle_hits"`
	FilehandleMisses int64 `json:"filehandle_misses"`
	MetadataHits     int64 `json:"metadata_hits"`
	MetadataMisses   int64 `json:"metadata_misses"`
}

type MemoryStats struct {
	TotalUsageBytes int64 `json:"total_usage_bytes"`
}

type ScanStats struct {
	EstimatedResultRows  int64 `json:"estimated_result_rows"`
	EstimatedScannedRows int64 `json:"estimated_scanned_rows"`
}

type QueryStats struct {
	Caching *CachingStats `json:"caching"`
	Memory  *MemoryStats  `json:"memory"`
	Scan    *ScanStats    `json:"scan"`
}

type Progress struct {
	Percentage    float64 `json:"percentage"`
	RowsProcessed int64   `json:"rows_processed"`
	TotalRows     int64   `json:"total_rows"`
}

type QueryLogResponse struct {
	UUID            string      `json:"uuid"`
	StartTime       string      `json:"start_time"`
	EndTime         *string     `json:"end_time"`
	DurationMS      *int64      `json:"duration_ms"`
	Query           string      `json:"query"`
	SessionID       string      `json:"session_id"`
	ClientInterface SessionKind `json:"client_interface"`
	Error           *string     `json:"error"`
	Stats           *QueryStats `json:"stats"`
	Progress        *Progress   `json:"progress,omitempty"`
	Visible         bool        `json:"visible"`
	RequestedBy     *string     `json:"requested_by"`
	UserAgent       *string     `json:"user_agent"`
}

type CancelQueryResponse struct {
	Cancelled bool   `json:"cancelled"`
	Message   string `json:"message"`
}

type UpsertParams struct {
	Catalog     string
	Schema      string
	Table       string
	Mode        UpsertMode
	PrimaryKey  string
}

func (p UpsertParams) values() url.Values {
	values := url.Values{
		"catalog": []string{p.Catalog},
		"schema":  []string{p.Schema},
		"table":   []string{p.Table},
	}
	if p.Mode != "" {
		values.Set("mode", string(p.Mode))
	}
	if p.PrimaryKey != "" {
		values.Set("primary_key", p.PrimaryKey)
	}
	return values
}

type ValidateRequest struct {
	Statement string `json:"statement"`
}

type ValidateResponse struct {
	Valid             bool              `json:"valid"`
	Statement         string            `json:"statement"`
	ConnectionsErrors map[string]string `json:"connections_errors"`
	Error             *string           `json:"error"`
}

type AutocompleteRequest struct {
	Statement      string  `json:"statement"`
	Catalog        *string `json:"catalog,omitempty"`
	Schema         *string `json:"schema,omitempty"`
	SessionID      *string `json:"session_id,omitempty"`
	MaxSuggestions *int    `json:"max_suggestions,omitempty"`
}

// AutocompleteSuggestion matches the Lakehouse autocomplete payload for each entry.
type AutocompleteSuggestion struct {
	Suggestion      string  `json:"suggestion"`
	SuggestionStart int     `json:"suggestion_start"`
	SuggestionType  string  `json:"suggestion_type"`
	SuggestionScore float64 `json:"suggestion_score"`
}

// AutocompleteResponse is the decoded /autocomplete JSON body.
type AutocompleteResponse struct {
	Suggestions       []AutocompleteSuggestion `json:"suggestions"`
	Statement         string                   `json:"statement"`
	ConnectionsErrors map[string]string        `json:"connections_errors"`
}

func (r *AutocompleteResponse) UnmarshalJSON(data []byte) error {
	var probe struct {
		Suggestions       json.RawMessage   `json:"suggestions"`
		Statement         string            `json:"statement"`
		ConnectionsErrors map[string]string `json:"connections_errors"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	r.Statement = probe.Statement
	r.ConnectionsErrors = probe.ConnectionsErrors

	var asStrings []string
	if err := json.Unmarshal(probe.Suggestions, &asStrings); err == nil {
		r.Suggestions = make([]AutocompleteSuggestion, len(asStrings))
		for i, s := range asStrings {
			r.Suggestions[i] = AutocompleteSuggestion{Suggestion: s}
		}
		return nil
	}

	var asRich []AutocompleteSuggestion
	if err := json.Unmarshal(probe.Suggestions, &asRich); err != nil {
		return err
	}
	r.Suggestions = asRich
	return nil
}
