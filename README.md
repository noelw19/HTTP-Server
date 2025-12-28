# HTTP Server from Scratch

A simple HTTP/1.1 server implementation built from scratch in Go. This server implements the HTTP/1.1 protocol (RFC 7230-7237) with support for routing, dynamic path parameters, query strings, and streaming responses.

## Simple Implementation

### Quick Start

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/noelw19/tcptohttp/internal/server"
    "github.com/noelw19/tcptohttp/internal/request"
    "github.com/noelw19/tcptohttp/internal/response"
)

const port = 8080

func main() {
    // Create server (default 404 handler is set automatically)
    srv := server.Serve(port)
    
    // Optionally override the default 404 handler
    srv.OverrideNotFoundHandler(notFound)
    
    // Register routes
    srv.AddHandler("/", home).GET()
    srv.AddHandler("/users/{id}", getUser).GET()
    srv.AddHandler("/users", createUser).POST()

    log.Printf("Server running on :%d", port)
    srv.Listen()

    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    srv.Close()
}

func home(w *response.Writer, req *request.Request) {
    body := []byte("Hello, World!")
    w.Respond(200, response.GetDefaultHeaders(len(body)), body)
}

func getUser(w *response.Writer, req *request.Request) {
    id := req.Vars["id"] // Path parameter
    body := []byte("User: " + id)
    w.Respond(200, response.GetDefaultHeaders(len(body)), body)
}

func createUser(w *response.Writer, req *request.Request) {
    // req.Body contains the request body
    body := []byte(`{"status": "created"}`)
    headers := response.GetDefaultHeaders(len(body))
    headers.Replace("content-type", "application/json")
    w.Respond(201, headers, body)
}

