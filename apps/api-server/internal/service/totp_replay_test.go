package service

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestTOTPSecret returns a fresh base32 TOTP secret usable with totp.GenerateCode/Validate.
func newTestTOTPSecret(t *testing.T) string {
	t.Helper()
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "OpenOMS", AccountName: "test@example.com"})
	require.NoError(t, err)
	return key.Secret()
}

func TestAcceptedTOTPStep_ValidCurrentCode(t *testing.T) {
	secret := newTestTOTPSecret(t)
	now := time.Now()

	code, err := totp.GenerateCode(secret, now)
	require.NoError(t, err)

	step, ok := acceptedTOTPStep(code, secret, now)
	require.True(t, ok, "current code must validate")
	assert.Equal(t, now.Unix()/totpPeriod, step, "step must match the current period counter")
}

func TestAcceptedTOTPStep_WrongCodeRejected(t *testing.T) {
	secret := newTestTOTPSecret(t)
	now := time.Now()

	// A code that does not match the current period. Derive the valid code and pick a
	// different value so the test never accidentally collides with the valid code.
	valid, err := totp.GenerateCode(secret, now)
	require.NoError(t, err)
	wrong := "000000"
	if valid == wrong {
		wrong = "111111"
	}

	step, ok := acceptedTOTPStep(wrong, secret, now)
	assert.False(t, ok, "wrong code must be rejected")
	assert.Equal(t, int64(0), step, "rejected code must report step 0")
}

func TestAcceptedTOTPStep_ReplayResolvesSameStep(t *testing.T) {
	secret := newTestTOTPSecret(t)
	now := time.Now()

	code, err := totp.GenerateCode(secret, now)
	require.NoError(t, err)

	step1, ok := acceptedTOTPStep(code, secret, now)
	require.True(t, ok)

	// Replaying the SAME code at the same time resolves to the SAME step. The service
	// rejects a code whose step is not strictly greater than the last accepted step,
	// so step2 <= step1 is the replay condition that closes the reuse window.
	step2, ok := acceptedTOTPStep(code, secret, now)
	require.True(t, ok)
	assert.Equal(t, step1, step2, "replayed code must resolve to the same step")
	assert.LessOrEqual(t, step2, step1, "replay must satisfy step <= lastStep (rejected by the service)")
}

func TestAcceptedTOTPStep_SkewAcceptsAdjacentWindows(t *testing.T) {
	secret := newTestTOTPSecret(t)
	base := time.Now().Truncate(totpPeriod * time.Second).Add(15 * time.Second) // mid-window
	baseStep := base.Unix() / totpPeriod

	code, err := totp.GenerateCode(secret, base)
	require.NoError(t, err)

	// Code from the previous evaluation point: still accepted one period later (skew +1),
	// and resolves to the ORIGINAL step (not the evaluator's current step).
	stepPrev, ok := acceptedTOTPStep(code, secret, base.Add(totpPeriod*time.Second))
	require.True(t, ok, "code must remain valid one period later within skew")
	assert.Equal(t, baseStep, stepPrev, "skew match must resolve to the code's own step")

	// Code evaluated one period early (skew -1) is also accepted and resolves to baseStep.
	stepNext, ok := acceptedTOTPStep(code, secret, base.Add(-totpPeriod*time.Second))
	require.True(t, ok, "code must validate one period early within skew")
	assert.Equal(t, baseStep, stepNext)

	// Two periods out is beyond the skew window and must be rejected.
	_, ok = acceptedTOTPStep(code, secret, base.Add(2*totpPeriod*time.Second))
	assert.False(t, ok, "code two periods out must be rejected (beyond skew)")
}

func TestAcceptedTOTPStep_NextWindowAdvancesStep(t *testing.T) {
	secret := newTestTOTPSecret(t)
	now := time.Now()

	step1, ok := acceptedTOTPStep(mustCode(t, secret, now), secret, now)
	require.True(t, ok)

	// A fresh code for the next window resolves to a strictly greater step, so a
	// legitimate subsequent login is accepted by the replay guard.
	next := now.Add(totpPeriod * time.Second)
	step2, ok := acceptedTOTPStep(mustCode(t, secret, next), secret, next)
	require.True(t, ok)
	assert.Greater(t, step2, step1, "next-window code must advance the step counter")
}

func mustCode(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, at)
	require.NoError(t, err)
	return code
}

// TestIsTOTPReplay covers the exact predicate Verify2FALogin uses to reject a reused code
// (the rejection path that was previously exercised only indirectly). A step is a replay
// when it is not strictly newer than the stored last-used step; the first-ever login (nil
// lastStep) is never a replay.
func TestIsTOTPReplay(t *testing.T) {
	step := func(v int64) *int64 { return &v }

	tests := []struct {
		name     string
		step     int64
		lastStep *int64
		want     bool
	}{
		{"first login (no prior step) is not a replay", 100, nil, false},
		{"same step as last is a replay", 100, step(100), true},
		{"older step than last is a replay", 99, step(100), true},
		{"strictly newer step is not a replay", 101, step(100), false},
		{"newer step after a NULL-equivalent zero baseline is not a replay", 1, step(0), false},
		{"equal zero baseline is a replay", 0, step(0), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isTOTPReplay(tc.step, tc.lastStep))
		})
	}
}
