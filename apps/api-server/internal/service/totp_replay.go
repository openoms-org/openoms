package service

import (
	"crypto/subtle"
	"time"

	"github.com/pquerna/otp/totp"
)

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