func notFound(w *response.Writer, req *request.Request) {
    body := []byte("404 Not Found")
    w.Respond(404, response.GetDefaultHeaders(len(body)), body)
}
```

### Basic Usage

1. **Create a server**: `server.Serve(port)` - Returns a `*Server` instance with a default 404 handler
2. **Optionally override 404 handler**: `server.OverrideNotFoundHandler(handler)`
3. **Add routes**: `server.AddHandler(path, handler).GET()`
4. **Start listening**: `server.Listen()`
5. **Handle requests**: Write responses using `response.Writer`

---

## API Documentation

### Package: `server`

#### `func Serve(port int) *Server`

Creates a new HTTP server instance with a default 404 handler.

- **Parameters**:
  - `port`: Port number to listen on
- **Returns**: `*Server` instance

The server is created with a default 404 handler. Use `OverrideNotFoundHandler()` to customize it.

#### `type Server struct`

The main server struct.

**Methods**:

- **`AddHandler(route string, handleFunc handler.HandlerFunc) *handler.Handler`**
  
  Registers a route handler. Returns a `*Handler` for method chaining.
  
  ```go
  server.AddHandler("/users", userHandler).GET()
  server.AddHandler("/users", createUserHandler).POST()
  ```
  
  - **Parameters**:
    - `route`: Route path (e.g., `/users` or `/users/{id}`)
    - `handleFunc`: Handler function
  - **Returns**: `*Handler` for method chaining

- **`Listen() error`**
  
  Starts the server and begins accepting connections. This is a non-blocking call that starts a goroutine.
  
  - **Returns**: Error if listener fails to start

- **`Close() error`**
  
  Closes the server listener and stops accepting new connections.
  
  - **Returns**: Error if close fails

- **`OverrideNotFoundHandler(notFoundHandler handler.HandlerFunc)`**
  
  Overrides the default 404 handler with a custom handler function.
  
  ```go
  srv := server.Serve(8080)
  srv.OverrideNotFoundHandler(customNotFound)
  ```
  
  - **Parameters**:
    - `notFoundHandler`: Handler function for 404 responses

- **`Show()`**
  
  Debug method that prints all registered routes.

#### Route Patterns

Routes support dynamic path parameters using `{name}` syntax:

- `/users/{id}` - Matches `/users/123`, extracts `id = "123"`
- `/posts/{postId}/comments/{commentId}` - Matches `/posts/5/comments/10`

Path variables are accessible via `req.Vars["name"]`.

---

### Package: `handler`

#### `type HandlerFunc func(w *response.Writer, req *request.Request)`

Function signature for all route handlers.

**Parameters**:
- `w *response.Writer`: Response writer for sending HTTP responses
- `req *request.Request`: Parsed HTTP request object

#### `type Handler struct`

Handler struct returned by `AddHandler()` for method chaining.

**Methods**:

- **`GET() *Handler`** - Registers handler for GET requests
- **`POST() *Handler`** - Registers handler for POST requests
- **`PATCH() *Handler`** - Registers handler for PATCH requests
- **`DELETE() *Handler`** - Registers handler for DELETE requests

**Example**:
```go
server.AddHandler("/api/users", handler).GET()
server.AddHandler("/api/users", createHandler).POST()
```

---

### Package: `request`

#### `type Request struct`

Represents a parsed HTTP request.

**Fields**:

- **`RequestLine RequestLine`** - HTTP request line (method, target, version)
  - `Method string` - HTTP method (GET, POST, etc.)
  - `RequestTarget string` - Full request target including query string
  - `HttpVersion string` - HTTP version (e.g., "1.1")

- **`Headers headers.Headers`** - HTTP headers map
  - Use `req.Headers.Get("header-name")` to read headers
  - Use `req.Headers.Set("header-name", "value")` to set headers

- **`Body []byte`** - Request body as byte slice
  - Access as `string(req.Body)` for text content

- **`Vars map[string]string`** - Path parameters from dynamic routes
  - Example: For route `/users/{id}`, `req.Vars["id"]` contains the value

- **`Params map[string]string`** - Query string parameters
  - Example: For `/search?q=golang&limit=10`, `req.Params["q"]` = "golang"

**Methods**:

- **`Path() string`** - Returns the path portion without query string

**Example**:
```go
func handler(w *response.Writer, req *request.Request) {
    // Path parameter
    userId := req.Vars["id"]
    
    // Query parameter
    query := req.Params["q"]
    
    // Request body
    bodyData := string(req.Body)
    
    // Headers
    userAgent := req.Headers.Get("user-agent")
}
```

---

### Package: `response`

#### `type Writer struct`

HTTP response writer that ensures proper HTTP/1.1 response formatting.

**Methods**:

- **`Respond(status StatusCode, headers headers.Headers, body []byte)`**
  
  Convenience method to send a complete HTTP response.
  
  ```go
  body := []byte("Hello")
  headers := response.GetDefaultHeaders(len(body))
  w.Respond(200, headers, body)
  ```
  
  - **Parameters**:
    - `status`: HTTP status code
    - `headers`: Response headers
    - `body`: Response body

- **`WriteStatusLine(status StatusCode) error`**
  
  Writes the HTTP status line (e.g., `HTTP/1.1 200 OK\r\n`).
  
  Must be called first, before headers or body.

- **`WriteHeaders(headers headers.Headers) error`**
  
  Writes HTTP headers. Must be called after `WriteStatusLine()`.

- **`WriteBody(body []byte) (int, error)`**
  
  Writes the response body. Must be called after `WriteHeaders()`.

**Manual Response Writing**:
```go
w.WriteStatusLine(200)
w.WriteHeaders(headers)
w.WriteBody(body)
```

#### `type StatusCode int`

HTTP status code constants:

- `response.StatusOK` (200)
- `response.StatusBadRequest` (400)
- `response.StatusInternalServerError` (500)

You can also use integer literals: `w.Respond(201, headers, body)`

#### `func GetDefaultHeaders(contentLen int) headers.Headers`

Creates default HTTP headers with:
- `Content-Length`: Set to `contentLen`
- `Connection`: `close`
- `Content-Type`: `text/plain`

**Example**:
```go
headers := response.GetDefaultHeaders(len(body))
headers.Replace("content-type", "application/json")
```

---

### Package: `headers`

#### `type Headers map[string]string`

HTTP headers map.

**Methods**:

- **`Get(key string) string`** - Get header value (case-insensitive)
- **`Set(key string, value string)`** - Set header value
- **`Replace(key string, value string)`** - Replace existing header or add new one
- **`HasContentLength() (int, bool)`** - Get Content-Length header value

**Example**:
```go
h := headers.NewHeaders()
h.Set("content-type", "application/json")
h.Set("x-custom-header", "value")
```

---

## Examples

### Dynamic Route Parameters

```go
// Route: /users/{id}/posts/{postId}
// Request: /users/123/posts/456
func handler(w *response.Writer, req *request.Request) {
    userId := req.Vars["id"]      // "123"
    postId := req.Vars["postId"]  // "456"
    
    body := []byte(fmt.Sprintf("User %s, Post %s", userId, postId))
    w.Respond(200, response.GetDefaultHeaders(len(body)), body)
}

server.AddHandler("/users/{id}/posts/{postId}", handler).GET()
```

### Query String Parameters

```go
// Request: /search?q=golang&limit=10&page=1
func handler(w *response.Writer, req *request.Request) {
    query := req.Params["q"]      // "golang"
    limit := req.Params["limit"]  // "10"
    page := req.Params["page"]    // "1"
    
    result := fmt.Sprintf("Query: %s, Limit: %s, Page: %s", query, limit, page)
    body := []byte(result)
    w.Respond(200, response.GetDefaultHeaders(len(body)), body)
}

