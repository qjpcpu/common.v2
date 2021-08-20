package static

import (
	_ "unsafe"

	_ "github.com/qjpcpu/common.v2/static/mockgen"
	"github.com/qjpcpu/common.v2/static/mockgen/model"
)

//go:linkname sourceMode github.com/qjpcpu/common.v2/static/mockgen.sourceMode
func sourceMode(source string) (*model.Package, error)

//go:linkname reflectMode github.com/qjpcpu/common.v2/static/mockgen.reflectMode
func reflectMode(importPath string, symbols []string) (*model.Package, error)

func ParseGoFile(sourcefile string) (*model.Package, error) {
	return sourceMode(sourcefile)
}

func ParsePackage(pkg string, symbols []string) (*model.Package, error) {
	return reflectMode(pkg, symbols)
}
