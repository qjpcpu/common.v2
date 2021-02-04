// +build go1.14

package overlap

//go:generate mockgen -package overlap -destination mock.go -source overlap.go -aux_files github.com/qjpcpu/common.v2/static/mockgen/internal/tests/overlapping_methods=interfaces.go
type ReadWriteCloser interface {
	ReadCloser
	WriteCloser
}
