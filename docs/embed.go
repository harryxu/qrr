// Package docs embeds the recipe contract for standalone CLI help.
package docs

import _ "embed"

// Schema contains the same recipe documentation shipped in docs/schema.md.
//
//go:embed schema.md
var Schema string
