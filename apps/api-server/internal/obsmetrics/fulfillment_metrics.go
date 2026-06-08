// Package obsmetrics holds additive, dependency-free Prometheus-compatible
// collectors for the fulfillment / orchestration / provider-validation paths
// (OPE-422). It deliberately reuses the same hand-rolled atomic-counter +
// Prometheus text-exposition approach as internal/middleware.MetricsCollector —
// this codebase has NO prometheus client library dependency and does not add one.
//
// CARDINALITY DISCIPLINE (critical): every metric label is a BOUNDED, low-cardinality
// enum (operation, status, category, result, state). High-cardinality identifiers —
// tenant_id, order_id, process_id, unit_id, step_id, user_id, or any UUID — are NEVER
// used as labels. Any value passed to a record method is validated against its bounded
// allow-list and coerced to "other" when unknown, so a calling bug cannot cause a
// cardinality explosion. Identifiers belong in logs and audit entries, not in metric
// label sets.
package obsmetrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// labelOther is the bounded fallback bucket for any value outside its allow-list.
const labelOther = "other"

// Bounded label allow-lists. These mirror the domain enums in internal/model but are
// duplicated here intentionally: the metrics package stays dependency-free of the
// domain model, and the allow-lists are what enforce cardinality at the metric edge.
var (
	allowedOperations = set("create_shipment", "generate_label", "download_label", "sync_tracking")
	allowedStatuses   = set("pending", "succeeded", "failed", "ready", "running",
		"waiting_external", "blocked", "cancelled", "skipped")
	allowedCategories         = set("integration", "supplier", "operator", "capability", "mapping")
	allowedOutboxResults      = set("claimed", "processed", "failed")
	allowedValidationVerdicts = set("pending", "passed", "failed", "error")
	allowedPublicationStates  = set("research", "designed", "adapter_in_progress", "internal_validation",
		"private_beta", "available", "deprecated", "retired")
)

func set(values ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	return m
}

// coerce returns value when it is in allowed, otherwise the bounded "other" bucket.
func coerce(value string, allowed map[string]struct{}) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return labelOther
}

// counter is a label-keyed set of atomic counters guarded by a RWMutex, matching the
// concurrency pattern used by middleware.MetricsCollector.
type counter struct {
	mu     sync.RWMutex
	values map[string]*atomic.Int64
}

func newCounter() *counter { return &counter{values: make(map[string]*atomic.Int64)} }

func (c *counter) inc(key string) {
	c.mu.RLock()
	v, ok := c.values[key]
	c.mu.RUnlock()
	if !ok {
		c.mu.Lock()
		if v, ok = c.values[key]; !ok {
			v = &atomic.Int64{}
			c.values[key] = v
		}
		c.mu.Unlock()
	}
	v.Add(1)
}

// snapshot returns a sorted copy of the counter's keys + current values.
func (c *counter) snapshot() []counterSample {
	c.mu.RLock()
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	c.mu.RUnlock()
	sort.Strings(keys)
	out := make([]counterSample, 0, len(keys))
	for _, k := range keys {
		c.mu.RLock()
		v := c.values[k]
		c.mu.RUnlock()
		if v != nil {
			out = append(out, counterSample{key: k, value: v.Load()})
		}
	}
	return out
}

type counterSample struct {
	key   string
	value int64
}

// FulfillmentMetrics is the additive collector for the OPE-414..419 fulfillment /
// orchestration paths and the provider validation / publication paths. Every method
// is safe to call on a nil receiver (returns immediately), so callers can record
// best-effort without a nil check and a missing collector is a no-op.
type FulfillmentMetrics struct {
	providerAttempts   *counter // labels: operation, status
	blockers           *counter // labels: category
	outboxEvents       *counter // labels: result
	unitTransitions    *counter // labels: status
	stepTransitions    *counter // labels: status
	validationRuns     *counter // labels: result (verdict)
	publicationMoves   *counter // labels: state (target)
	validationFailures atomic.Int64

	outboxQueueDepth atomic.Int64
	// stuckProcesses / blockedProcesses are GLOBAL (cross-tenant) gauges published by a
	// periodic sweep on the privileged worker pool (OPE-422 followup). They carry NO
	// tenant label — a single aggregate across all tenants — so they are bounded and do
	// not flap (unlike the earlier, removed, per-tenant-read version).
	stuckProcesses   atomic.Int64
	blockedProcesses atomic.Int64
}

// NewFulfillmentMetrics creates an initialized collector.
func NewFulfillmentMetrics() *FulfillmentMetrics {
	return &FulfillmentMetrics{
		providerAttempts: newCounter(),
		blockers:         newCounter(),
		outboxEvents:     newCounter(),
		unitTransitions:  newCounter(),
		stepTransitions:  newCounter(),
		validationRuns:   newCounter(),
		publicationMoves: newCounter(),
	}
}

// RecordProviderAttempt counts a provider attempt by bounded (operation, status).
func (m *FulfillmentMetrics) RecordProviderAttempt(operation, status string) {
	if m == nil {
		return
	}
	op := coerce(operation, allowedOperations)
	st := coerce(status, allowedStatuses)
	m.providerAttempts.inc("operation=" + op + "\x00status=" + st)
}

// RecordBlocker counts a fulfillment blocker by bounded category.
func (m *FulfillmentMetrics) RecordBlocker(category string) {
	if m == nil {
		return
	}
	m.blockers.inc("category=" + coerce(category, allowedCategories))
}

