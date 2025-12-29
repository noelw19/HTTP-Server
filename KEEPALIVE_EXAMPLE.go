// This is an example implementation of proper keep-alive handling
// Use this as a reference to implement keep-alive in your server.go file

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

// Example: Proper handle function with keep-alive support
func (s *Server) handleWithKeepAlive(conn net.Conn) {
	// Set keep-alive on the TCP connection when it's first accepted
	// This should be done ONCE when the connection is established
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(30 * time.Second) // Send keep-alive probes every 30 seconds
		// Set read deadline to detect closed connections
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}

	// Use a loop to handle multiple requests on the same connection
	// This is the key difference from the recursive approach
	for {
		// Try to read the next request
		req, err := request.RequestFromReader(conn)
		
		// Check if there was an error reading the request
		if err != nil {
			// If it's EOF or connection closed, the client disconnected
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				// Client closed the connection, exit the loop
				break
			}
			
			// For other errors, send error response and close
			hErr := &HandlerError{
				StatusCode: int(response.StatusBadRequest),
				Message:    err.Error(),
			}
			fmt.Println(err)
			hErr.Write(conn)
			break // Close connection on error
		}

		// Check if client wants to close the connection
		// Look for "Connection: close" header
		connectionHeader := strings.ToLower(req.Headers.Get("connection"))
		shouldClose := connectionHeader == "close"

		fmt.Println("request received for endpoint: ", req.RequestLine.RequestTarget, ", Method: ", req.RequestLine.Method)

		// Create a new response writer for this request
		writer := response.NewResponseWriter(conn)

		// Process the request
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

		// Check if we should close the connection
		if shouldClose {
			// Client requested connection close, exit the loop
			break
		}

		// Reset read deadline for the next request
		// This allows the connection to stay open and wait for the next request
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}

	// Close the connection when the loop exits
	// This happens when:
	// 1. Client sends "Connection: close"
	// 2. Error reading request (client disconnected)
	// 3. EOF (client closed connection)
	conn.Close()
}

// Alternative: More robust version with timeout handling
func (s *Server) handleWithKeepAliveRobust(conn net.Conn) {
	// Set keep-alive immediately when connection is accepted
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(30 * time.Second)
	}

	// Connection timeout - close idle connections after 60 seconds
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Loop to handle multiple requests
	for {
		req, err := request.RequestFromReader(conn)
		
		if err != nil {
			// Check for timeout
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Connection timed out (no data received for 60 seconds)
				// This is normal for keep-alive - just close it
				break
			}
			
			// Check for EOF or closed connection
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				// Client closed the connection
				break
			}
			
			// Other errors - send error response and close
			hErr := &HandlerError{
				StatusCode: int(response.StatusBadRequest),
				Message:    err.Error(),
			}
			fmt.Println("Error reading request:", err)
			hErr.Write(conn)
			break
		}

		// Check Connection header
		connectionHeader := strings.ToLower(req.Headers.Get("connection"))
		clientWantsKeepAlive := connectionHeader == "keep-alive" || connectionHeader == ""
		clientWantsClose := connectionHeader == "close"

		fmt.Println("request received for endpoint: ", req.RequestLine.RequestTarget, ", Method: ", req.RequestLine.Method)

		writer := response.NewResponseWriter(conn)

		// Process request
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

		// If client explicitly wants to close, do it
		if clientWantsClose {
			break
		}

		// If client doesn't want keep-alive (HTTP/1.0 or explicit close), close
		if !clientWantsKeepAlive && req.RequestLine.HttpVersion != "1.1" {
			break
		}

		// Reset deadline for next request
		// This allows the connection to wait for the next request
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}

	// Clean up: close the connection
	conn.Close()
}

// Example: How to modify the Listen() function to use keep-alive
func (s *Server) ListenWithKeepAlive() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	s.Listener = listener

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) || !s.running {
					break
				}
				if s.running {
					fmt.Println(err)
				}
				continue
			}

			s.running = true
			// Use the keep-alive handler instead of regular handle
			go s.handleWithKeepAlive(conn)
		}
	}()
	return nil
}

// Key Points to Remember:
//
// 1. SET KEEP-ALIVE EARLY: Set TCP keep-alive when connection is first accepted,
//    not after handling a request
//
// 2. USE A LOOP: Don't use recursion. Use a for loop to handle multiple requests
//    on the same connection
//
// 3. CHECK CONNECTION HEADER: Respect the client's "Connection: close" or
//    "Connection: keep-alive" header
//
// 4. HANDLE ERRORS: Check for EOF, closed connections, and timeouts
//
// 5. SET READ DEADLINE: Use SetReadDeadline to detect when client closes
//    connection or times out
//
// 6. CLOSE PROPERLY: Only close the connection when:
//    - Client sends "Connection: close"
//    - Error reading request (client disconnected)
//    - Timeout (no data for X seconds)
//    - EOF (client closed connection)
//
// 7. REMOVE DEFER: Don't use defer conn.Close() at the start - close it
//    explicitly when the loop exits

