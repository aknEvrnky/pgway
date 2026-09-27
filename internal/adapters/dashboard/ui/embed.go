//go:build embeddashboard

package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distEmbed embed.FS

// FS is the generated Nuxt static site (contents of dist/).
var FS fs.FS

func init() {
	sub, err := fs.Sub(distEmbed, "dist")
	if err != nil {
		panic("dashboard ui embed: " + err.Error())
	}
	FS = sub
}

// Enabled reports whether the dashboard UI was embedded at build time.
func Enabled() bool { return true }
