package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// mapAllegroStatus — all known mappings
// ---------------------------------------------------------------------------

func TestMapAllegroStatus_ReadyForProcessing(t *testing.T) {
	assert.Equal(t, "new", mapAllegroStatus("READY_FOR_PROCESSING"))
}

func TestMapAllegroStatus_Processing(t *testing.T) {
	assert.Equal(t, "processing", mapAllegroStatus("PROCESSING"))
}

func TestMapAllegroStatus_Sent(t *testing.T) {
	assert.Equal(t, "shipped", mapAllegroStatus("SENT"))
}

func TestMapAllegroStatus_Delivered(t *testing.T) {
	assert.Equal(t, "delivered", mapAllegroStatus("DELIVERED"))
}

func TestMapAllegroStatus_Cancelled(t *testing.T) {
	assert.Equal(t, "cancelled", mapAllegroStatus("CANCELLED"))
}

func TestMapAllegroStatus_Returned(t *testing.T) {
	assert.Equal(t, "returned", mapAllegroStatus("RETURNED"))
}

// ---------------------------------------------------------------------------
// mapAllegroStatus — unknown / empty
// ---------------------------------------------------------------------------

func TestMapAllegroStatus_UnknownStatus(t *testing.T) {
	assert.Equal(t, "", mapAllegroStatus("UNKNOWN_STATUS"))
}

func TestMapAllegroStatus_EmptyString(t *testing.T) {
	assert.Equal(t, "", mapAllegroStatus(""))
}

func TestMapAllegroStatus_LowercaseNotMatched(t *testing.T) {
	// mapAllegroStatus uses exact match (uppercase), lowercase should not match.
	assert.Equal(t, "", mapAllegroStatus("sent"))
}

func TestMapAllegroStatus_MixedCaseNotMatched(t *testing.T) {
	assert.Equal(t, "", mapAllegroStatus("Ready_For_Processing"))
}

// ---------------------------------------------------------------------------
// mapAllegroStatus — table-driven comprehensive test
// ---------------------------------------------------------------------------

func TestMapAllegroStatus_AllMappings(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"READY_FOR_PROCESSING", "new"},
		{"PROCESSING", "processing"},
		{"SENT", "shipped"},
		{"DELIVERED", "delivered"},
		{"CANCELLED", "cancelled"},
		{"RETURNED", "returned"},
		{"", ""},
		{"PENDING", ""},
		{"COMPLETED", ""},
		{"REFUNDED", ""},
	}

	for _, tt := range tests {
		t.Run("allegro_status_"+tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapAllegroStatus(tt.input))
		})
	}
}
