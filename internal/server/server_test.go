package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/noelw19/tcptohttp/internal/request"
	"github.com/noelw19/tcptohttp/internal/response"
)

// readFullHTTPResponse reads a complete HTTP response from the connection
// It handles Content-Length and reads until the full response is received
func readFullHTTPResponse(conn net.Conn, timeout time.Duration) (string, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))

	reader := bufio.NewReader(conn)

	// Read status line
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read status line: %w", err)
	}

	response := statusLine

	// Read headers
	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read headers: %w", err)
		}

		response += line

		// Empty line indicates end of headers
		if line == "\r\n" || line == "\n" {
			break
		}

		// Parse Content-Length header
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				cl, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					contentLength = cl
				}
			}
		}
	}

	// Read body if Content-Length is specified
	if contentLength > 0 {
		body := make([]byte, contentLength)
		n, err := io.ReadFull(reader, body)
		if err != nil {
			return "", fmt.Errorf("failed to read body: %w (read %d/%d bytes)", err, n, contentLength)
		}
		response += string(body)
	}

	return response, nil
}

// TestKeepAlive tests that the server properly handles keep-alive connections
// by processing multiple requests on the same connection
func TestKeepAlive(t *testing.T) {
	// Create a test server on a random port (port 0 = OS chooses)
	testPort := 0
	srv := Serve(testPort)

	// Add a simple test handler
	srv.AddHandler("/test", func(w *response.Writer, req *request.Request) {
		body := []byte("test response")
		w.Respond(200, body)
	}).GET()

	// Start the server
	err := srv.Listen()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Close()

	// Wait a bit for server to start accepting connections
	time.Sleep(50 * time.Millisecond)

	// Get the actual port the server is listening on
	addr := srv.Listener.Addr().String()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("Failed to parse address: %v", err)
	}

	t.Logf("Server listening on port %s", port)

	// Connect to the server
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// First request
	request1 := "GET /test HTTP/1.1\r\n" +
		"Host: localhost:" + port + "\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n"

	_, err = conn.Write([]byte(request1))
	if err != nil {
		t.Fatalf("Failed to write first request: %v", err)
	}

	// Read first response (complete HTTP response)
	response1, err := readFullHTTPResponse(conn, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to read first response: %v", err)
	}

	t.Logf("First response received (%d bytes)", len(response1))

	// Verify first response
	if !strings.Contains(response1, "HTTP/1.1 200") {
		t.Errorf("Expected HTTP/1.1 200, got: %s", response1[:100])
	}
	if !strings.Contains(response1, "connection: keep-alive") {
		t.Error("Response should include 'Connection: keep-alive' header")
	}
	if !strings.Contains(response1, "test response") {
		t.Error("Response should include 'test response' body")
	}

	fmt.Println(response1)
	fmt.Println("--------------------------------")

	// Wait a bit before second request
	time.Sleep(100 * time.Millisecond)

	// Second request on the SAME connection
	request2 := "GET /test HTTP/1.1\r\n" +
		"Host: localhost:" + port + "\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n"

	_, err = conn.Write([]byte(request2))
	if err != nil {
		t.Fatalf("Failed to write second request: %v", err)
	}

	// Read second response (complete HTTP response)
	response2, err := readFullHTTPResponse(conn, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to read second response: %v", err)
	}

	t.Logf("Second response received (%d bytes)", len(response2))

	// Verify second response
	if !strings.Contains(response2, "HTTP/1.1 200") {
		t.Errorf("Expected HTTP/1.1 200, got: %s", response2[:100])
	}
	if !strings.Contains(response2, "test response") {
		t.Error("Second response should include 'test response' body")
	}

	fmt.Println(response2)
	fmt.Println("--------------------------------")

	// Both requests were processed on the same connection
	t.Log("✅ Keep-alive test passed: Both requests processed on same connection")
}

// TestKeepAliveConnectionClose tests that the server respects Connection: close header
func TestKeepAliveConnectionClose(t *testing.T) {
	testPort := 0
	srv := Serve(testPort)

	srv.AddHandler("/test", func(w *response.Writer, req *request.Request) {
		body := []byte("test")
		w.Respond(200, body)
	}).GET()

	err := srv.Listen()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Close()

	time.Sleep(50 * time.Millisecond)

	addr := srv.Listener.Addr().String()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("Failed to parse address: %v", err)
	}

	// Connect and send request with Connection: close
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	request := "GET /test HTTP/1.1\r\n" +
		"Host: localhost:" + port + "\r\n" +
		"Connection: close\r\n" +
		"\r\n"

	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// Read response
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	response := string(buffer[:n])
	if !strings.Contains(response, "HTTP/1.1 200") {
		t.Errorf("Expected HTTP/1.1 200, got: %s", response[:100])
	}

	// Try to send another request - should fail because connection should be closed
	// Set a short deadline to detect if connection is closed
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	request2 := "GET /test HTTP/1.1\r\n" +
		"Host: localhost:" + port + "\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n"

	_, err = conn.Write([]byte(request2))
	if err == nil {
		// Try to read - should timeout or get error
		_, err = conn.Read(buffer)
		if err == nil {
			t.Error("Connection should have been closed after Connection: close, but second request succeeded")
		}
	}

	t.Log("✅ Connection: close test passed")
}

// TestKeepAliveMultipleRequests tests handling many requests on the same connection
func TestKeepAliveMultipleRequests(t *testing.T) {
	testPort := 0
	srv := Serve(testPort)

	requestCount := 0
	srv.AddHandler("/test", func(w *response.Writer, req *request.Request) {
		requestCount++
		body := []byte("test response")
		w.Respond(200, body)
	}).GET()

	err := srv.Listen()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Close()

	time.Sleep(50 * time.Millisecond)

	addr := srv.Listener.Addr().String()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("Failed to parse address: %v", err)
	}

	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send 5 requests on the same connection
	for i := 1; i <= 5; i++ {
		request := "GET /test HTTP/1.1\r\n" +
			"Host: localhost:" + port + "\r\n" +
			"Connection: keep-alive\r\n" +
			"\r\n"

		_, err = conn.Write([]byte(request))
		if err != nil {
			t.Fatalf("Failed to write request %d: %v", i, err)
		}

		// Read complete HTTP response
		response, err := readFullHTTPResponse(conn, 5*time.Second)
		if err != nil {
			t.Fatalf("Failed to read response %d: %v", i, err)
		}

		if !strings.Contains(response, "HTTP/1.1 200") {
			t.Errorf("Request %d: Expected HTTP/1.1 200", i)
		}
		if !strings.Contains(response, "test response") {
			t.Errorf("Request %d: Response should include 'test response' body", i)
		}

		time.Sleep(50 * time.Millisecond)
	}

	if requestCount != 5 {
		t.Errorf("Expected 5 requests to be processed, got %d", requestCount)
	}

	t.Logf("✅ Multiple requests test passed: %d requests processed on same connection", requestCount)
}
