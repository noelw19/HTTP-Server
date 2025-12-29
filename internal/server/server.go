package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/noelw19/tcptohttp/internal/handler"
	"github.com/noelw19/tcptohttp/internal/headers"
	"github.com/noelw19/tcptohttp/internal/middleware.go"
	"github.com/noelw19/tcptohttp/internal/request"
	"github.com/noelw19/tcptohttp/internal/response"
)

type HandlerError struct {
	StatusCode int
	Message    string
}

func (h HandlerError) Write(w io.Writer) {
	fmt.Fprintf(w, "HTTP/1.1 %d %s", h.StatusCode, h.Message)
}

type Server struct {
	Listener   net.Listener
	port       int
	running    bool
	notFound   handler.HandlerFunc
	handlers   *handler.Handlers
	middleware []middleware.MiddlewareHandler
}

func (s *Server) Show() {
	for r := range *s.handlers {
		fmt.Printf("%+v\n", (*s.handlers)[r])

	}
}

func Serve(port int) *Server {
	server := &Server{
		port:       port,
		running:    false,
		handlers:   &handler.Handlers{},
		middleware: []middleware.MiddlewareHandler{},
	}
	server.OverrideNotFoundHandler(defaultNotFoundHandler)

	return server
}

func (s *Server) Close() error {
	s.running = false
	if s.Listener != nil {
		return s.Listener.Close()
	}
	return nil
}

func (s *Server) Listen() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	s.Listener = listener

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// If the listener was closed (expected during shutdown), break the loop
				if errors.Is(err, net.ErrClosed) || !s.running {
					break
				}
				// Only log unexpected errors
				if s.running {
					fmt.Println(err)
				}
				continue
			}

			s.running = true
			go s.handle(conn)
		}
	}()
	return nil
}

func (s *Server) AddHandler(route string, handleFunc handler.HandlerFunc) *handler.Handler {
	if !strings.Contains(route, "/") {
		log.Fatalf("Route %s is implimented wrong, be sure to add a / before the route path", route)
	}

	handler := s.handlers.Add(route, handleFunc)
	return handler
}

func (s *Server) handle(conn net.Conn) {
	// defer conn.Close()

	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(30 * time.Second)
	}

	// ✅ Set read deadline to detect closed connections
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		req, err := request.RequestFromReader(conn)
		if err != nil {
			// Check for timeout (no data received within deadline)
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Connection timed out - this is normal for keep-alive
				// Just close the connection silently
				break
			}

			// Check for EOF or closed connection
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				// Client closed the connection
				break
			}

			// For other errors, log and close connection
			fmt.Println("Error reading request:", err)
			break
		}

		// Validate that we got a proper request BEFORE processing
		// Empty request usually means EOF was hit before any data was read
		if req.RequestLine.Method == "" || req.RequestLine.RequestTarget == "" {
			// This typically means the connection was closed or no data was available
			// In keep-alive, this shouldn't happen - treat as connection closed
			fmt.Println("Empty request received - connection likely closed or client didn't send next request")
			// Check if connection is still alive by trying to peek at it
			// If we can't read, the connection is definitely closed
			break
		}

		fmt.Printf("DEBUG: Parsed request - Method: '%s', Target: '%s', Version: '%s'\n",
			req.RequestLine.Method, req.RequestLine.RequestTarget, req.RequestLine.HttpVersion)

		fmt.Println("request received for endpoint: ", req.RequestLine.RequestTarget, ", Method: ", req.RequestLine.Method)

		// Check if client wants to close connection
		connectionHeader := strings.ToLower(req.Headers.Get("connection"))
		shouldClose := connectionHeader == "close"

		writer := response.NewResponseWriter(conn)

		// Use just the path part (without query string) for route matching
		path := req.Path()
		matchResult, err := s.handlers.MatchWithVars(path, handler.AllowedMethod(req.RequestLine.Method))
		if err == nil {
			// Populate path variables into the request
			maps.Copy(req.Vars, matchResult.Vars)
			s.executeMiddlewares(writer, req, matchResult)
		} else {
			if err.Error() == "Method not allowed" {
				body := respond405()
				writer.Respond(405, response.GetDefaultHeaders(len(body)), body)
			} else {
				s.notFound(writer, req)
			}
		}

		// If client wants to close, exit loop
		if shouldClose {
			break
		}

		// IMPORTANT: Reset the response writer state for the next request
		// This ensures we're ready to handle the next request on this connection
		// The connection itself stays open for keep-alive

		// Reset deadline for next request
		// This gives the client 60 seconds to send the next request
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}

	fmt.Println("Closing conn")

	conn.Close()
}

func (s *Server) Use(m middleware.MiddlewareHandler) {
	s.middleware = append(s.middleware, m)
}

func (s *Server) OverrideNotFoundHandler(notFoundHandler handler.HandlerFunc) {
	s.notFound = notFoundHandler
}

func (s *Server) executeMiddlewares(w *response.Writer, r *request.Request, next *handler.MatchResult) {
	middlewares := slices.Clone(s.middleware)

	slices.Reverse(middlewares)
	finalHandler := next.Handler.ExecuteMiddlewares(w, r, middleware.MiddlewareFunc(next.HandlerFunc))

	for _, m := range middlewares {
		finalHandler = m(finalHandler)
	}

	finalHandler(w, r)
}

func respond405() []byte {
	return []byte(`<html>
  <head>
    <title>405 Method Not Allowed</title>
  </head>
  <body>
    <h1>Method Not Allowed</h1>
    <p>That method is not allowed for this endpoint</p>
  </body>
</html>`)
}

func defaultNotFoundHandler(w *response.Writer, req *request.Request) {
	h := headers.NewHeaders()
	w.Respond(404, h, respond404())
}

func respond404() []byte {
	return []byte(`<html>
  <head>
    <title>404 Not Found</title>
  </head>
  <body>
    <h1>Not Found</h1>
    <p>Could not find what you are looking for.</p>
  </body>
</html>`)
}