// RecordOutboxEvent counts an orchestration outbox event by bounded result
// (claimed | processed | failed).
func (m *FulfillmentMetrics) RecordOutboxEvent(result string) {
	if m == nil {
		return
	}
	m.outboxEvents.inc("result=" + coerce(result, allowedOutboxResults))
}

// RecordUnitTransition counts a fulfillment unit status transition.
func (m *FulfillmentMetrics) RecordUnitTransition(status string) {
	if m == nil {
		return
	}
	m.unitTransitions.inc("status=" + coerce(status, allowedStatuses))
}

// RecordStepTransition counts a fulfillment step status transition.
func (m *FulfillmentMetrics) RecordStepTransition(status string) {
	if m == nil {
		return
	}
	m.stepTransitions.inc("status=" + coerce(status, allowedStatuses))
}

// RecordValidationRun counts a completed provider validation run by bounded verdict.
func (m *FulfillmentMetrics) RecordValidationRun(verdict string) {
	if m == nil {
		return
	}
	m.validationRuns.inc("result=" + coerce(verdict, allowedValidationVerdicts))
}

// RecordValidationFailure counts one failed/errored validation probe result.
func (m *FulfillmentMetrics) RecordValidationFailure() {
	if m == nil {
		return
	}
	m.validationFailures.Add(1)
}

// RecordPublicationTransition counts a provider-version publication transition by the
// bounded target state.
func (m *FulfillmentMetrics) RecordPublicationTransition(toState string) {
	if m == nil {
		return
	}
	m.publicationMoves.inc("state=" + coerce(toState, allowedPublicationStates))
}

// SetOutboxQueueDepth sets the current pending orchestration outbox depth gauge.
func (m *FulfillmentMetrics) SetOutboxQueueDepth(n int) {
	if m == nil {
		return
	}
	m.outboxQueueDepth.Store(int64(n))
}

// SetStuckProcesses sets the GLOBAL (cross-tenant) stuck-process gauge — processes in a
// system-error/unhealthy state across all tenants. Set by the gauge sweeper (OPE-422).
func (m *FulfillmentMetrics) SetStuckProcesses(n int) {
	if m == nil {
		return
	}
	m.stuckProcesses.Store(int64(n))
}

// SetBlockedProcesses sets the GLOBAL (cross-tenant) blocked-process gauge — processes
// parked on an open blocker across all tenants. Set by the gauge sweeper (OPE-422).
func (m *FulfillmentMetrics) SetBlockedProcesses(n int) {
	if m == nil {
		return
	}
	m.blockedProcesses.Store(int64(n))
}

// labelKeyToProm converts an internal "k1=v1\x00k2=v2" counter key into a Prometheus
// label set "{k1=\"v1\",k2=\"v2\"}". An empty key yields an empty string (no braces).
func labelKeyToProm(key string) string {
	if key == "" {
		return ""
	}
	pairs := strings.Split(key, "\x00")
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%q", kv[0], kv[1]))
	}
	if len(parts) == 0 {
		return ""
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// Render appends this collector's metrics to b in Prometheus text exposition format.
// It satisfies the structural PromCollector contract consumed by
// middleware.MetricsCollector, so the existing /metrics handler exposes these too. A
// nil receiver renders nothing.
func (m *FulfillmentMetrics) Render(b *strings.Builder) {
	if m == nil {
		return
	}

	writeCounter(b, "openoms_fulfillment_provider_attempts_total",
		"Provider attempts by operation and status.", m.providerAttempts)
	writeCounter(b, "openoms_fulfillment_blockers_total",
		"Fulfillment blockers created by category.", m.blockers)
	writeCounter(b, "openoms_orchestration_outbox_events_total",
		"Orchestration outbox events by result.", m.outboxEvents)
	writeCounter(b, "openoms_fulfillment_unit_transitions_total",
		"Fulfillment unit status transitions by status.", m.unitTransitions)
	writeCounter(b, "openoms_fulfillment_step_transitions_total",
		"Fulfillment step status transitions by status.", m.stepTransitions)
	writeCounter(b, "openoms_provider_validation_runs_total",
		"Completed provider validation runs by verdict.", m.validationRuns)
	writeCounter(b, "openoms_provider_publication_transitions_total",
		"Provider version publication transitions by target state.", m.publicationMoves)

	writeScalarCounter(b, "openoms_provider_validation_failures_total",
		"Failed or errored provider validation probe results.", m.validationFailures.Load())

	writeGauge(b, "openoms_orchestration_outbox_queue_depth",
		"Pending orchestration outbox events awaiting processing.", m.outboxQueueDepth.Load())
	writeGauge(b, "openoms_fulfillment_stuck_processes",
		"Fulfillment processes in a stuck (system-error) state, all tenants.", m.stuckProcesses.Load())
	writeGauge(b, "openoms_fulfillment_blocked_processes",
		"Fulfillment processes currently blocked, all tenants.", m.blockedProcesses.Load())
}

func writeCounter(b *strings.Builder, name, help string, c *counter) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s counter\n", name)
	for _, s := range c.snapshot() {
		fmt.Fprintf(b, "%s%s %d\n", name, labelKeyToProm(s.key), s.value)
	}
}

func writeScalarCounter(b *strings.Builder, name, help string, value int64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s counter\n", name)
	fmt.Fprintf(b, "%s %d\n", name, value)
}

func writeGauge(b *strings.Builder, name, help string, value int64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s gauge\n", name)
	fmt.Fprintf(b, "%s %d\n", name, value)
}
