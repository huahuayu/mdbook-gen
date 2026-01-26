package templates

import "embed"

//go:embed main.css book.yaml sample/*.md
var Assets embed.FS
