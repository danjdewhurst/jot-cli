package model

import "time"

// ExportVersion is the current export format version.
const ExportVersion = 1

// ExportEnvelope wraps exported notes with metadata for format versioning.
type ExportEnvelope struct {
	Version    int       `json:"version"`
	ExportedAt time.Time `json:"exported_at"`
	Count      int       `json:"count"`
	Notes      []Note    `json:"notes"`
}

// ImportResult summarises the outcome of an import operation.
type ImportResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}
