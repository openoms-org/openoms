package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateProbes(t *testing.T) {
	assert.NoError(t, ValidateProbes([]ProviderValidationProbe{
		{Label: "auth", ProbeType: ProbeAuthCheck},
		{Label: "create", ProbeType: ProbeSandboxOrderCreate, Destructive: true},
	}))
	assert.Error(t, ValidateProbes([]ProviderValidationProbe{{Label: "", ProbeType: ProbeAuthCheck}}))
	assert.Error(t, ValidateProbes([]ProviderValidationProbe{{Label: "x", ProbeType: "telepathy"}}))
	assert.Error(t, ValidateProbes([]ProviderValidationProbe{{Label: "d", ProbeType: ProbeAuthCheck}, {Label: "d", ProbeType: ProbeFeedFetch}}))
}

func TestVerdictFromResults(t *testing.T) {
	assert.Equal(t, RunVerdictError, VerdictFromResults(nil))
	assert.Equal(t, RunVerdictPassed, VerdictFromResults([]ProviderValidationResult{{Status: ResultStatusPassed}, {Status: ResultStatusSkipped}}))
	assert.Equal(t, RunVerdictFailed, VerdictFromResults([]ProviderValidationResult{{Status: ResultStatusPassed}, {Status: ResultStatusFailed}}))
	assert.Equal(t, RunVerdictError, VerdictFromResults([]ProviderValidationResult{{Status: ResultStatusFailed}, {Status: ResultStatusError}}))
}

func TestGapForFailedProbe(t *testing.T) {
	gt, sev := GapForFailedProbe(ProbeAuthCheck, ResultStatusFailed)
	assert.Equal(t, GapTypeAuthFailure, gt)
	assert.Equal(t, GapSeverityActionRequired, sev)

	gt, sev = GapForFailedProbe(ProbeFeedParse, ResultStatusError)
	assert.Equal(t, GapTypeParserFailure, gt)
	assert.Equal(t, GapSeveritySystemError, sev)

	gt, _ = GapForFailedProbe(ProbeOrderStatusRead, ResultStatusFailed)
	assert.Equal(t, GapTypeProviderBusinessError, gt)

	gt, sev = GapForFailedProbe(ProbeAuthCheck, ResultStatusPassed)
	assert.Equal(t, "", gt)
	assert.Equal(t, "", sev)
}

func TestValidationEnumValidatorsAndHash(t *testing.T) {
	assert.True(t, IsValidProbeType(ProbeSandboxOrderCreate))
	assert.False(t, IsValidProbeType("nope"))
	assert.True(t, IsValidValidationEnv(ValidationEnvProduction))
	assert.False(t, IsValidValidationEnv("staging"))
	assert.True(t, IsValidResultStatus(ResultStatusError))
	assert.False(t, IsValidResultStatus("meh"))
	assert.Len(t, HashPayload([]byte("hello")), 64)
	assert.Equal(t, HashPayload([]byte("x")), HashPayload([]byte("x")))
	assert.NotEqual(t, HashPayload([]byte("x")), HashPayload([]byte("y")))
}
