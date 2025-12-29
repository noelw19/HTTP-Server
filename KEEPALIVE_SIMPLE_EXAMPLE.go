// SIMPLE EXAMPLE: Key changes needed in your handle() function
// This shows the minimal changes to make keep-alive work

package server

import (
	"errors"
	"io"
	"maps"
	"net"
	"strings"
	"time"

	"github.com/noelw19/tcptohttp/internal/handler"
	"github.com/noelw19/tcptohttp/internal/request"
	"github.com/noelw19/tcptohttp/internal/response"
)

// BEFORE (your current code - doesn't work):
/*
func (s *Server) handle(conn net.Conn) {
	defer conn.Close()  // ❌ This closes connection immediately

	req, err := request.RequestFromReader(conn)
	// ... handle request ...

	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(5 * time.Second)
		s.handle(conn)  // ❌ Recursive call on closed connection
	}
}
*/

// AFTER (correct implementation):
func (s *Server) handle(conn net.Conn) {
	// ✅ Set keep-alive ONCE when connection is first accepted
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(30 * time.Second)
	}

	// ✅ Set read deadline to detect closed connections
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// ✅ Use a LOOP instead of recursion
	for {
		req, err := request.RequestFromReader(conn)

		// ✅ Handle errors - client may have closed connection
		if err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				break // Client closed, exit loop
			}
			// Send error and close
			hErr := &HandlerError{
				StatusCode: int(response.StatusBadRequest),
				Message:    err.Error(),
			}
			hErr.Write(conn)
			break
		}

		// ✅ Check if client wants to close connection
		connectionHeader := strings.ToLower(req.Headers.Get("connection"))
		if connectionHeader == "close" {
			break // Client wants to close, exit loop
		}

		// ... your existing request handling code ...
		writer := response.NewResponseWriter(conn)
		path := req.Path()
		matchResult, err := s.handlers.MatchWithVars(path, handler.AllowedMethod(req.RequestLine.Method))

		if err == nil {
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

		// ✅ Reset deadline for next request
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}

	// ✅ Close connection when loop exits
	conn.Close()
}

// KEY CHANGES SUMMARY:
//
// 1. REMOVE: defer conn.Close() at the start
// 2. ADD: Set keep-alive at the START of the function
// 3. CHANGE: Use a for loop instead of recursion
// 4. ADD: Check Connection header to respect client's wishes
// 5. ADD: Handle errors (EOF, closed connections)
// 6. ADD: Set/Reset read deadline in the loop
// 7. ADD: conn.Close() at the END when loop exits
