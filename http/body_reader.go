package http

import (
	"bytes"
	"io"
	"io/ioutil"
	syshttp "net/http"
)

type repeatableReader struct {
	*bytes.Reader
}

func (rr *repeatableReader) SeekStart() error {
	_, err := rr.Seek(0, io.SeekStart)
	return err
}

func (rr *repeatableReader) Close() error {
	return rr.SeekStart()
}

func RepeatableReadResponse(res *syshttp.Response) ([]byte, error) {
	if res == nil || res.Body == nil {
		return nil, nil
	}
	if rr, ok := res.Body.(*repeatableReader); ok {
		defer rr.Close()
		return ioutil.ReadAll(res.Body)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		res.Body.Close()
		return nil, err
	}
	res.Body.Close()
	res.Body = &repeatableReader{Reader: bytes.NewReader(data)}
	return data, nil
}

func RepeatableReadRequest(res *syshttp.Request) ([]byte, error) {
	if res.Body == nil {
		return nil, nil
	}
	if rr, ok := res.Body.(*repeatableReader); ok {
		defer rr.Close()
		return ioutil.ReadAll(res.Body)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		res.Body.Close()
		return nil, err
	}
	res.Body.Close()
	res.Body = &repeatableReader{Reader: bytes.NewReader(data)}
	return data, nil
}
