package http

import (
	"bytes"
	"io"
	syshttp "net/http"

	"encoding/json"
)

type Request struct {
	*syshttp.Request
}

func FromRequest(req *syshttp.Request) *Request {
	return &Request{Request: req}
}

func (req *Request) AddRetryHook(hook RetryHook) {
	getValue(req.Request).AddRetryHook(hook)
}

type Response struct {
	*syshttp.Response
	Err error
}

func (r *Response) Result() (*syshttp.Response, error) {
	return r.Response, r.Err
}

func (r *Response) Unmarshal(obj interface{}) error {
	data, err := r.GetBody()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

func (r *Response) MustGetBody() []byte {
	data, err := r.GetBody()
	if err != nil {
		panic(err)
	}
	return data
}

func (r *Response) GetBody() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := r.Save(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *Response) Save(w io.Writer) error {
	if r.Err != nil {
		return r.Err
	}
	if r.Response == nil || r.Response.Body == nil {
		return nil
	}
	defer r.Response.Body.Close()
	_, err := io.Copy(w, r.Response.Body)
	return err
}

func buildResponse(res *syshttp.Response, err error) *Response {
	if res == nil {
		res = &syshttp.Response{}
	}
	return &Response{Response: res, Err: err}
}
