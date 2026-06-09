package model

import "time"

// DefaultFreshnessWindowSeconds is used when neither the policy nor the snapshot's feed
// declares one — an hour is a safe conservative default for supplier feeds.
const DefaultFreshnessWindowSeconds = 3600

// Availability decision statuses.
const (
	AvailabilityStatusTrusted        = "trusted"
	AvailabilityStatusStale          = "stale"
	AvailabilityStatusUnknown        = "unknown"
	AvailabilityStatusPaused         = "paused"
	AvailabilityStatusManualOverride = "manual_override"
)

// EffectivePolicy is the policy after the 4-scope precedence chain is resolved. It mirrors
// the rule fields of SupplierAvailabilityPolicy but with concrete (already-resolved) values.
type EffectivePolicy struct {
	Mode                   string
	SafetyBuffer           int
	FreshnessWindowSeconds int
	MaxLeadTimeDays        *int
	OverrideQuantity       *int
	AllowChannelIncrease   bool
	RequireReservation     bool
	RequirePreflight       bool
}

// AvailabilityDecision is the resolver output consumed by routing + propagation.
type AvailabilityDecision struct {
	AvailableToSell        int
	Status                 string
	AutoRoutable           bool
	Backorder              bool
	ChannelIncreaseAllowed bool
	BlockerCode            *string // nil when nothing is wrong
}

// ResolvePolicyChain folds a precedence-ordered slice of scope policies (LEAST specific
// first: supplier, product, listing, channel) into one EffectivePolicy. Each field is
// taken from the MOST specific policy that sets it; unset fields inherit. A zero
// SafetyBuffer / false bool from a more specific scope is treated as "set" (explicit),
// so callers should only include policy rows that actually exist for the context. An
// empty Mode is treated as "inherit" so a row that overrides only e.g. safety_buffer does
// not clobber a less-specific scope's mode.
func ResolvePolicyChain(chain []SupplierAvailabilityPolicy) EffectivePolicy {
	eff := EffectivePolicy{Mode: PolicyModeAuto, FreshnessWindowSeconds: 0}
	for _, p := range chain { // chain is ordered least->most specific; later wins
		if p.Mode != "" {
			eff.Mode = p.Mode
		}
		eff.SafetyBuffer = p.SafetyBuffer
		if p.FreshnessWindowSecs != nil {
			eff.FreshnessWindowSeconds = *p.FreshnessWindowSecs
		}
		if p.MaxLeadTimeDays != nil {
			eff.MaxLeadTimeDays = p.MaxLeadTimeDays
		}
		if p.OverrideQuantity != nil {
			eff.OverrideQuantity = p.OverrideQuantity
		}
		eff.AllowChannelIncrease = p.AllowChannelIncrease
		eff.RequireReservation = p.RequireReservation
		eff.RequirePreflight = p.RequirePreflight
	}
	return eff
}

// ResolveAvailability turns a snapshot + resolved policy into a decision. PURE (no DB).
// requestedQty is the quantity the order line needs; preflightSupported reflects whether
// the supplier capability supports a preflight check (false until the supplier-order
// engine spec lands, so require_preflight conservatively yields a manual unit).
func ResolveAvailability(snap SupplierAvailability, pol EffectivePolicy, requestedQty int, preflightSupported bool, now time.Time) AvailabilityDecision {
	d := AvailabilityDecision{}

	// 1. Manual override wins outright.
	if pol.OverrideQuantity != nil {
		d.Status = AvailabilityStatusManualOverride
		d.AvailableToSell = max(0, *pol.OverrideQuantity)
		d.AutoRoutable = d.AvailableToSell >= requestedQty && pol.Mode != PolicyModePaused
		d.ChannelIncreaseAllowed = false // a manual override never auto-raises a channel
		return d
	}

	// 2. Operator pause.
	if pol.Mode == PolicyModePaused {
		d.Status = AvailabilityStatusPaused
		return d // zero, not routable, no blocker (intentional)
	}

	// 3. Unknown availability is untrusted.
	if snap.AvailabilityType == AvailabilityUnknown {
		d.Status = AvailabilityStatusUnknown
		code := BlockerSupplierAvailabilityUnknown
		d.BlockerCode = &code
		return d
	}

	// 4. Freshness.
	window := pol.FreshnessWindowSeconds
	if window <= 0 {
		if snap.SourceMaxStaleSeconds != nil && *snap.SourceMaxStaleSeconds > 0 {
			window = *snap.SourceMaxStaleSeconds
		} else {
			window = DefaultFreshnessWindowSeconds
		}
	}
	if now.Sub(snap.FreshnessObservedAt) > time.Duration(window)*time.Second {
		d.Status = AvailabilityStatusStale
		code := BlockerSupplierAvailabilityStale
		d.BlockerCode = &code
		return d
	}

	// 5. Trusted.
	d.Status = AvailabilityStatusTrusted
	d.AvailableToSell = max(0, snap.SourceQuantity-pol.SafetyBuffer)
	d.ChannelIncreaseAllowed = pol.AllowChannelIncrease && leadTimeOK(snap, pol)

	if d.AvailableToSell >= requestedQty &&
		(!pol.RequireReservation || snap.ReservationSupported) &&
		(!pol.RequirePreflight || preflightSupported) &&
		leadTimeOK(snap, pol) {
		d.AutoRoutable = true
		return d
	}

	// Trusted but cannot auto-route: backorder if an ETA exists, else insufficient blocker.
	if d.AvailableToSell < requestedQty {
		if snap.NextDeliveryDate != nil {
			d.Backorder = true
			return d
		}
		code := BlockerSupplierAvailabilityInsufficient
		d.BlockerCode = &code
	}
	return d
}

// leadTimeOK reports whether the supplier handling time fits the policy lead-time cap.
func leadTimeOK(snap SupplierAvailability, pol EffectivePolicy) bool {
	if pol.MaxLeadTimeDays == nil {
		return true
	}
	if snap.MaxHandlingDays == nil {
		return true // no handling data — do not block on lead time
	}
	return *snap.MaxHandlingDays <= *pol.MaxLeadTimeDays
}
