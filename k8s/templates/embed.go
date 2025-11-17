package templates

import "embed"

// Templates embeds all our official, build-in templates.
//
//go:embed *
var Templates embed.FS
