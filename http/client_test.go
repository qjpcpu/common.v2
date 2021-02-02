package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetMock(t *testing.T) {
	suite := assert.New(t)
	client := NewClient()
	res := &http.Response{}
	var val int
	client.SetMock(func(*http.Request) (*http.Response, error) {
		val = 1
		return res, nil
	})

	res1 := client.Get(nil, "http://ssssss")
	suite.Nil(res1.Err)
	suite.Equal(1, val)
	suite.Equal(res, res1.Response)
}

func TestMiddleware(t *testing.T) {
	suite := assert.New(t)
	client := NewClient()
	var val int
	server := NewMockServer().Handle("/hello", func(w http.ResponseWriter, req *http.Request) {
		val = 1
		w.Write([]byte("OK"))
	})
	defer server.ServeBackground()()

	var slice []int
	client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *http.Request) (*http.Response, error) {
			slice = append(slice, 1)
			a, b := next(req)
			slice = append(slice, 2)
			return a, b
		}
	})
	client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *http.Request) (*http.Response, error) {
			slice = append(slice, 3)
			return next(req)
		}
	})

	res1 := client.Get(nil, server.URLPrefix+"/hello")
	suite.Nil(res1.Err)
	suite.Equal(1, val)
	suite.ElementsMatch([]int{1, 3, 2}, slice)
}

func TestResponse(t *testing.T) {
	suite := assert.New(t)
	server := NewMockServer().Handle("/hello", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(`{"a":1,"b":"HELLO"}`))
	})
	defer server.ServeBackground()()

	client := NewClient()

	res := struct {
		A int    `json:"a"`
		B string `json:"b"`
	}{}
	res1 := client.Post(nil, server.URLPrefix+"/hello", nil)
	suite.Nil(res1.Unmarshal(&res))
	suite.Nil(res1.Err)
	suite.Equal(1, res.A)
	suite.Equal("HELLO", res.B)
}

func TestGet(t *testing.T) {
	suite := assert.New(t)
	server := NewMockServer().Handle("/get", func(w http.ResponseWriter, req *http.Request) {
		args := make(map[string]string)
		qs := req.URL.Query()
		for k := range qs {
			args[k] = qs.Get(k)
		}
		data, _ := json.Marshal(map[string]interface{}{
			"args": args,
		})
		w.Write(data)
	})
	defer server.ServeBackground()()

	client := NewClient()
	res := struct {
		Args struct {
			A string `json:"a"`
		} `json:"args"`
	}{}
	res1 := client.Get(nil, server.URLPrefix+"/get?a=hello")
	suite.Nil(res1.Unmarshal(&res))
	suite.Nil(res1.Err)
	suite.Equal("hello", res.Args.A)
}

func interceptStdout() func() []byte {
	stdout := os.Stdout
	stderr := os.Stderr
	fname := filepath.Join(os.TempDir(), "stdout")
	temp, _ := os.Create(fname)
	os.Stdout = temp
	os.Stderr = temp
	return func() []byte {
		temp.Sync()
		data, _ := ioutil.ReadFile(fname)
		temp.Close()
		os.Remove(fname)
		os.Stderr = stderr
		os.Stdout = stdout
		return data
	}
}

func TestDebug(t *testing.T) {
	stdout := interceptStdout()
	server := NewMockServer()
	defer server.ServeBackground()()

	suite := assert.New(t)
	client := NewClient().SetDebug(DefaultLogger)
	res := struct {
		A int    `json:"a"`
		B string `json:"b"`
	}{
		A: 100,
		B: "HELLO",
	}
	res1 := client.PostJSON(nil, server.URLPrefix+"/echo", res)
	suite.Nil(res1.Err)
	suite.Nil(res1.Unmarshal(&res))
	out := string(stdout())

	suite.Contains(out, server.URLPrefix+"/echo")
	suite.Contains(out, `200 OK`)
	suite.Contains(out, `"{\"a\":100,\"b\":\"HELLO\"}"`)
	suite.Contains(out, `application/json`)
	suite.Contains(out, `Response-Headers`)
	suite.Contains(out, `Request-Body`)
	suite.Contains(out, `[Response-Body]`)
}

