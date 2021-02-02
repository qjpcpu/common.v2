package http

import (
	"context"
	"io"
	syshttp "net/http"
	"time"
)

type contextKey string

const (
	keyContext = contextKey("http-context")
)

type gValue struct {
	BodySaver   io.Writer
	Timeout     time.Duration
	Mock        Endpoint
	Debugger    HTTPLogger
	RetryOption *RetryOption
	RetryHooks  []RetryHook
}

func getValue(req *syshttp.Request) *gValue {
	if gv := req.Context().Value(keyContext); gv == nil {
		return nil
	} else if val, ok := gv.(*gValue); ok {
		return val
	}
	return nil
}

func getOrCreateValue(req *syshttp.Request) *gValue {
	if gv := getValue(req); gv == nil {
		gv := &gValue{}
		return gv
	} else {
		return gv
	}
}

func setValue(req *syshttp.Request, v *gValue) *syshttp.Request {
	if v != nil {
		ctx := context.WithValue(req.Context(), keyContext, v)
		req = req.WithContext(ctx)
	}
	return req
}

func (v *gValue) AddRetryHook(hook RetryHook) {
	v.RetryHooks = append(v.RetryHooks, hook)
}
