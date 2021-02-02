package http

import (
	"io"
	syshttp "net/http"
	"strings"
	"time"
)

type options struct {
	Middlewares []Middleware
}

type Option func(*options)

/* private methods */
func newOptions() *options {
	return &options{}
}

/* option middlewares */
func WithMiddleware(m Middleware) Option {
	return func(opt *options) {
		opt.Middlewares = append(opt.Middlewares, m)
	}
}

func WithBeforeHook(hook func(*syshttp.Request)) Option {
	return WithMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			hook(req)
			return next(req)
		}
	})
}

func WithTimeout(tm time.Duration) Option {
	return WithMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			getValue(req).Timeout = tm
			return next(req)
		}
	})
}

func WithRetry(opt RetryOption) Option {
	return WithMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			getValue(req).RetryOption = &opt
			return next(req)
		}
	})
}

func WithBody(w io.Writer) Option {
	return WithMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			getValue(req).BodySaver = w
			return next(req)
		}
	})
}

func WithAfterHook(hook func(*syshttp.Response)) Option {
	return WithMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			res, err := next(req)
			if err == nil && res != nil {
				hook(res)
			}
			return res, err
		}
	})
}

func WithHeaders(hdr map[string]string) Option {
	return WithMiddleware(func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			setRequestHeader(req, hdr)
			return next(req)
		}
	})
}

func WithHeader(k, v string) Option {
	return WithHeaders(map[string]string{k: v})
}

type RetryHook func(*syshttp.Request, int)

type RetryOption struct {
	RetryMax      int
	RetryWaitMin  time.Duration                                     // optional
	RetryWaitMax  time.Duration                                     // optional
	CheckResponse func(*syshttp.Response, error) (shouldRetry bool) // optional
}

func setRequestHeader(req *syshttp.Request, header map[string]string) {
	for k, v := range header {
		req.Header.Set(k, v)
		if strings.ToLower(k) == "host" {
			req.Host = v
		}
	}
}
