package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/noelw19/tcptohttp/internal/handler"
	"github.com/noelw19/tcptohttp/internal/headers"
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
	Listener net.Listener
	port     int
	running  bool
	notFound handler.HandlerFunc
	handlers *handler.Handlers
}

func (s *Server) Show() {
	for r := range *s.handlers {
		fmt.Printf("%+v\n", (*s.handlers)[r])

	}
}

func Serve(port int) (*Server) {
	server := &Server{
		port:     port,
		running:  false,
		handlers: &handler.Handlers{},
	}
	server.OverrideNotFoundHandler(defaultNotFoundHandler)

	return server
}

func (s *Server) OverrideNotFoundHandler(notFoundHandler handler.HandlerFunc) {
	s.notFound = notFoundHandler
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
	defer conn.Close()

	req, err := request.RequestFromReader(conn)
	if err != nil {
		hErr := &HandlerError{
			StatusCode: int(response.StatusBadRequest),
			Message:    err.Error(),
		}
		fmt.Println(err)
		hErr.Write(conn)
		return
	}
	fmt.Println("request received for endpoint: ", req.RequestLine.RequestTarget, ", Method: ", req.RequestLine.Method)

	writer := response.NewResponseWriter(conn)

	// Use just the path part (without query string) for route matching
	path := req.Path()
	matchResult, err := s.handlers.MatchWithVars(path, handler.AllowedMethod(req.RequestLine.Method))
	if err == nil {
		// Populate path variables into the request
		for key, value := range matchResult.Vars {
			req.Vars[key] = value
		}
		matchResult.Handler(writer, req)
	} else {
		if err.Error() == "Method not allowed" {
			body := respond405()
			writer.Respond(405, response.GetDefaultHeaders(len(body)), body)
		} else {
			s.notFound(writer, req)
		}
	}

	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.CloseWrite()
	}
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