func TestDebugWithErr(t *testing.T) {
	stdout := interceptStdout()
	suite := assert.New(t)
	client := NewClient().SetDebug(DefaultLogger)
	client.SetMock(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("user error")
	})
	res := struct {
		A int    `json:"a"`
		B string `json:"b"`
	}{
		A: 100,
		B: "HELLO",
	}
	res1 := client.PostJSON(nil, "http://wwws", res)
	suite.NotNil(res1.Err)
	out := string(stdout())
	suite.Contains(out, `[Response Error]`)
	suite.Contains(out, `user error`)
}

func TestRepeatableRead(t *testing.T) {
	suite := assert.New(t)
	client := NewClient()
	res := &http.Response{
		Body: ioutil.NopCloser(strings.NewReader("HELLO")),
	}
	client.SetMock(func(*http.Request) (*http.Response, error) {
		return res, nil
	})
	client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *http.Request) (*http.Response, error) {
			res, err := next(req)
			RepeatableReadResponse(res)
			RepeatableReadResponse(res)
			RepeatableReadResponse(res)
			return res, err
		}
	})

	res1 := client.Get(nil, "http://sss")
	suite.Nil(res1.Err)
	suite.Equal("HELLO", string(res1.MustGetBody()))
}

func TestRepeatableReadRequest(t *testing.T) {
	suite := assert.New(t)
	client := NewClient()
	reqBody := `GOGOGOGOGOGO`
	res := &http.Response{
		Body: ioutil.NopCloser(strings.NewReader("HELLO")),
	}
	client.SetMock(func(req *http.Request) (*http.Response, error) {
		data, e := ioutil.ReadAll(req.Body)
		suite.Nil(e)
		suite.Equal(reqBody, string(data))
		return res, nil
	})
	client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *http.Request) (*http.Response, error) {
			data, e := RepeatableReadRequest(req)
			suite.Nil(e)
			suite.Equal(reqBody, string(data))
			RepeatableReadRequest(req)
			return next(req)
		}
	})

	res1 := client.Post(nil, "http://sss", []byte(reqBody))
	suite.Nil(res1.Err)
}

func TestGlobalHeader(t *testing.T) {
	suite := assert.New(t)
	client := NewClient()
	res := &http.Response{
		Body: ioutil.NopCloser(strings.NewReader("HELLO")),
	}
	hdl := make(map[string]string)
	client.SetMock(func(req *http.Request) (*http.Response, error) {
		for k := range req.Header {
			hdl[strings.ToLower(k)] = req.Header.Get(k)
		}
		return res, nil
	})
	client.SetHeader("AA", "BB")

	res1 := client.Get(nil, "http://sss")
	suite.Nil(res1.Err)
	suite.Equal("BB", hdl["aa"])
}

func TestOptionMiddleware(t *testing.T) {
	suite := assert.New(t)
	client := NewClient()
	res := &http.Response{
		Body: ioutil.NopCloser(strings.NewReader("HELLO")),
	}
	client.SetMock(func(req *http.Request) (*http.Response, error) {
		return res, nil
	})
	var val int
	mid := func(next Endpoint) Endpoint {
		return func(req *http.Request) (*http.Response, error) {
			val++
			return next(req)
		}
	}
	res1 := client.Get(nil, "http://sss", WithMiddleware(mid))
	suite.Nil(res1.Err)
	res1 = client.Get(nil, "http://sss")
	suite.Nil(res1.Err)
	/* execute once */
	suite.Equal(1, val)
}

type httpClientor interface {
	Do(*http.Request) (*http.Response, error)
}

func runHTTP(h httpClientor) (*http.Response, error) {
	req, _ := http.NewRequest("GET", "http://sss", nil)
	return h.Do(req)
}

