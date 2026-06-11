package model

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Pagination defaults and limits.
const (
	DefaultLimit = 20
	MaxLimit     = 100
	MaxOffset    = 100000
)

// ListResponse is a generic paginated list response.
type ListResponse[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// NewListResponse builds a ListResponse, normalizing a nil items slice to an
// empty slice so JSON serialization emits [] instead of null.
func NewListResponse[T any](items []T, total, limit, offset int) ListResponse[T] {
	if items == nil {
		items = []T{}
	}
	return ListResponse[T]{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

// PaginationParams holds parsed limit, offset and sort parameters.
type PaginationParams struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// ParsePagination extracts pagination parameters from an HTTP request's query string.
func ParsePagination(r *http.Request) PaginationParams {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	if offset < 0 {
		offset = 0
	}
	if offset > MaxOffset {
		offset = MaxOffset
	}

	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := strings.ToLower(r.URL.Query().Get("sort_order"))
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	return PaginationParams{Limit: limit, Offset: offset, SortBy: sortBy, SortOrder: sortOrder}
}

// BuildOrderByClause builds a safe ORDER BY clause from user input.
// allowed maps API field names to actual database column names.
// If sortBy is not in the allowlist, it falls back to "ORDER BY created_at DESC".
func BuildOrderByClause(sortBy, sortOrder string, allowed map[string]string) string {
	direction := strings.ToUpper(sortOrder)
	if direction != "ASC" && direction != "DESC" {
		direction = "DESC"
	}

	if dbColumn, ok := allowed[sortBy]; ok {
		return fmt.Sprintf("ORDER BY %s %s", dbColumn, direction)
	}
	return "ORDER BY created_at DESC"
}
