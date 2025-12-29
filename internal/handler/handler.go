package handler

import (
	"slices"

	"github.com/noelw19/tcptohttp/internal/middleware.go"
	"github.com/noelw19/tcptohttp/internal/request"
	"github.com/noelw19/tcptohttp/internal/response"
)

type AllowedMethod string

const (
	GET    AllowedMethod = "GET"
	POST   AllowedMethod = "POST"
	PATCH  AllowedMethod = "PATCH"
	DELETE AllowedMethod = "DELETE"
)

type Params map[string]string
type Vars map[string]string

type HandlerFunc func(w *response.Writer, req *request.Request)
type Handler struct {
	route          string
	MethodFuncs    map[AllowedMethod]*HandlerFunc
	HandleFunc     *HandlerFunc
	AllowedMethods []AllowedMethod
	Vars           Vars
	Params         Params
	middlewares    []middleware.MiddlewareHandler
}

func NewHandler(route string, hf HandlerFunc) Handler {
	handler := Handler{
		route:          route,
		HandleFunc:     &hf,
		MethodFuncs:    map[AllowedMethod]*HandlerFunc{},
		AllowedMethods: []AllowedMethod{},
		Vars:           Vars{},
		Params:         Params{},
		middlewares:    []middleware.MiddlewareHandler{},
	}

	return handler
}

func (h *Handler) ExecuteMiddlewares(w *response.Writer, r *request.Request, final middleware.MiddlewareFunc) middleware.MiddlewareFunc {
	middlewares := slices.Clone(h.middlewares)
	slices.Reverse(middlewares)
	finalHandler := middleware.MiddlewareFunc(final)

	for _, m := range middlewares {
		finalHandler = m(finalHandler)
	}

	return finalHandler
}

func (h *Handler) Use(m middleware.MiddlewareHandler) *Handler {
	h.middlewares = append(h.middlewares, m)
	return h
}

func (h *Handler) GET() *Handler {
	h.MethodFuncs[GET] = h.HandleFunc
	return h
}

func (h *Handler) POST() *Handler {
	h.MethodFuncs[POST] = h.HandleFunc
	return h
}

func (h *Handler) PATCH() *Handler {
	h.MethodFuncs[PATCH] = h.HandleFunc
	return h
}

func (h *Handler) DELETE() *Handler {
	h.MethodFuncs[DELETE] = h.HandleFunc
	return h
}