func TestDoer(t *testing.T) {
	suite := assert.New(t)
	client := NewClient()
	res := &http.Response{
		Body: ioutil.NopCloser(strings.NewReader("HELLO")),
	}
	client.SetMock(func(req *http.Request) (*http.Response, error) {
		return res, nil
	})
	var val int
	mid := func(next Endpoint) Endpoint {
		return func(req *http.Request) (*http.Response, error) {
			val++
			return next(req)
		}
	}
	doer := client.MakeDoer(WithMiddleware(mid))
	_, err := runHTTP(doer)
	suite.Nil(err)
	suite.Equal(1, val)
}

func TestBeforeHook(t *testing.T) {
	suite := assert.New(t)
	client := NewClient()
	res := &http.Response{
		Body: ioutil.NopCloser(strings.NewReader("HELLO")),
	}
	client.SetMock(func(req *http.Request) (*http.Response, error) {
		return res, nil
	})
	var val int
	res1 := client.Get(nil, "http://sss", WithBeforeHook(func(*http.Request) { val++ }))
	suite.Nil(res1.Err)
	suite.Equal(1, val)
}

func TestAfterHook(t *testing.T) {
	suite := assert.New(t)
	client := NewClient()
	res := &http.Response{
		Body: ioutil.NopCloser(strings.NewReader("HELLO")),
	}
	client.SetMock(func(req *http.Request) (*http.Response, error) {
		return res, nil
	})
	var val int
	res1 := client.Get(nil, "http://sss", WithAfterHook(func(*http.Response) { val++ }))
	suite.Nil(res1.Err)
	suite.Equal(1, val)
}

func TestTimeout(t *testing.T) {
	suite := assert.New(t)
	stopChan := make(chan struct{}, 1)
	server := NewMockServer().Handle("/delay", func(w http.ResponseWriter, req *http.Request) {
		select {
		case <-time.After(1 * time.Hour):
		case <-stopChan:
		}
		w.Write([]byte("OK"))
	})
	defer server.ServeBackground()()

	client := NewClient()
	client.SetTimeout(1 * time.Millisecond)
	res := client.Get(nil, server.URLPrefix+"/delay")
	suite.NotNil(res.Err)
	suite.Contains(res.Err.Error(), "context deadline exceeded")

	close(stopChan)
}

func TestTimeoutOverwrite(t *testing.T) {
	suite := assert.New(t)
	stopChan := make(chan struct{}, 1)
	server := NewMockServer().Handle("/delay", func(w http.ResponseWriter, req *http.Request) {
		select {
		case <-time.After(30 * time.Millisecond):
		case <-stopChan:
		}
		w.Write([]byte("OK"))
	})
	defer server.ServeBackground()()

	client := NewClient()
	client.SetTimeout(100 * time.Hour)
	err := client.Get(nil, server.URLPrefix+"/delay", WithTimeout(time.Millisecond)).Err
	suite.NotNil(err)
	suite.True(strings.Contains(err.Error(), `context deadline exceeded`) ||
		strings.Contains(err.Error(), `timeout`))
	close(stopChan)
}

func TestTimeoutOverwrite2(t *testing.T) {
	suite := assert.New(t)
	stopChan := make(chan struct{}, 1)
	server := NewMockServer().Handle("/delay", func(w http.ResponseWriter, req *http.Request) {
		select {
		case <-time.After(3 * time.Millisecond):
		case <-stopChan:
		}
		w.Write([]byte("OK"))
	})
	defer server.ServeBackground()()

	client := NewClient()
	client.SetTimeout(1 * time.Millisecond)
	/* should not timeout */
	err := client.Get(nil, server.URLPrefix+"/delay", WithTimeout(time.Hour)).Err
	suite.Nil(err)
	close(stopChan)
}

func TestDownload(t *testing.T) {
	suite := assert.New(t)
	body := `"BJLKJLJLJL:JL:JKLJ`
	server := NewMockServer().Handle("/hello", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(body))
	})
	defer server.ServeBackground()()

	client := NewClient()

	buf := &bytes.Buffer{}
	err := client.Download(nil, server.URLPrefix+"/hello", buf)
	suite.Nil(err)
	suite.Equal(body, buf.String())
}

