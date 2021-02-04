//go:generate mockgen -package core -self_package github.com/qjpcpu/common.v2/static/mockgen/internal/tests/self_package -destination mock.go github.com/qjpcpu/common.v2/static/mockgen/internal/tests/self_package Methods
package core

type Info struct {
	name string
}

type Methods interface {
	getInfo() Info
}
