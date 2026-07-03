// Package web embeds the static assets for the translation UI so the binary is
// fully self-contained (no external files at runtime).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:assets
var files embed.FS

// FS returns the embedded asset filesystem rooted at the asset directory.
func FS() fs.FS {
	sub, err := fs.Sub(files, "assets")
	if err != nil {
		panic(err) // embedded path is fixed; a failure is a build-time bug
	}
	return sub
}
