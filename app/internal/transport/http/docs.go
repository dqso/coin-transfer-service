package http

import (
	_ "embed"
	"net/http"
)

//go:embed docs.html
var docsPage []byte

// Docs shows the Swagger UI page. The page loads Swagger UI from a CDN and reads the spec from /openapi.yaml.
func (h *Handler) Docs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(docsPage)
}