func TestMockServer(t *testing.T) {
	suite := assert.New(t)
	body := `"BJLKJLJLJL:JL:JKLJ`
	server := NewMockServer().Handle("/hello", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(body))
	})
	defer server.ServeBackground()()
	client := NewClient()

	res := client.Get(nil, server.URLPrefix+"/hello")
	suite.Nil(res.Err)
	suite.Equal(body, string(res.MustGetBody()))
}

func TestRetryCheckResponse(t *testing.T) {
	suite := assert.New(t)
	var val int
	server := NewMockServer().Handle("/hello", func(w http.ResponseWriter, req *http.Request) {
		val++
		if val < 3 {
			w.Write([]byte("FAIL"))
			return
		}
		w.Write([]byte("OK"))
	})
	defer server.ServeBackground()()
	client := NewClient()

	res := client.Get(nil, server.URLPrefix+"/hello", WithRetry(RetryOption{
		RetryMax:     3,
		RetryWaitMin: 1 * time.Millisecond,
		RetryWaitMax: 3 * time.Millisecond,
		CheckResponse: func(res *http.Response, err error) bool {
			data, _ := RepeatableReadResponse(res)
			return string(data) == "FAIL"
		},
	}))
	suite.Nil(res.Err)
	suite.Equal(3, val)
	suite.Equal("OK", string(res.MustGetBody()))
}

func TestRetryModifyRequest(t *testing.T) {
	suite := assert.New(t)
	var val int
	server := NewMockServer().Handle("/hello", func(w http.ResponseWriter, req *http.Request) {
		val++
		if val > 1 {
			suite.Contains(req.URL.String(), "second")
			suite.NotContains(req.URL.String(), "first")
		} else {
			suite.Contains(req.URL.String(), "first")
			suite.NotContains(req.URL.String(), "second")
		}
		if val < 3 {
			w.Write([]byte("FAIL"))
			return
		}
		w.Write([]byte("OK"))
	})
	defer server.ServeBackground()()
	client := NewClient()

	res := client.Get(nil, server.URLPrefix+"/hello?args=first", WithRetry(RetryOption{
		RetryMax:     3,
		RetryWaitMin: 1 * time.Millisecond,
		RetryWaitMax: 3 * time.Millisecond,
		CheckResponse: func(res *http.Response, err error) bool {
			data, _ := RepeatableReadResponse(res)
			return string(data) == "FAIL"
		},
	}), WithMiddleware(func(next Endpoint) Endpoint {
		return func(req *http.Request) (*http.Response, error) {
			FromRequest(req).AddRetryHook(func(nreq *http.Request, i int) {
				qs := nreq.URL.Query()
				qs.Set("args", "second")
				nreq.URL.RawQuery = qs.Encode()
			})
			return next(req)
		}
	}))
	suite.Nil(res.Err)
	suite.Equal(3, val)
	suite.Equal("OK", string(res.MustGetBody()))
}

func TestRetryModifyRequestByPrevMiddleware(t *testing.T) {
	suite := assert.New(t)
	var val int
	server := NewMockServer().Handle("/hello", func(w http.ResponseWriter, req *http.Request) {
		val++
		if val > 1 {
			suite.Contains(req.URL.String(), "extra")
			suite.NotContains(req.URL.String(), "first")
		} else {
			suite.Contains(req.URL.String(), "first")
			suite.NotContains(req.URL.String(), "extra")
		}
		if val < 3 {
			w.Write([]byte("FAIL"))
			return
		}
		w.Write([]byte("OK"))
	})
	defer server.ServeBackground()()
	client := NewClient()
	client.AddMiddleware(func(next Endpoint) Endpoint {
		return func(req *http.Request) (*http.Response, error) {
			ctx := req.Context()
			req = req.WithContext(context.WithValue(ctx, "prev", "extra"))
			return next(req)
		}
	})

	res := client.Get(nil, server.URLPrefix+"/hello?args=first", WithRetry(RetryOption{
		RetryMax:     3,
		RetryWaitMin: 1 * time.Millisecond,
		RetryWaitMax: 3 * time.Millisecond,
		CheckResponse: func(res *http.Response, err error) bool {
			data, _ := RepeatableReadResponse(res)
			return string(data) == "FAIL"
		},
	}), WithMiddleware(func(next Endpoint) Endpoint {
		return func(req *http.Request) (*http.Response, error) {
			FromRequest(req).AddRetryHook(func(nreq *http.Request, i int) {
				if v := nreq.Context().Value("prev"); v != nil {
					qs := nreq.URL.Query()
					qs.Set("args", v.(string))
					nreq.URL.RawQuery = qs.Encode()
				}
			})
			return next(req)
		}
	}))
	suite.Nil(res.Err)
	suite.Equal(3, val)
	suite.Equal("OK", string(res.MustGetBody()))
}

