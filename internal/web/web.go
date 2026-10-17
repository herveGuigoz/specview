package web

import (
	"embed"
	"io/fs"
)

// Build the frontend first: npm --prefix ui run build.
//
//go:embed dist
var embedded embed.FS

func Assets() fs.FS {
	assets, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return assets
}
