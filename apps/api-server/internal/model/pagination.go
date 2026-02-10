package model

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type ListResponse[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type PaginationParams struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

func ParsePagination(r *http.Request) PaginationParams {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	if offset < 0 {
		offset = 0
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
