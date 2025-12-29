# Testing Keep-Alive Connections

## Current Issues

The current implementation has problems that prevent keep-alive from working:

1. **Connection is always closed**: `defer conn.Close()` on line 105 closes the connection after each request
2. **Test code in production**: Lines 140-141 have `time.Sleep` and `tcp.Close()` that shouldn't be there
3. **No request loop**: The server doesn't loop to handle multiple requests on the same connection
4. **Missing Connection header check**: The server doesn't check if the client wants keep-alive

## How to Test Keep-Alive

### Method 1: Using curl (Simple)

Test if the connection stays open for multiple requests:

```bash
# Start your server first
go run cmd/httpserver/main.go

# In another terminal, test with curl
# The -v flag shows verbose output including connection reuse
curl -v http://localhost:42069/test

# Make multiple requests - if keep-alive works, you'll see "Re-using existing connection"
curl -v http://localhost:42069/test
curl -v http://localhost:42069/test
```

**What to look for:**
- `Connection: keep-alive` in the response headers
- `Re-using existing connection` message in curl output
- Same connection being reused for multiple requests

### Method 2: Using curl with Connection Header

Explicitly request keep-alive:

```bash
# Request with keep-alive header
curl -v -H "Connection: keep-alive" http://localhost:42069/test

# Make another request on the same connection
curl -v -H "Connection: keep-alive" http://localhost:42069/test
```

### Method 3: Using netcat/telnet (Manual)

Test the raw HTTP connection:

```bash
# Connect to server
nc localhost 42069

# Send first request
GET /test HTTP/1.1
Host: localhost:42069
Connection: keep-alive


# You should get a response. If keep-alive works, the connection stays open.
# Send another request without reconnecting:
GET /test HTTP/1.1
Host: localhost:42069
Connection: keep-alive


# The connection should still be open and handle the second request
```

### Method 4: Using a Go Test Client

Create a test program to verify connection reuse:

```go
package main

import (
    "fmt"
    "io"
    "net"
    "time"
)

func main() {
    conn, err := net.Dial("tcp", "localhost:42069")
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    
    // Send first request
    request1 := "GET /test HTTP/1.1\r\nHost: localhost:42069\r\nConnection: keep-alive\r\n\r\n"
    conn.Write([]byte(request1))
    
    // Read response
    buffer := make([]byte, 4096)
    n, _ := conn.Read(buffer)
    fmt.Printf("Response 1:\n%s\n", string(buffer[:n]))
    
    // Wait a bit
    time.Sleep(1 * time.Second)
    
    // Send second request on SAME connection
    request2 := "GET /test HTTP/1.1\r\nHost: localhost:42069\r\nConnection: keep-alive\r\n\r\n"
    conn.Write([]byte(request2))
    
    // Read second response
    buffer = make([]byte, 4096)
    n, _ = conn.Read(buffer)
    fmt.Printf("Response 2:\n%s\n", string(buffer[:n]))
    
    fmt.Println("If you see both responses, keep-alive is working!")
}
```

### Method 5: Using Apache Bench (ab)

Test connection reuse with multiple requests:

```bash
# Install ab if needed: brew install httpd (macOS) or apt-get install apache2-utils (Linux)

# Test with keep-alive (default)
ab -n 100 -c 10 http://localhost:42069/test

# Test without keep-alive for comparison
ab -n 100 -c 10 -H "Connection: close" http://localhost:42069/test
```

**What to compare:**
- With keep-alive: Faster total time, fewer connection establishments
- Without keep-alive: Slower, new connection for each request

### Method 6: Monitor Network Connections

Use system tools to see if connections stay open:

**On macOS/Linux:**
```bash
# Watch connections to your server
watch -n 1 'netstat -an | grep 42069'

# Or use lsof
lsof -i :42069

# Or use ss (Linux)
ss -tn | grep 42069
```

**What to look for:**
- Connection state should be `ESTABLISHED` and stay that way between requests
- With keep-alive: One connection handles multiple requests
- Without keep-alive: New connection for each request

### Method 7: Check Response Headers

Verify the server sends the correct headers:

```bash
curl -I http://localhost:42069/test
```

**Expected output:**
```
HTTP/1.1 200 OK
Connection: keep-alive
Content-Length: ...
Content-Type: ...
```

## Expected Behavior

### With Proper Keep-Alive Implementation:

1. **First request:**
   - Client sends: `Connection: keep-alive`
   - Server responds: `Connection: keep-alive`
   - Connection stays open

2. **Subsequent requests:**
   - Client reuses the same connection
   - Server handles multiple requests on one connection
   - Connection closes only when:
     - Client sends `Connection: close`
     - Connection times out
     - Server explicitly closes it

3. **Performance:**
   - Faster response times (no connection setup overhead)
   - Lower resource usage (fewer connections)
   - Better throughput

### Current Behavior (With Issues):

1. Connection closes after each request (due to `defer conn.Close()`)
2. Each request requires a new connection
3. No connection reuse
4. Slower performance

## Quick Test Script

Save this as `test_keepalive.sh`:

```bash
#!/bin/bash

echo "Testing Keep-Alive on http://localhost:42069/test"
echo ""

echo "Request 1:"
curl -v -H "Connection: keep-alive" http://localhost:42069/test 2>&1 | grep -i "connection\|re-using"

echo ""
echo "Request 2 (should reuse connection):"
curl -v -H "Connection: keep-alive" http://localhost:42069/test 2>&1 | grep -i "connection\|re-using"

echo ""
echo "If you see 'Re-using existing connection' in request 2, keep-alive is working!"
```

Run with: `chmod +x test_keepalive.sh && ./test_keepalive.sh`

## What Needs to be Fixed

For keep-alive to work properly, the server needs:

1. **Remove `defer conn.Close()`** - or make it conditional
2. **Add request loop** - handle multiple requests on one connection
3. **Check Connection header** - respect client's keep-alive preference
4. **Remove test code** - lines 140-141 with `time.Sleep` and `tcp.Close()`
5. **Set Connection header** - already done in `GetDefaultHeaders()`, but should be conditional
6. **Handle connection timeout** - close idle connections after a timeout
7. **Handle Connection: close** - close when client requests it

