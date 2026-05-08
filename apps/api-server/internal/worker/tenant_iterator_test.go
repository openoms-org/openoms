package worker

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// checkWorkerContext
// ---------------------------------------------------------------------------

func TestCheckWorkerContext_NotCancelled(t *testing.T) {
	assert.NoError(t, checkWorkerContext(context.Background()))
}

func TestCheckWorkerContext_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := checkWorkerContext(ctx)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestCheckWorkerContext_DeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	err := checkWorkerContext(ctx)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// ---------------------------------------------------------------------------
// truncateErrorMessage
// ---------------------------------------------------------------------------

func TestTruncateErrorMessage_ShortMessage(t *testing.T) {
	msg := "something went wrong"
	assert.Equal(t, msg, truncateErrorMessage(msg))
}

func TestTruncateErrorMessage_EmptyString(t *testing.T) {
	assert.Equal(t, "", truncateErrorMessage(""))
}

func TestTruncateErrorMessage_ExactBoundary(t *testing.T) {
	msg := strings.Repeat("a", maxErrorMessageLen)
	result := truncateErrorMessage(msg)
	assert.Equal(t, maxErrorMessageLen, len(result))
	assert.Equal(t, msg, result)
}

func TestTruncateErrorMessage_OneBeyondBoundary(t *testing.T) {
	msg := strings.Repeat("b", maxErrorMessageLen+1)
	result := truncateErrorMessage(msg)
	assert.Equal(t, maxErrorMessageLen, len(result))
}

func TestTruncateErrorMessage_LongASCII(t *testing.T) {
	msg := strings.Repeat("x", 1000)
	result := truncateErrorMessage(msg)
	assert.Equal(t, maxErrorMessageLen, len(result))
	assert.Equal(t, strings.Repeat("x", maxErrorMessageLen), result)
}

func TestTruncateErrorMessage_TwoByteCharStraddlesBoundary(t *testing.T) {
	// 2-byte UTF-8 char: Polish a-ogonek U+0105 = 0xC4 0x85
	prefix := strings.Repeat("a", maxErrorMessageLen-1) // 499 ASCII bytes
	msg := prefix + "\xC4\x85"                          // total = 501 bytes

	result := truncateErrorMessage(msg)
	// Truncating at 500 splits the 2-byte char, so the function walks back to 499.
	assert.Equal(t, maxErrorMessageLen-1, len(result))
	assert.Equal(t, prefix, result)
	assert.True(t, utf8.ValidString(result))
}

func TestTruncateErrorMessage_ThreeByteCharStraddlesBoundary(t *testing.T) {
	// 3-byte UTF-8 char: U+4E16 = 0xE4 0xB8 0x96
	prefix := strings.Repeat("a", maxErrorMessageLen-2) // 498 bytes
	msg := prefix + "\xE4\xB8\x96"                      // total = 501 bytes

	result := truncateErrorMessage(msg)
	// Truncating at 500 includes only 2 of 3 bytes -> invalid. Walk back to 498.
	assert.Equal(t, maxErrorMessageLen-2, len(result))
	assert.Equal(t, prefix, result)
	assert.True(t, utf8.ValidString(result))
}

func TestTruncateErrorMessage_FourByteCharStraddlesBoundary(t *testing.T) {
	// 4-byte UTF-8 char: U+1F600 = 0xF0 0x9F 0x98 0x80
	prefix := strings.Repeat("a", maxErrorMessageLen-3) // 497 bytes
	msg := prefix + "\xF0\x9F\x98\x80"                  // total = 501 bytes

	result := truncateErrorMessage(msg)
	// Truncating at 500 includes only 3 of 4 bytes -> invalid. Walk back to 497.
	assert.Equal(t, maxErrorMessageLen-3, len(result))
	assert.Equal(t, prefix, result)
	assert.True(t, utf8.ValidString(result))
}

