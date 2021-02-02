package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type MockServer struct {
	mux       *http.ServeMux
	server    *ServerOnAnyPort
	URLPrefix string
}

func NewMockServer() *MockServer {
	mux := http.NewServeMux()
	mux.HandleFunc("/echo", Echo)
	return &MockServer{mux: mux}
}

func (ms *MockServer) Handle(path string, fn func(w http.ResponseWriter, req *http.Request)) *MockServer {
	ms.mux.HandleFunc(path, fn)
	return ms
}

func (ms *MockServer) ServeBackground() func() {
	ms.server = ListenOnAnyPort(ms.mux)
	go ms.server.Serve()
	ms.URLPrefix = "http://127.0.0.1" + ms.server.Addr()
	return func() {
		ms.server.Close()
	}
}

func Echo(w http.ResponseWriter, req *http.Request) {
	args := make(map[string]string)
	qs := req.URL.Query()
	for k := range qs {
		args[k] = qs.Get(k)
	}

	header := make(map[string]string)
	for k := range req.Header {
		header[k] = req.Header.Get(k)
	}

	body, _ := ioutil.ReadAll(req.Body)

	output, _ := json.Marshal(map[string]interface{}{
		"args":    args,
		"headers": header,
		"body":    string(body),
		"url":     req.URL.String(),
	})
	w.Write(output)
}
