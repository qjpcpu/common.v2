package core

//go:generate mockgen -package core -self_package github.com/qjpcpu/common.v2/static/mockgen/internal/tests/self_package -destination mock.go github.com/qjpcpu/common.v2/static/mockgen/internal/tests/self_package Methods

type Info struct{}

type Methods interface {
	getInfo() Info
}
