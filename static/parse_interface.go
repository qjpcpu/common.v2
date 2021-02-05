package static

import (
	"github.com/qjpcpu/common.v2/static/mockgen"
	"github.com/qjpcpu/common.v2/static/mockgen/model"
)

func ParseGoFile(sourcefile string) (*model.Package, error) {
	return mockgen.ParseGoFile(sourcefile)
}

func ParsePackage(pkg string, symbols []string) (*model.Package, error) {
	return mockgen.ParsePackage(pkg, symbols)
}
