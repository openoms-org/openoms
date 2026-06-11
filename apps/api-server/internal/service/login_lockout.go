package service

import (
	"context"
	"time"
)

// LoginLockoutStore abstracts the storage for login attempt tracking.
type LoginLockoutStore interface {
	// IncrFailures increments failure count, returns new count. Sets TTL on first failure.
	IncrFailures(ctx context.Context, key string, window time.Duration) (int, error)
	// ResetFailures clears the failure counter (on successful login).
	ResetFailures(ctx context.Context, key string) error
	// GetLockoutRemaining returns remaining lockout duration, or 0 if not locked.
	GetLockoutRemaining(ctx context.Context, key string) (time.Duration, error)
	// SetLockout sets a lockout key with the given duration.
	SetLockout(ctx context.Context, key string, duration time.Duration) error
}

const (
	maxLoginAttempts = 5
	lockoutPrefix    = "login_lockout:"
	failurePrefix    = "login_failures:"
	// 2FA brute-force tracking uses a SEPARATE key namespace from password login.
	// This is deliberate: the password-login counter is reset on a successful
	// password check (ResetOnSuccess in Login), so if 2FA shared that namespace an
	// attacker who knows the password could reset the 2FA failure counter at will by
	// re-authenticating, defeating the per-account lockout. The 2FA counter is keyed
	// on the immutable tenant+user IDs carried by the 2fa_pending token and is only
	// reset by a successful TOTP verification.
	lockout2FAPrefix = "2fa_lockout:"
	failure2FAPrefix = "2fa_failures:"
)

// lockoutDuration returns exponential backoff: 1m, 5m, 15m, 30m.
func lockoutDuration(failures int) time.Duration {
	switch {
	case failures <= 5:
		return 1 * time.Minute
	case failures <= 7:
		return 5 * time.Minute
	case failures <= 10:
		return 15 * time.Minute
	default:
		return 30 * time.Minute
	}
}

// LoginLockout tracks failed login attempts per account.
type LoginLockout struct {
	store LoginLockoutStore
}

// NewLoginLockout creates a new LoginLockout with the given store.
func NewLoginLockout(store LoginLockoutStore) *LoginLockout {
	return &LoginLockout{store: store}
}

// CheckLocked returns the remaining lockout duration if the account is currently locked, or 0 if not.
func (l *LoginLockout) CheckLocked(ctx context.Context, tenantSlug, email string) (time.Duration, error) {
	if l == nil || l.store == nil {
		return 0, nil // fail open if no store configured
	}
	key := lockoutPrefix + tenantSlug + ":" + email
	return l.store.GetLockoutRemaining(ctx, key)
}

// RecordFailure increments the failure counter and sets a lockout if threshold exceeded.
func (l *LoginLockout) RecordFailure(ctx context.Context, tenantSlug, email string) error {
	if l == nil || l.store == nil {
		return nil
	}
	failKey := failurePrefix + tenantSlug + ":" + email
	count, err := l.store.IncrFailures(ctx, failKey, 30*time.Minute)
	if err != nil {
		return nil //nolint:nilerr // fail open on Redis error — lockout is best-effort
	}
	if count >= maxLoginAttempts {
		lockKey := lockoutPrefix + tenantSlug + ":" + email
		dur := lockoutDuration(count)
		return l.store.SetLockout(ctx, lockKey, dur)
	}
	return nil
}

// ResetOnSuccess clears failure tracking after successful login.
func (l *LoginLockout) ResetOnSuccess(ctx context.Context, tenantSlug, email string) {
	if l == nil || l.store == nil {
		return
	}
	failKey := failurePrefix + tenantSlug + ":" + email
	lockKey := lockoutPrefix + tenantSlug + ":" + email
	_ = l.store.ResetFailures(ctx, failKey)
	_ = l.store.ResetFailures(ctx, lockKey)
}

// twofaKey builds the lockout/failure key suffix for a 2FA attempt. tenantID and
// userID are the immutable identifiers carried by the 2fa_pending token.
func twofaKey(tenantID, userID string) string {
	return tenantID + ":" + userID
}

// Check2FALocked returns the remaining 2FA lockout duration for an account, or 0 if not locked.
// Mirrors CheckLocked but uses the dedicated 2FA key namespace.
func (l *LoginLockout) Check2FALocked(ctx context.Context, tenantID, userID string) (time.Duration, error) {
	if l == nil || l.store == nil {
		return 0, nil // fail open if no store configured
	}
	return l.store.GetLockoutRemaining(ctx, lockout2FAPrefix+twofaKey(tenantID, userID))
}

// Record2FAFailure increments the 2FA failure counter and sets a lockout once the
// threshold is exceeded. Uses the same threshold and exponential backoff as password login.
func (l *LoginLockout) Record2FAFailure(ctx context.Context, tenantID, userID string) error {
	if l == nil || l.store == nil {
		return nil
	}
	count, err := l.store.IncrFailures(ctx, failure2FAPrefix+twofaKey(tenantID, userID), 30*time.Minute)
	if err != nil {
		return nil //nolint:nilerr // fail open on store error — lockout is best-effort
	}
	if count >= maxLoginAttempts {
		return l.store.SetLockout(ctx, lockout2FAPrefix+twofaKey(tenantID, userID), lockoutDuration(count))
	}
	return nil
}

// Reset2FAOnSuccess clears 2FA failure tracking after a successful TOTP verification.
func (l *LoginLockout) Reset2FAOnSuccess(ctx context.Context, tenantID, userID string) {
	if l == nil || l.store == nil {
		return
	}
	_ = l.store.ResetFailures(ctx, failure2FAPrefix+twofaKey(tenantID, userID))
	_ = l.store.ResetFailures(ctx, lockout2FAPrefix+twofaKey(tenantID, userID))
}