server.AddHandler("/search", handler).GET()
```

### POST Request with JSON

```go
func createUser(w *response.Writer, req *request.Request) {
    // Parse JSON from req.Body
    var userData map[string]interface{}
    json.Unmarshal(req.Body, &userData)
    
    // Create response
    responseBody := []byte(`{"status": "created", "id": 123}`)
    headers := response.GetDefaultHeaders(len(responseBody))
    headers.Replace("content-type", "application/json")
    w.Respond(201, headers, responseBody)
}

server.AddHandler("/users", createUser).POST()
```

### Custom Headers

```go
func handler(w *response.Writer, req *request.Request) {
    body := []byte("Response")
    headers := response.GetDefaultHeaders(len(body))
    headers.Replace("content-type", "application/json")
    headers.Set("x-api-version", "1.0")
    headers.Set("cache-control", "no-cache")
    w.Respond(200, headers, body)
}
```

### Reading Request Headers

```go
func handler(w *response.Writer, req *request.Request) {
    userAgent := req.Headers.Get("user-agent")
    contentType := req.Headers.Get("content-type")
    auth := req.Headers.Get("authorization")
    
    // Use headers...
    body := []byte("OK")
    w.Respond(200, response.GetDefaultHeaders(len(body)), body)
}
```

### Streaming Response

```go
import "github.com/noelw19/tcptohttp/internal/stream"

func streamHandler(w *response.Writer, req *request.Request) {
    file, err := os.Open("large-file.txt")
    if err != nil {
        body := []byte("Error")
        w.Respond(500, response.GetDefaultHeaders(len(body)), body)
        return
    }
    defer file.Close()
    
    headers := headers.NewHeaders()
    headers.Set("content-type", "text/plain")
    stream.Streamer(w, headers, file)
}

server.AddHandler("/stream", streamHandler)
```

---

## Improvement Tips

Here are some suggestions to enhance this HTTP server package:

### 1. **Connection Keep-Alive Support**
   - HTTP/1.1 supports persistent connections via the `Connection: keep-alive` header
   - Currently, the server closes connections after each request
   - Implement connection pooling and reuse for better performance

### 2. **Middleware Support**
   - Add middleware chain support (logging, authentication, CORS, etc.)
   - Example: `server.Use(loggingMiddleware, authMiddleware)`

### 3. **Request Timeout Handling**
   - Add configurable read/write timeouts
   - Prevent slowloris attacks and resource exhaustion

### 4. **Better Error Handling**
   - More comprehensive HTTP status code support (currently limited)
   - Standardized error response format
   - Better error messages and logging

### 5. **Request Size Limits**
   - Add configurable maximum request body size
   - Prevent memory exhaustion from large uploads

### 6. **Security Headers**
   - Add support for security headers (X-Frame-Options, CSP, etc.)
   - HTTPS/TLS support
   - Input validation and sanitization

### 7. **CORS Support**
   - Built-in CORS middleware
   - Configurable allowed origins, methods, and headers

### 8. **Content Negotiation**
   - Support for Accept headers
   - Automatic content-type negotiation
   - Support for different response formats (JSON, XML, etc.)

### 9. **Compression Support**
   - Gzip/deflate compression for responses
   - Automatic compression based on Accept-Encoding header

### 10. **Better Logging**
   - Structured logging (log levels, request IDs)
   - Access logs with request/response details
   - Configurable log format

### 11. **Graceful Shutdown Improvements**
   - Wait for in-flight requests to complete
   - Configurable shutdown timeout
   - Better connection cleanup

### 12. **Route Grouping**
   - Support for route prefixes and groups
   - Example: `server.Group("/api/v1").AddHandler("/users", ...)`

### 13. **Request Context**
   - Add request context with cancellation support
   - Timeout propagation
   - Request-scoped values

### 14. **Static File Serving**
   - Built-in static file server
   - Directory listing support
   - MIME type detection

### 15. **HTTP/2 Support**
   - Upgrade path to HTTP/2
   - Server push capabilities

### 16. **Testing Utilities**
   - Test helpers for making requests
   - Mock request/response builders
   - Integration test utilities

### 17. **Configuration Options**
   - Server configuration struct
   - Environment variable support
   - Config file support

### 18. **Better HTTP Status Code Support**
   - Complete status code enum
   - Standard reason phrases
   - Proper status code handling

### 19. **Request Validation**
   - Built-in validation helpers
   - Schema validation for JSON bodies
   - Parameter validation

### 20. **Documentation**
   - API documentation generation
   - OpenAPI/Swagger support
   - Code examples and tutorials
