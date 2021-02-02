package http

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	syshttp "net/http"
	"os"
	"strings"
	"time"
)

type Endpoint func(*syshttp.Request) (*syshttp.Response, error)

type Middleware func(Endpoint) Endpoint

func middlewareInitCtx(next Endpoint) Endpoint {
	return func(req *syshttp.Request) (*syshttp.Response, error) {
		req = setValue(req, getOrCreateValue(req))
		return next(req)
	}
}

func middlewareContext(next Endpoint) Endpoint {
	return func(req *syshttp.Request) (*syshttp.Response, error) {
		gv := getValue(req)
		if gv == nil {
			return next(req)
		}

		/* mock */
		if gv.Mock != nil {
			next = middlewareSetMock(gv.Mock)(next)
		}

		/* download body */
		next = middlewareSaveResponse(gv.BodySaver)(next)

		/* log */
		if gv.Debugger != nil {
			next = middlewareDebug(gv.Debugger)(next)
		}

		/* timeout */
		if gv.Timeout > 0 {
			next = middlewareTimeout(gv.Timeout)(next)
		}

		/* retry */
		if gv.RetryOption != nil && gv.RetryOption.RetryMax > 0 {
			next = middlewareRetry(gv.RetryOption)(next)
		}
		return next(req)
	}
}

func middlewareSetMock(fn func(*syshttp.Request) (*syshttp.Response, error)) Middleware {
	return func(next Endpoint) Endpoint {
		return fn
	}
}

func middlewareTimeout(tm time.Duration) Middleware {
	return func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			ctx, cancel := context.WithTimeout(req.Context(), tm)
			defer cancel()
			req = req.WithContext(ctx)
			res, err := next(req)
			if err != nil && (strings.Contains(err.Error(), `context`) || strings.Contains(err.Error(), "timeout")) {
				err = fmt.Errorf("%v timeout:%v", err, tm)
			}
			return res, err
		}
	}
}

func middlewareSaveResponse(w io.Writer) Middleware {
	return func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			res, err := next(req)
			if res != nil && res.Body != nil {
				if w != nil {
					defer res.Body.Close()
					if _, err := io.Copy(w, res.Body); err != nil {
						return nil, err
					}
				} else {
					if _, err := RepeatableReadResponse(res); err != nil {
						return nil, err
					}
				}
			}

			return res, err
		}
	}
}

type HTTPLogger func(context.Context, *TransportInfo)

type TransportEntity struct {
	Header http.Header
	Body   func() []byte
}

type TransportInfo struct {
	Method   string
	URL      string
	Status   string
	StartAt  time.Time
	Cost     time.Duration
	Err      error
	Request  *TransportEntity
	Response *TransportEntity
}

func DefaultLogger(ctx context.Context, info *TransportInfo) {
	w := os.Stdout
	/* status line */
	fmt.Fprintf(
		w,
		"[%s] %s %s reqat:%s cost:%v\n",
		info.Method,
		info.URL,
		info.Status,
		info.StartAt.Format("2006-01-02 15:04:05.000"),
		info.Cost,
	)
	/* request */
	fmt.Fprintln(w, "[Request-Headers]")
	for k := range info.Request.Header {
		fmt.Fprintf(w, "  %s:%s\n", k, info.Request.Header.Get(k))
	}
	if reqBody := info.Request.Body(); len(reqBody) > 0 {
		fmt.Fprintf(w, "[Request-Body]\n%s\n", reqBody)
	} else {
		fmt.Fprintln(w, "[Request-Body]")
	}
	/* response */
	if err := info.Err; err != nil {
		fmt.Fprintf(w, "[Response Error]:%s\n", err)
	} else {
		fmt.Fprintln(w, "[Response-Headers]")
		for k := range info.Response.Header {
			fmt.Fprintf(w, "  %s:%s\n", k, info.Response.Header.Get(k))
		}
		if resBody := info.Response.Body(); len(resBody) > 0 {
			fmt.Fprintf(w, "[Response-Body]\n%s\n", resBody)
		} else {
			fmt.Fprintln(w, "[Response-Body]")
		}
	}
}

