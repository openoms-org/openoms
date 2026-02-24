package model

import "github.com/microcosm-cc/bluemonday"

// stripPolicy is initialized once and reused (thread-safe).
var stripPolicy = bluemonday.StrictPolicy()

// StripHTMLTags removes all HTML tags and attributes using bluemonday.
// Handles HTML entities, nested tags, SVG vectors, and other bypass techniques
// that a naive character-by-character parser would miss.
func StripHTMLTags(s string) string {
	return stripPolicy.Sanitize(s)
}
