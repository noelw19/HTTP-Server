package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// readFullHTTPResponse reads a complete HTTP response from the connection
func readFullHTTPResponse(conn net.Conn, timeout time.Duration) (string, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))

	reader := bufio.NewReader(conn)

	// Read status line
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read status line: %v", err)
	}

	response := statusLine

	// Read headers
	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read headers: %v", err)
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
			return "", fmt.Errorf("failed to read body: %v (read %d/%d bytes)", err, n, contentLength)
		}
		response += string(body)
	}

	return response, nil
}

func main() {
	fmt.Println("Testing Keep-Alive Connection...")
	fmt.Println("Connecting to localhost:42069...")

	// Connect to server
	conn, err := net.Dial("tcp", "localhost:42069")
	if err != nil {
		panic(fmt.Sprintf("Failed to connect: %v", err))
	}
	defer conn.Close()

	fmt.Println("✅ Connected! Sending first request...\n")

	// First request
	request1 := "GET /test HTTP/1.1\r\n" +
		"Host: localhost:42069\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n"

	_, err = conn.Write([]byte(request1))
	if err != nil {
		panic(fmt.Sprintf("Failed to write request 1: %v", err))
	}

	// Read first response (complete HTTP response)
	response1, err := readFullHTTPResponse(conn, 5*time.Second)
	if err != nil {
		panic(fmt.Sprintf("Failed to read response 1: %v", err))
	}
	fmt.Printf("📥 Response 1 (%d bytes):\n%s\n", len(response1), response1)

	// Wait a bit
	fmt.Println("\n⏳ Waiting 2 seconds before second request...")
	time.Sleep(2 * time.Second)

	// Second request on SAME connection
	fmt.Println("📤 Sending second request on same connection...\n")
	request2 := "GET /test HTTP/1.1\r\n" +
		"Host: localhost:42069\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n"

	_, err = conn.Write([]byte(request2))
	if err != nil {
		panic(fmt.Sprintf("Failed to write request 2: %v", err))
	}

	// Read second response (complete HTTP response)
	response2, err := readFullHTTPResponse(conn, 5*time.Second)
	if err != nil {
		panic(fmt.Sprintf("Failed to read response 2: %v", err))
	}
	fmt.Printf("📥 Response 2 (%d bytes):\n%s\n", len(response2), response2)

	fmt.Println("\n✅ SUCCESS! Keep-alive is working!")
	fmt.Println("   Both requests used the same connection.")
	fmt.Println("   Check your server logs - you should see:")
	fmt.Println("   - Two requests processed")
	fmt.Println("   - Only ONE 'Closing conn' message at the end")
}