func TestOverwriteRetry(t *testing.T) {
	suite := assert.New(t)
	var val int
	client := NewClient()
	client.SetMock(func(req *http.Request) (*http.Response, error) {
		val++
		return nil, errors.New("err")
	})

	client.AddMiddleware(RetryMiddleware(RetryOption{RetryMax: 2}))

	res := client.Get(nil, "http://hello", WithRetry(RetryOption{
		RetryMax: 0,
	}))
	suite.NotNil(res.Err)
	suite.Equal(1, val)
}

func TestSetHeader(t *testing.T) {
	suite := assert.New(t)
	server := NewMockServer().Handle("/header", func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Host") == "" {
			req.Header.Set("Host", req.Host)
		}
		data, _ := json.Marshal(req.Header)
		t.Log(string(data))
		w.Write(data)
	})
	defer server.ServeBackground()()
	client := NewClient()

	client.SetHeader("AA", "BB")

	res1 := client.Get(nil, server.URLPrefix+"/header", WithHeaders(map[string]string{
		"c":    "eS",
		"host": "www.baidu.com",
	}))
	suite.Nil(res1.Err)
	headers := make(http.Header)
	res1.Unmarshal(&headers)
	suite.Equal("BB", headers.Get("AA"))
	suite.Equal("eS", headers.Get("c"))
	suite.Equal("www.baidu.com", headers.Get("host"))
}
func TestContextCancel(t *testing.T) {
	suite := assert.New(t)
	body := []byte(strings.Repeat("x", 65535))
	server := NewMockServer().Handle("/header", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("content-length", fmt.Sprint(len(body)))
		w.Write(body)
	})
	defer server.ServeBackground()()
	client := NewClient()
	res1 := client.Get(nil, server.URLPrefix+"/header")
	suite.Nil(res1.Err)
	data, err := res1.GetBody()
	suite.Nil(err)
	i, _ := strconv.Atoi(res1.Header.Get("Content-Length"))
	suite.Equal(i, len(data))
}

func TestDownload2(t *testing.T) {
	suite := assert.New(t)
	body := []byte(strings.Repeat("x", 65535))
	server := NewMockServer().Handle("/header", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("content-length", fmt.Sprint(len(body)))
		w.Write(body)
	})
	defer server.ServeBackground()()
	client := NewClient()
	buf := new(bytes.Buffer)
	err := client.Download(nil, server.URLPrefix+"/header", buf)
	suite.Nil(err)
	suite.Equal(len(body), buf.Len())
}

func TestStatusCode(t *testing.T) {
	suite := assert.New(t)
	body := []byte(`BODY`)
	server := NewMockServer().Handle("/code", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(500)
		w.Header().Set("content-length", fmt.Sprint(len(body)))
		w.Write(body)
	})
	defer server.ServeBackground()()
	client := NewClient().AddMiddleware(MiddlewareSetAllowedStatusCode(http.StatusOK))
	res1 := client.Get(nil, server.URLPrefix+"/code")
	suite.NotNil(res1.Err)
	suite.Equal(`500 Internal Server Error BODY`, res1.Err.Error())

	client = NewClient().AddMiddleware(MiddlewareSetBlockedStatusCode(http.StatusInternalServerError))
	res1 = client.Get(nil, server.URLPrefix+"/code")
	suite.NotNil(res1.Err)
	suite.Equal(`500 Internal Server Error BODY`, res1.Err.Error())
}
