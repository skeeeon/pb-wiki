package main

import (
	"embed"
	"io/fs"
)

// frontendEmbed is the compiled Vue app bundled into the Go binary at build
// time. The `//go:embed` directive must live at the same level as the embedded
// directory (Go disallows `..` in embed paths), which is why this file is at
// the project root rather than under internal/static.
//
//go:embed all:frontend/dist
var frontendEmbed embed.FS

// frontendDist returns the embedded filesystem re-rooted at frontend/dist so
// callers don't need to know about the leading prefix.
func frontendDist() fs.FS {
	sub, err := fs.Sub(frontendEmbed, "frontend/dist")
	if err != nil {
		// Unreachable in practice — frontend/dist is guaranteed to exist at
		// compile time (the //go:embed directive would have failed otherwise).
		panic(err)
	}
	return sub
}
