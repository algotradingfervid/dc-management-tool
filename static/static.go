package static

import "embed"

// FS holds embedded CSS and JS static assets compiled into the binary.
// The uploads/ directory is intentionally excluded and served from disk.
//
//go:embed css js
var FS embed.FS
