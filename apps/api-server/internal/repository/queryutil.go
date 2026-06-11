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

// UpdateBuilder helps construct parameterized SET clauses for partial UPDATE statements.
// It is the SET-clause sibling of QueryBuilder.
type UpdateBuilder struct {
	setClauses []string
	args       []any
	argIdx     int
}

// NewUpdateBuilder creates a new UpdateBuilder with arg indices starting at 1.
func NewUpdateBuilder() *UpdateBuilder {
	return &UpdateBuilder{argIdx: 1}
}

// Set appends "column = $N" with the given value and advances the arg index.
func (ub *UpdateBuilder) Set(column string, value any) {
	ub.setClauses = append(ub.setClauses, fmt.Sprintf("%s = $%d", column, ub.argIdx))
	ub.args = append(ub.args, value)
	ub.argIdx++
}

// SetRaw appends a raw SET expression with no parameter (e.g., "updated_at = NOW()").
func (ub *UpdateBuilder) SetRaw(clause string) {
	ub.setClauses = append(ub.setClauses, clause)
}

// IsEmpty reports whether no parameterized SET clauses have been added.
func (ub *UpdateBuilder) IsEmpty() bool {
	return len(ub.setClauses) == 0
}

// Len returns the number of accumulated SET clauses (parameterized and raw).
func (ub *UpdateBuilder) Len() int {
	return len(ub.setClauses)
}

// SetClause returns the joined SET clauses (without the leading SET keyword).
func (ub *UpdateBuilder) SetClause() string {
	return strings.Join(ub.setClauses, ", ")
}

// Args returns all accumulated parameter values.
func (ub *UpdateBuilder) Args() []any {
	return ub.args
}

// NextArgIdx returns the next available parameter index (e.g., for a trailing WHERE placeholder).
func (ub *UpdateBuilder) NextArgIdx() int {
	return ub.argIdx
}

// AddArg appends a single trailing arg (e.g., a WHERE id value) and returns its placeholder index.
func (ub *UpdateBuilder) AddArg(value any) int {
	idx := ub.argIdx
	ub.args = append(ub.args, value)
	ub.argIdx++
	return idx
}

// SetPtr appends "column = $N" with the dereferenced pointer value when ptr is non-nil.
// It is a free function because Go does not support generic methods.
func SetPtr[T any](ub *UpdateBuilder, column string, ptr *T) {
	if ptr == nil {
		return
	}
	ub.Set(column, *ptr)
}
