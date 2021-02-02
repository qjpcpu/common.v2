package http

import (
	"errors"
	"net"
	nethttp "net/http"
	"strconv"
	"time"
)

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// ServerOnAnyPort handle
type ServerOnAnyPort struct {
	server *nethttp.Server
	addr   string
	fn     func() error
}

// Addr return :port
func (sp *ServerOnAnyPort) Addr() string {
	return sp.addr
}

func (sp *ServerOnAnyPort) Close() error {
	return sp.server.Close()
}

// Serve http service
func (sp *ServerOnAnyPort) Serve() error {
	if sp.fn == nil {
		return errors.New("serve nothing")
	}
	return sp.fn()
}

// ListenOnAnyPort serve http on any port
func ListenOnAnyPort(h nethttp.Handler) *ServerOnAnyPort {
	sp := &ServerOnAnyPort{}
	server := &nethttp.Server{Handler: h}
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		sp.fn = func() error {
			return err
		}
		return sp
	}
	port := ln.Addr().(*net.TCPAddr).Port
	server.Addr = ":" + strconv.Itoa(port)

	sp.addr = server.Addr
	sp.fn = func() error {
		return server.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
	}
	sp.server = server
	return sp
}
