package model

import "strings"

// StripHTMLTags removes HTML tags from input to prevent stored XSS.
// This is a simple approach -- strips anything between < and >.
func StripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
