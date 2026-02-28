package model

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeListingHTML_AllowsValidTags(t *testing.T) {
	input := `<h1>Title</h1><h2>Section</h2><p>Text</p><ul><li>Item</li></ul><ol><li>Num</li></ol>`
	result := SanitizeListingHTML(input)
	if result != input {
		t.Errorf("expected valid tags preserved, got %q", result)
	}
}

func TestSanitizeListingHTML_StripsDisallowedTags(t *testing.T) {
	input := `<p>Hello <b>bold</b> <script>alert(1)</script> <img src="x"/> world</p>`
	result := SanitizeListingHTML(input)
	// bluemonday strips script content entirely, img is removed, b content preserved
	if result == input {
		t.Error("expected disallowed tags stripped")
	}
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "bold") || !strings.Contains(result, "world") {
		t.Errorf("expected text content preserved, got %q", result)
	}
	if strings.Contains(result, "<script>") || strings.Contains(result, "<b>") || strings.Contains(result, "<img") {
		t.Errorf("expected disallowed tags removed, got %q", result)
	}
}

func TestSanitizeListingHTML_StripsAttributes(t *testing.T) {
	input := `<p class="foo" style="color:red" onclick="alert(1)">Text</p>`
	result := SanitizeListingHTML(input)
	expected := `<p>Text</p>`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSanitizeListingHTML_EmptyInput(t *testing.T) {
	result := SanitizeListingHTML("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSanitizeListingHTML_TruncatesAtRuneBoundary(t *testing.T) {
	// Build a string over the byte limit using multi-byte Polish characters.
	// "ą" is 2 bytes in UTF-8. Naive byte truncation would split a character.
	base := strings.Repeat("ą", maxListingHTMLLength) // 2x bytes
	result := SanitizeListingHTML(base)
	if !utf8.ValidString(result) {
		t.Error("expected valid UTF-8 after truncation")
	}
	if len(result) > maxListingHTMLLength {
		t.Errorf("expected result <= %d bytes, got %d", maxListingHTMLLength, len(result))
	}
}
