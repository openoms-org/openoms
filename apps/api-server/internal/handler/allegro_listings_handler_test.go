package handler

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeForAllegro_BTPFeedHTML(t *testing.T) {
	// BTP-style HTML with div, img, h3, b, hr, style attributes
	input := `<div style="text-align: center; margin-bottom: 20px"><img src="https://pub.btp.pro/pic.jpg" alt="Product" style="max-width: 100%"></div><hr><h3>Product Name – Description</h3><hr><p><b>Product Name</b> is a great product.</p>`

	result := sanitizeForAllegro(input)

	// img should be removed entirely (not leaked as text)
	assert.NotContains(t, result, "pub.btp.pro")
	assert.NotContains(t, result, "pic.jpg")

	// h3 should be converted to h2
	assert.Contains(t, result, "<h2>")
	assert.Contains(t, result, "Product Name")
	assert.NotContains(t, result, "<h3>")

	// b tags stripped, content preserved
	assert.NotContains(t, result, "<b>")
	assert.Contains(t, result, "is a great product")

	// div tags stripped
	assert.NotContains(t, result, "<div>")

	// No bare text outside block elements
	assertValidAllegroHTML(t, result)
}

func TestSanitizeForAllegro_StyleBlock(t *testing.T) {
	input := `<style>.foo { color: red; }</style><p>Hello</p>`
	result := sanitizeForAllegro(input)

	assert.NotContains(t, result, "color")
	assert.NotContains(t, result, ".foo")
	assert.Contains(t, result, "<p>Hello</p>")
}

func TestSanitizeForAllegro_HTMLComments(t *testing.T) {
	input := `<!-- comment --><p>Hello</p>`
	result := sanitizeForAllegro(input)

	assert.NotContains(t, result, "comment")
	assert.Contains(t, result, "<p>Hello</p>")
}

func TestSanitizeForAllegro_NbspReplaced(t *testing.T) {
	input := `<p>Hello&nbsp;World</p>`
	result := sanitizeForAllegro(input)

	assert.NotContains(t, result, "&nbsp;")
	assert.Contains(t, result, "Hello World")
}

func TestSanitizeForAllegro_AmpersandEscaped(t *testing.T) {
	input := `<p>Tom &amp; Jerry &amp; friends</p>`
	result := sanitizeForAllegro(input)

	// Already escaped & should not be double-escaped
	assert.NotContains(t, result, "&amp;amp;")
	assert.Contains(t, result, "Tom &amp; Jerry")
}

func TestSanitizeForAllegro_BareAmpersandEscaped(t *testing.T) {
	input := `<p>Tom & Jerry</p>`
	result := sanitizeForAllegro(input)

	assert.Contains(t, result, "Tom &amp; Jerry")
}

func TestSanitizeForAllegro_PreservesLists(t *testing.T) {
	input := `<ul><li>Item 1</li><li>Item 2</li></ul>`
	result := sanitizeForAllegro(input)

	assert.Contains(t, result, "<ul>")
	assert.Contains(t, result, "<li>Item 1</li>")
	assert.Contains(t, result, "<li>Item 2</li>")
}

func TestSanitizeForAllegro_EmptyInput(t *testing.T) {
	assert.Equal(t, "<p></p>", sanitizeForAllegro(""))
}

func TestSanitizeForAllegro_PlainText(t *testing.T) {
	result := sanitizeForAllegro("Just plain text")
	assert.Equal(t, "<p>Just plain text</p>", result)
}

func TestSanitizeForAllegro_BareTextBetweenBlocks(t *testing.T) {
	input := `<p>First</p>Some bare text<p>Second</p>`
	result := sanitizeForAllegro(input)

	assert.Contains(t, result, "<p>First</p>")
	assert.Contains(t, result, "<p>Some bare text</p>")
	assert.Contains(t, result, "<p>Second</p>")
	assertValidAllegroHTML(t, result)
}

func TestSanitizeForAllegro_TableStripped(t *testing.T) {
	input := `<table><tr><td>Cell 1</td><td>Cell 2</td></tr></table>`
	result := sanitizeForAllegro(input)

	assert.NotContains(t, result, "<table>")
	assert.NotContains(t, result, "<tr>")
	assert.NotContains(t, result, "<td>")
	assert.Contains(t, result, "Cell 1")
	assert.Contains(t, result, "Cell 2")
}

// assertValidAllegroHTML checks that all text content is inside allowed block elements.
func assertValidAllegroHTML(t *testing.T, html string) {
	t.Helper()

	// Remove all block elements with their content (innermost first, repeat for nesting)
	remaining := html
	for range 10 { // max nesting depth
		prev := remaining
		for _, tag := range []string{"li", "p", "h1", "h2"} {
			re := regexp.MustCompile(`<` + tag + `>` + `[^<]*` + `</` + tag + `>`)
			remaining = re.ReplaceAllString(remaining, "")
		}
		// Also handle list containers (may contain <li> tags already removed)
		for _, tag := range []string{"ul", "ol"} {
			re := regexp.MustCompile(`<` + tag + `>\s*</` + tag + `>`)
			remaining = re.ReplaceAllString(remaining, "")
		}
		if remaining == prev {
			break
		}
	}

	trimmed := strings.TrimSpace(remaining)
	if trimmed != "" {
		t.Errorf("found bare text outside block elements: %q\nfull html: %s", trimmed, html)
	}
}
