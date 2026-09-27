//go:build !embeddashboard

package ui

import "io/fs"

// FS is empty when the binary was built without -tags embeddashboard.
var FS fs.FS = emptyFS{}

type emptyFS struct{}

func (emptyFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

// Enabled reports whether the dashboard UI was embedded at build time.
func Enabled() bool { return false }
