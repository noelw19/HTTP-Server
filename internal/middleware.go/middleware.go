package middleware

import (
	"github.com/noelw19/tcptohttp/internal/request"
	"github.com/noelw19/tcptohttp/internal/response"
)

type MiddlewareFunc func(w *response.Writer, req *request.Request)
type MiddlewareHandler func(next MiddlewareFunc) MiddlewareFunc
