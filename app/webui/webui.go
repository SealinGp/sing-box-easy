// Package webui embeds the built frontend (Vue dist) into the binary so a
// single executable can serve the full application — required for the OpenWrt
// ipk package and convenient everywhere else.
//
// The real dist is copied into ./dist by the release workflow before the Go
// build. The committed placeholder index.html only ships in local dev builds,
// where the on-disk ./dist (vite output) takes precedence anyway — see
// routes.Route.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// FS returns the embedded frontend rooted at the dist directory itself
// (index.html at the top level).
func FS() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		// Cannot happen: "dist" is a compile-time embedded directory.
		panic(err)
	}
	return sub
}
