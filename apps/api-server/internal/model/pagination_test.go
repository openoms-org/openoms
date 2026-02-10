package model

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePagination(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		r := &http.Request{URL: &url.URL{}}
		p := ParsePagination(r)
		assert.Equal(t, DefaultLimit, p.Limit)
		assert.Equal(t, 0, p.Offset)
		assert.Equal(t, "desc", p.SortOrder)
	})

	t.Run("custom values", func(t *testing.T) {
		r := &http.Request{URL: &url.URL{RawQuery: "limit=50&offset=10&sort_by=total_amount&sort_order=asc"}}
		p := ParsePagination(r)
		assert.Equal(t, 50, p.Limit)
		assert.Equal(t, 10, p.Offset)
		assert.Equal(t, "total_amount", p.SortBy)
		assert.Equal(t, "asc", p.SortOrder)
	})

	t.Run("limit exceeds max", func(t *testing.T) {
		r := &http.Request{URL: &url.URL{RawQuery: "limit=999"}}
		p := ParsePagination(r)
		assert.Equal(t, DefaultLimit, p.Limit)
	})

	t.Run("negative offset clamped", func(t *testing.T) {
		r := &http.Request{URL: &url.URL{RawQuery: "offset=-5"}}
		p := ParsePagination(r)
		assert.Equal(t, 0, p.Offset)
	})

	t.Run("invalid sort_order defaults to desc", func(t *testing.T) {
		r := &http.Request{URL: &url.URL{RawQuery: "sort_order=INVALID"}}
		p := ParsePagination(r)
		assert.Equal(t, "desc", p.SortOrder)
	})
}

func TestBuildOrderByClause(t *testing.T) {
	allowed := map[string]string{
		"total_amount": "total_amount",
		"created_at":   "created_at",
		"customer":     "customer_name",
	}

	t.Run("valid sort field", func(t *testing.T) {
		clause := BuildOrderByClause("total_amount", "asc", allowed)
		assert.Equal(t, "ORDER BY total_amount ASC", clause)
	})

	t.Run("mapped field name", func(t *testing.T) {
		clause := BuildOrderByClause("customer", "desc", allowed)
		assert.Equal(t, "ORDER BY customer_name DESC", clause)
	})

	t.Run("unknown sort field falls back", func(t *testing.T) {
		clause := BuildOrderByClause("nonexistent", "asc", allowed)
		assert.Equal(t, "ORDER BY created_at DESC", clause)
	})

	t.Run("invalid direction defaults to DESC", func(t *testing.T) {
		clause := BuildOrderByClause("total_amount", "INVALID", allowed)
		assert.Equal(t, "ORDER BY total_amount DESC", clause)
	})
}
