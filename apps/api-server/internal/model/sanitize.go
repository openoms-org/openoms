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

var listingHTMLPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("h1", "h2", "p", "ul", "ol", "li")
	return p
}()

const maxListingHTMLLength = 50000

// SanitizeListingHTML sanitizes HTML for marketplace listing descriptions.
// Only allows structural tags: h1, h2, p, ul, ol, li. Strips all attributes.
func SanitizeListingHTML(s string) string {
	if len(s) > maxListingHTMLLength {
		s = s[:maxListingHTMLLength]
	}
	return listingHTMLPolicy.Sanitize(s)
}
