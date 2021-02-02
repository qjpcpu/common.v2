package mfs

import (
	"bytes"
	"io"
)

type bytesReadWriteCloser struct {
	*bytes.Reader
}

func NewBuffer(b []byte) *bytesReadWriteCloser {
	return &bytesReadWriteCloser{Reader: bytes.NewReader(b)}
}

func (b *bytesReadWriteCloser) Write(p []byte) (int, error) {
	b.Reader = bytes.NewReader(p)
	return len(p), nil
}

func (b *bytesReadWriteCloser) Close() error {
	b.Reader.Seek(0, io.SeekStart)
	return nil
}
