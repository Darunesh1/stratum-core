package docs

import "embed"

//go:embed agents/* methodology/* workflows/* api/*
var DocsFS embed.FS
