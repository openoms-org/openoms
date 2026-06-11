package repository

import (
	"fmt"
	"strings"
)

// QueryBuilder helps construct parameterized WHERE clauses for SQL queries.
type QueryBuilder struct {
	conditions []string
	args       []any
	argIdx     int
}

// NewQueryBuilder creates a new QueryBuilder with arg indices starting at 1.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{argIdx: 1}
}

// Add appends a condition with a single parameter placeholder.
// Example: qb.Add("p.name ILIKE '%%' || $%d || '%%'", name)
func (qb *QueryBuilder) Add(template string, value any) {
	qb.conditions = append(qb.conditions, fmt.Sprintf(template, qb.argIdx))
	qb.args = append(qb.args, value)
	qb.argIdx++
}

// AddMultiRef appends a condition where the same parameter index appears multiple times.
// The template should use %d for each occurrence of the parameter index.
func (qb *QueryBuilder) AddMultiRef(template string, refCount int, value any) {
	indices := make([]any, refCount)
	for i := range indices {
		indices[i] = qb.argIdx
	}
	qb.conditions = append(qb.conditions, fmt.Sprintf(template, indices...))
	qb.args = append(qb.args, value)
	qb.argIdx++
}

// AddRaw appends a raw condition with no parameter (e.g., subqueries).
func (qb *QueryBuilder) AddRaw(condition string) {
	qb.conditions = append(qb.conditions, condition)
}

// WhereClause returns the WHERE clause string (empty string if no conditions).
func (qb *QueryBuilder) WhereClause() string {
	if len(qb.conditions) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(qb.conditions, " AND ")
}

// Args returns all accumulated parameter values.
func (qb *QueryBuilder) Args() []any {
	return qb.args
}

// AddArgs appends additional args (e.g., for LIMIT/OFFSET) and returns their starting index.
func (qb *QueryBuilder) AddArgs(values ...any) int {
	startIdx := qb.argIdx
	qb.args = append(qb.args, values...)
	qb.argIdx += len(values)
	return startIdx
}
