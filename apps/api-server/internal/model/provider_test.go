package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanTransition_LegalEdges(t *testing.T) {
	legal := [][2]string{
		{ProviderStateResearch, ProviderStateDesigned},
		{ProviderStateDesigned, ProviderStateAdapterInProgress},
		{ProviderStateAdapterInProgress, ProviderStateInternalValidation},
		{ProviderStateInternalValidation, ProviderStatePrivateBeta},
		{ProviderStateInternalValidation, ProviderStateDesigned}, // gaps found
		{ProviderStatePrivateBeta, ProviderStateAvailable},
		{ProviderStatePrivateBeta, ProviderStateInternalValidation}, // beta issue
		{ProviderStateAvailable, ProviderStateDeprecated},
		{ProviderStateAvailable, ProviderStateInternalValidation}, // regression
		{ProviderStateDeprecated, ProviderStateRetired},
	}
	for _, e := range legal {
		assert.Truef(t, CanTransition(e[0], e[1]), "expected %s -> %s to be legal", e[0], e[1])
	}
}

func TestCanTransition_IllegalEdges(t *testing.T) {
	illegal := [][2]string{
		{ProviderStateResearch, ProviderStateAvailable}, // skip the pipeline
		{ProviderStateResearch, ProviderStateInternalValidation},
		{ProviderStateRetired, ProviderStateAvailable}, // terminal
		{ProviderStateRetired, ProviderStateDeprecated},
		{ProviderStateAvailable, ProviderStateResearch},   // backward skip
		{ProviderStateDeprecated, ProviderStateAvailable}, // no un-deprecate
		{ProviderStateDesigned, ProviderStateAvailable},
		{ProviderStateAvailable, ProviderStateAvailable}, // no self-loop
	}
	for _, e := range illegal {
		assert.Falsef(t, CanTransition(e[0], e[1]), "expected %s -> %s to be illegal", e[0], e[1])
	}
}

func TestIsPublishedState(t *testing.T) {
	for _, s := range []string{ProviderStatePrivateBeta, ProviderStateAvailable, ProviderStateDeprecated, ProviderStateRetired} {
		assert.Truef(t, IsPublishedState(s), "%s should be a published/frozen state", s)
	}
	for _, s := range []string{ProviderStateResearch, ProviderStateDesigned, ProviderStateAdapterInProgress, ProviderStateInternalValidation} {
		assert.Falsef(t, IsPublishedState(s), "%s should be a draft (editable) state", s)
	}
}

func TestValidators(t *testing.T) {
	assert.True(t, IsValidProviderType(ProviderTypeSupplier))
	assert.False(t, IsValidProviderType("banana"))
	assert.True(t, IsValidPublicationState(ProviderStateAvailable))
	assert.False(t, IsValidPublicationState("shipped"))
}
