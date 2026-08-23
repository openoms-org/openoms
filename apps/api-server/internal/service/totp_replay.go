package service

import (
	"crypto/subtle"
	"time"

	"github.com/pquerna/otp/totp"
)

// totpPersistStepOutcome decides whether Verify2FALogin may issue tokens after
// attempting to record the accepted TOTP step. Persist errors are fail-closed:
// tokens must not be minted when single-use protection could not be written.
func totpPersistStepOutcome(advanced bool, persistErr error) error {
	if persistErr != nil {
		return persistErr
	}
	if !advanced {
		return ErrInvalid2FACode
	}
	return nil
}

// totpPeriod is the TOTP time-step in seconds and totpSkew is the number of
// periods of clock drift accepted on either side of the current time. These match
// the defaults used by totp.Validate (period 30, skew 1), which the rest of the
// codebase relies on — keep them in sync if totp.Validate's behavior is ever changed.
const (
	totpPeriod = 30
	totpSkew   = 1
)

// acceptedTOTPStep validates a TOTP code against a secret at time now and, when
// valid, returns the matching time-step (Unix/period counter). The valid window is
// [base-skew, base+skew] where base = floor(now/period), identical to the skew
// behavior of totp.Validate. The returned step is used for replay protection: a code
// whose step is not strictly greater than the last accepted step is a replay.
//
// It returns (0, false) when the code is invalid. Comparison is constant-time.
func acceptedTOTPStep(code, secret string, now time.Time) (int64, bool) {
	base := now.Unix() / totpPeriod
	for delta := int64(-totpSkew); delta <= int64(totpSkew); delta++ {
		step := base + delta
		generated, err := totp.GenerateCode(secret, time.Unix(step*totpPeriod, 0))
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(generated), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

// isTOTPReplay reports whether an accepted TOTP step has already been consumed, i.e. it
// is not strictly newer than the last accepted step recorded for the user. lastStep is
// nil when no successful 2FA login has been recorded yet (first login is never a replay).
// This is the exact predicate Verify2FALogin uses to reject a reused code; it is the
// fast, in-process pre-check. The authoritative single-use guarantee is enforced by the
// monotonic conditional UPDATE in SetTOTPLastUsedStep (which also closes the concurrent
// double-use race that a stale lastStep read cannot catch).
func isTOTPReplay(step int64, lastStep *int64) bool {
	return lastStep != nil && step <= *lastStep
}
