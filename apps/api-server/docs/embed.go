// Package docs embeds the OpenAPI specification for the OpenOMS API server.
package docs

import _ "embed"

// OpenAPISpec contains the raw bytes of the OpenAPI YAML specification file.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
