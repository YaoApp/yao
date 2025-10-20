package seed

// DuplicateMode the duplicate mode
type DuplicateMode string

// ImportMode the import mode
type ImportMode string

const (

	// ImportModeBatch the batch import mode
	ImportModeBatch ImportMode = "batch"
	// ImportModeEach the each import mode
	ImportModeEach ImportMode = "each"

	// DuplicateIgnore when the record is duplicate, ignore the record
	DuplicateIgnore DuplicateMode = "ignore"
	// DuplicateUpdate when the record is duplicate, update the record
	DuplicateUpdate DuplicateMode = "update"
	// DuplicateError when the record is duplicate, raise an error
	DuplicateError DuplicateMode = "error"
	// DuplicateAbort when the record is duplicate, abort the record
	DuplicateAbort DuplicateMode = "abort"
)

const (
	// ChunkSizeDefault the default chunk size
	ChunkSizeDefault = 500
)

// ImportOption the seed import option
type ImportOption struct {
	ChunkSize int           `json:"chunk_size,omitempty"`
	Duplicate DuplicateMode `json:"duplicate,omitempty"`
	Mode      ImportMode    `json:"mode,omitempty"`
}

// ImportHandler the seed import handler
type ImportHandler func(line int, data [][]interface{}) error

// ImportResult the seed import result
type ImportResult struct {
	Total   int           `json:"total,omitempty"`
	Success int           `json:"success,omitempty"`
	Failure int           `json:"failure,omitempty"`
	Ignore  int           `json:"ignore,omitempty"`
	Errors  []ImportError `json:"errors,omitempty"`
}

// ImportError the seed import error
type ImportError struct {
	Row     int           `json:"row,omitempty"`
	Message string        `json:"message,omitempty"`
	Code    int           `json:"code,omitempty"`
	Data    []interface{} `json:"data,omitempty"`
}
