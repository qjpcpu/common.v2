package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	syshttp "net/http"
	"net/http/cookiejar"
	"net/url"
	"runtime"
	"time"

	"encoding/json"
)

// NewClient new client
func NewClient() Client {
	cli := &clientImpl{
		Client: &syshttp.Client{Transport: DefaultPooledTransport()},
	}
	cli.SetTimeout(5 * time.Second)
	return cli
}

// clientImpl client
type clientImpl struct {
	Client      *syshttp.Client
	middlewares []Middleware
}

// EnableCookie use cookie
func (client *clientImpl) EnableCookie() Client {
	jar, _ := cookiejar.New(nil)
	client.Client.Jar = jar
	return client
}

// SetTimeout timeout
func (client *clientImpl) SetTimeout(tm time.Duration) Client {
	client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			getValue(req).Timeout = tm
			return next(req)
		}
	})
	return client
}

func (client *clientImpl) SetMock(fn Endpoint) Client {
	client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			getValue(req).Mock = fn
			return next(req)
		}
	})
	return client
}

func (client *clientImpl) SetDebug(w HTTPLogger) Client {
	client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			getValue(req).Debugger = w
			return next(req)
		}
	})
	return client
}

func (client *clientImpl) SetRetry(opt RetryOption) Client {
	client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			getValue(req).RetryOption = &opt
			return next(req)
		}
	})
	return client
}

func (client *clientImpl) SetHeader(name, val string) Client {
	return client.SetHeaders(map[string]string{name: val})
}

func (client *clientImpl) SetHeaders(hder map[string]string) Client {
	return client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			setRequestHeader(req, hder)
			return next(req)
		}
	})
}

func (client *clientImpl) AddMiddleware(m ...Middleware) Client {
	client.middlewares = append(client.middlewares, m...)
	return client
}

func (client *clientImpl) AddBeforeHook(hook func(*syshttp.Request)) Client {
	return client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			hook(req)
			return next(req)
		}
	})
}

func (client *clientImpl) AddAfterHook(hook func(*syshttp.Response)) Client {
	return client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			res, err := next(req)
			if err == nil && res != nil {
				hook(res)
			}
			return res, err
		}
	})
}

func (client *clientImpl) MakeDoer(opts ...Option) Doer {
	return (Doer)(client.makeFinalHandler(client.getOptionMiddlewares(opts...)...))
}

func (client *clientImpl) DoRequest(req *syshttp.Request, opts ...Option) *Response {
	res, err := client.makeFinalHandler(client.getOptionMiddlewares(opts...)...)(req)
	return buildResponse(res, err)
}

func (client *clientImpl) Do(ctx context.Context, method string, uri string, body io.Reader, opts ...Option) *Response {
	req, err := syshttp.NewRequest(method, uri, body)
	if err != nil {
		return buildResponse(nil, err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	res, err := client.makeFinalHandler(client.getOptionMiddlewares(opts...)...)(req)
	return buildResponse(res, err)
}

func (client *clientImpl) Download(ctx context.Context, uri string, w io.Writer, opts ...Option) error {
	opts = append(opts, WithBody(w))
	return client.Do(ctx, "GET", uri, nil, opts...).Err
}

// Get get url
func (client *clientImpl) Get(ctx context.Context, uri string, opts ...Option) *Response {
	return client.Do(ctx, "GET", uri, nil, opts...)
}

// Post data
func (client *clientImpl) Post(ctx context.Context, urlstr string, data []byte, opts ...Option) *Response {
	return client.Do(ctx, "POST", urlstr, bytes.NewBuffer(data), opts...)
}

func (client *clientImpl) Put(ctx context.Context, urlstr string, data []byte, opts ...Option) *Response {
	return client.Do(ctx, "PUT", urlstr, bytes.NewBuffer(data), opts...)
}

// PostForm post form
func (client *clientImpl) PostForm(ctx context.Context, urlstr string, data map[string]interface{}, opts ...Option) *Response {
	values := url.Values{}
	for k, v := range data {
		values.Set(k, fmt.Sprint(v))
	}
	opts = append(opts, WithHeader("Content-Type", "application/x-www-form-urlencoded"))
	return client.Post(ctx, urlstr, []byte(values.Encode()), opts...)
}

// PostJSON post json
func (c *clientImpl) PostJSON(ctx context.Context, urlstr string, data interface{}, opts ...Option) *Response {
	var payload []byte
	var err error
	switch d := data.(type) {
	case string:
		payload = []byte(d)
	case []byte:
		payload = d
	case nil:
		// do nothing
	case io.Reader:
		payload, err = ioutil.ReadAll(d)
		if err != nil {
			return buildResponse(nil, err)
		}
	default:
		payload, err = json.Marshal(data)
		if err != nil {
			return buildResponse(nil, err)
		}
	}
	opts = append(opts, WithHeader("Content-Type", "application/json; charset=utf-8"))
	return c.Post(ctx, urlstr, payload, opts...)
}

func (client *clientImpl) makeFinalHandler(extraMiddlewares ...Middleware) Endpoint {
	next := client.Client.Do

	next = middlewareContext(next)

	for i := len(extraMiddlewares) - 1; i >= 0; i-- {
		next = extraMiddlewares[i](next)
	}
	for i := len(client.middlewares) - 1; i >= 0; i-- {
		next = client.middlewares[i](next)
	}
	/* must create context */
	next = middlewareInitCtx(next)

	return next
}

func (client *clientImpl) getOptionMiddlewares(opts ...Option) []Middleware {
	opt := newOptions()
	for _, fn := range opts {
		fn(opt)
	}
	return opt.Middlewares
}

type Doer func(*syshttp.Request) (*syshttp.Response, error)

func (hd Doer) Do(req *syshttp.Request) (*syshttp.Response, error) {
	return hd(req)
}

func DefaultPooledTransport() *syshttp.Transport {
	transport := &syshttp.Transport{
		Proxy: syshttp.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}
	return transport
}
