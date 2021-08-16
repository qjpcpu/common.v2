package http

import (
	"context"
	"io"
	"net/http"
	"time"
)

type Client interface {
	EnableCookie() Client
	SetTimeout(tm time.Duration) Client
	SetMock(fn Endpoint) Client
	SetDebug(w HTTPLogger) Client
	SetRetry(opt RetryOption) Client
	SetHeader(name, val string) Client
	SetHeaders(hder map[string]string) Client
	MakeDoer(opts ...Option) Doer
	AddMiddleware(m ...Middleware) Client
	AddBeforeHook(hook func(*http.Request)) Client
	AddAfterHook(hook func(*http.Response)) Client
	DoRequest(req *http.Request, opts ...Option) *Response
	Do(ctx context.Context, method string, uri string, body io.Reader, opts ...Option) *Response
	Download(ctx context.Context, uri string, w io.Writer, opts ...Option) error
	Get(ctx context.Context, uri string, opts ...Option) *Response
	Post(ctx context.Context, urlstr string, data []byte, opts ...Option) *Response
	Put(ctx context.Context, urlstr string, data []byte, opts ...Option) *Response
	PostForm(ctx context.Context, urlstr string, data map[string]interface{}, opts ...Option) *Response
	PostJSON(ctx context.Context, urlstr string, data interface{}, opts ...Option) *Response
}