func TestTruncateErrorMessage_TwoByteCharFitsExactly(t *testing.T) {
	// 2-byte char ending exactly at position 500 should NOT be truncated.
	prefix := strings.Repeat("a", maxErrorMessageLen-2) // 498 bytes
	msg := prefix + "\xC4\x85"                          // total = 500 bytes

	result := truncateErrorMessage(msg)
	assert.Equal(t, maxErrorMessageLen, len(result))
	assert.Equal(t, msg, result)
	assert.True(t, utf8.ValidString(result))
}

func TestTruncateErrorMessage_AllTwoByteChars(t *testing.T) {
	// 250 x 2-byte chars = 500 bytes, exactly at boundary
	msg := strings.Repeat("\xC4\x85", 250)
	require.Equal(t, 500, len(msg))

	result := truncateErrorMessage(msg)
	assert.Equal(t, maxErrorMessageLen, len(result))
	assert.Equal(t, msg, result)
}

func TestTruncateErrorMessage_AllTwoByteCharsOverBoundary(t *testing.T) {
	// 251 x 2-byte chars = 502 bytes, over boundary
	msg := strings.Repeat("\xC4\x85", 251)
	require.Equal(t, 502, len(msg))

	result := truncateErrorMessage(msg)
	// Truncating at 500 lands cleanly between two 2-byte chars.
	assert.Equal(t, maxErrorMessageLen, len(result))
	assert.True(t, utf8.ValidString(result))
}

func TestTruncateErrorMessage_MixedMultiBytePreservesUTF8(t *testing.T) {
	// Mixed ASCII and multi-byte; result must always be valid UTF-8.
	prefix := strings.Repeat("hello \xC4\x85 ", 100) // well over 500 bytes
	result := truncateErrorMessage(prefix)
	assert.LessOrEqual(t, len(result), maxErrorMessageLen)
	assert.True(t, utf8.ValidString(result), "truncated result must be valid UTF-8")
}

func TestTruncateErrorMessage_SingleCharOverBoundary(t *testing.T) {
	// Single 4-byte char at position 0 that makes the string exactly 4 bytes,
	// which is under the limit. Should pass through unchanged.
	msg := "\xF0\x9F\x98\x80" // 4 bytes
	result := truncateErrorMessage(msg)
	assert.Equal(t, msg, result)
}

// ---------------------------------------------------------------------------
// closeProvider
// ---------------------------------------------------------------------------

func TestCloseProvider_ImplementsCloser(t *testing.T) {
	closed := false
	p := &testProviderWithClose{onClose: func() { closed = true }}
	closeProvider(p)
	assert.True(t, closed, "Close() should have been called")
}

func TestCloseProvider_DoesNotImplementCloser_String(_ *testing.T) {
	// Should not panic when passed a type without Close().
	closeProvider("just a string")
}

func TestCloseProvider_DoesNotImplementCloser_Int(_ *testing.T) {
	closeProvider(42)
}

func TestCloseProvider_DoesNotImplementCloser_Nil(_ *testing.T) {
	closeProvider(nil)
}

func TestCloseProvider_DoesNotImplementCloser_Struct(_ *testing.T) {
	closeProvider(struct{}{})
}

func TestCloseProvider_CalledMultipleTimes(t *testing.T) {
	callCount := 0
	p := &testProviderWithClose{onClose: func() { callCount++ }}
	closeProvider(p)
	closeProvider(p)
	assert.Equal(t, 2, callCount, "Close() should be called each time closeProvider is called")
}

// ---------------------------------------------------------------------------
// TenantIntegration struct
// ---------------------------------------------------------------------------

func TestTenantIntegration_SyncCursorNilable(t *testing.T) {
	ti := TenantIntegration{}
	assert.Nil(t, ti.SyncCursor, "SyncCursor should be nil by default")

	cursor := "2024-01-01T00:00:00Z"
	ti.SyncCursor = &cursor
	assert.Equal(t, "2024-01-01T00:00:00Z", *ti.SyncCursor)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type testProviderWithClose struct {
	onClose func()
}

func (p *testProviderWithClose) Close() {
	if p.onClose != nil {
		p.onClose()
	}
}