func middlewareDebug(loggerFn HTTPLogger) Middleware {
	return func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			if loggerFn == nil {
				return next(req)
			}
			info := &TransportInfo{}
			info.Status = "-1"
			info.Method = req.Method
			info.URL = req.URL.String()
			info.Request = &TransportEntity{
				Header: req.Header,
			}
			reqBody, _ := RepeatableReadRequest(req)
			info.Request.Body = func() []byte {
				return reqBody
			}
			now := time.Now()
			res, err := next(req)
			info.StartAt = now
			info.Cost = time.Since(now)
			info.Response = &TransportEntity{}
			if err != nil {
				info.Err = err
			} else {
				info.Status = res.Status
				info.Response.Header = res.Header
				info.Response.Body = func() []byte {
					resBody, _ := RepeatableReadResponse(res)
					return resBody
				}
			}
			loggerFn(req.Context(), info)
			return res, err
		}
	}
}

func RetryMiddleware(retryOpt RetryOption) Middleware {
	return func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			getValue(req).RetryOption = &retryOpt
			return next(req)
		}
	}
}

func middlewareRetry(retryOpt *RetryOption) Middleware {
	if retryOpt.RetryWaitMin <= 0 {
		retryOpt.RetryWaitMin = 1 * time.Second
	}
	if retryOpt.RetryWaitMax <= 0 {
		retryOpt.RetryWaitMax = 3 * time.Second
	}
	shouldRetry := func(res *syshttp.Response, err error) bool {
		return err != nil
	}
	if retryOpt.CheckResponse != nil {
		shouldRetry = retryOpt.CheckResponse
	}
	return func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (res *syshttp.Response, err error) {
			retryHookList := getValue(req).RetryHooks
			for i := 0; i < retryOpt.RetryMax+1; i++ {
				/* save request body */
				if req.Body != nil {
					if _, err := RepeatableReadRequest(req); err != nil {
						return nil, err
					}
				}

				/* do retry hook */
				if i > 0 {
					for _, hook := range retryHookList {
						hook(req, i)
					}
				}

				/* do request */
				res, err = next(req)
				if !shouldRetry(res, err) {
					break
				}

				if res != nil && res.Body != nil {
					drainBody(res.Body)
				}
				if i < retryOpt.RetryMax {
					time.Sleep(linearJitterBackoff(retryOpt.RetryWaitMin, retryOpt.RetryWaitMax, i))
				}
			}
			return
		}
	}
}

func drainBody(body io.ReadCloser) error {
	defer body.Close()
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, 4096))
	return err
}

func linearJitterBackoff(min, max time.Duration, attemptNum int) time.Duration {
	// attemptNum always starts at zero but we want to start at 1 for multiplication
	attemptNum++

	if max <= min {
		// Unclear what to do here, or they are the same, so return min *
		// attemptNum
		return min * time.Duration(attemptNum)
	}

	// Seed rand; doing this every time is fine
	rand := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	// Pick a random number that lies somewhere between the min and max and
	// multiply by the attemptNum. attemptNum starts at zero so we always
	// increment here. We first get a random percentage, then apply that to the
	// difference between min and max, and add to min.
	jitter := rand.Float64() * float64(max-min)
	jitterMin := int64(jitter) + int64(min)
	return time.Duration(jitterMin * int64(attemptNum))
}

func MiddlewareSetAllowedStatusCode(codes ...int) Middleware {
	codeMap := make(map[int]bool)
	for _, code := range codes {
		codeMap[code] = true
	}
	check := func(c int) bool {
		if len(codes) == 0 {
			return true
		}
		return codeMap[c]
	}
	return MiddlewareCheckStatusCode(check)
}

func MiddlewareSetBlockedStatusCode(codes ...int) Middleware {
	codeMap := make(map[int]bool)
	for _, code := range codes {
		codeMap[code] = true
	}
	check := func(c int) bool {
		if len(codes) == 0 {
			return true
		}
		return !codeMap[c]
	}
	return MiddlewareCheckStatusCode(check)
}

func MiddlewareCheckStatusCode(fn func(int) bool) Middleware {
	return func(next Endpoint) Endpoint {
		return func(req *syshttp.Request) (*syshttp.Response, error) {
			resp, err := next(req)
			if err != nil {
				return resp, err
			}
			if !fn(resp.StatusCode) {
				data, _ := RepeatableReadResponse(resp)
				return nil, fmt.Errorf("%s %s", resp.Status, data)
			}
			return resp, err
		}
	}
}
