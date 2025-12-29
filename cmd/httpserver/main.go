package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/noelw19/tcptohttp/internal/middleware.go"
	"github.com/noelw19/tcptohttp/internal/request"
	"github.com/noelw19/tcptohttp/internal/response"
	"github.com/noelw19/tcptohttp/internal/server"
)

const port = 42069

func main() {
	server := server.Serve(port)

	server.Use(func(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
		return func(w *response.Writer, req *request.Request) {
			fmt.Println("log 1")
			next(w, req)
		}
	})

	server.Use(func(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
		return func(w *response.Writer, req *request.Request) {
			fmt.Println("log 2")
			next(w, req)

		}
	})

	server.Use(func(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
		return func(w *response.Writer, req *request.Request) {
			fmt.Println("log 3")
			next(w, req)
		}
	})

	server.AddHandler("/test", wakandaHandler).GET()
	server.AddHandler("/wakanda", wakandaHandler).GET()
	server.AddHandler("/wakanda", wakandaPOSTHandler).POST()
	server.AddHandler("/wakanda/{id}/{lala}", wakandaIDHandler).GET()
	server.AddHandler("/query", queryHandler).GET().Use(func(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
		return func(w *response.Writer, req *request.Request) {
			fmt.Println("specfic middleware")
			next(w, req)
		}
	}).Use(func(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
		return func(w *response.Writer, req *request.Request) {
			fmt.Println("specfic middleware 1")
			next(w, req)
		}
	})
	server.AddHandler("/httpbin/stream", streamHandler)
	server.AddHandler("/video", videoHandler)

	log.Println("Server started on port", port)

	server.Listen()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	server.Close()
	log.Println("Server gracefully stopped")
}
