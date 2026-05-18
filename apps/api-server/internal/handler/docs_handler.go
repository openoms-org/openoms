package handler

import (
	"net/http"
)

// DocsHandler serves the OpenAPI spec and Swagger UI.
type DocsHandler struct {
	spec []byte
}

// NewDocsHandler creates a new DocsHandler with the given embedded spec bytes.
func NewDocsHandler(spec []byte) *DocsHandler {
	return &DocsHandler{spec: spec}
}

// ServeSpec serves the OpenAPI spec file at /v1/openapi.yaml.
func (h *DocsHandler) ServeSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Security-Policy", openAPISpecContentSecurityPolicy)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.spec)
}

// ServeSwaggerUI serves a minimal HTML page that loads Swagger UI from a CDN.
func (h *DocsHandler) ServeSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Security-Policy", swaggerUIContentSecurityPolicy)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerHTML))
}

const openAPISpecContentSecurityPolicy = "default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; object-src 'none'"

const swaggerUIContentSecurityPolicy = "default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; object-src 'none'; img-src 'self' data:; font-src https://unpkg.com data:; style-src https://unpkg.com 'unsafe-inline'; script-src https://unpkg.com 'unsafe-inline'; connect-src 'self'"

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>OpenOMS API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.18.2/swagger-ui.css" crossorigin="anonymous">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.18.2/swagger-ui-bundle.js" crossorigin="anonymous"></script>
  <script>
    SwaggerUIBundle({
      url: "/v1/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
      ],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`
