//go:generate go run assets_generate.go

package runtime

import "github.com/malivvan/rumo/std/cui/edit"

var Files = edit.NewRuntimeFiles(files)
