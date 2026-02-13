package model

// ImportColumnMapping maps a CSV column header to an order field.
type ImportColumnMapping struct {
	CSVColumn  string `json:"csv_column"`
	OrderField string `json:"order_field"`
}

// ImportPreviewRow represents one row from the CSV preview.
type ImportPreviewRow struct {
	Row    int            `json:"row"`
	Data   map[string]any `json:"data"`
	Errors []string       `json:"errors,omitempty"`
}

// ImportPreviewResponse is returned by the CSV preview endpoint.
type ImportPreviewResponse struct {
	Headers    []string              `json:"headers"`
	TotalRows  int                   `json:"total_rows"`
	SampleRows []ImportPreviewRow    `json:"sample_rows"`
	Mappings   []ImportColumnMapping `json:"mappings,omitempty"`
}

// ImportResult is the summary returned after a batch import.
type ImportResult struct {
	TotalRows int           `json:"total_rows"`
	Imported  int           `json:"imported"`
	Skipped   int           `json:"skipped"`
	Errors    []ImportError `json:"errors"`
}

// ImportError describes a single per-row error during import.
type ImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}
