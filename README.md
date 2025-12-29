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
3. **Add global middleware** (optional): `server.Use(middleware)` - Applies to all routes
4. **Add routes**: `server.AddHandler(path, handler).GET()`
5. **Add route-specific middleware** (optional): `server.AddHandler(path, handler).Use(middleware).GET()`
6. **Start listening**: `server.Listen()`
7. **Handle requests**: Write responses using `response.Writer`

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

- **`Use(m middleware.MiddlewareHandler)`**
  
  Registers global middleware that applies to all routes. Middleware executes in the order they are added.
  
  ```go
  server.Use(loggingMiddleware)
  server.Use(authMiddleware)
  ```
  
  - **Parameters**:
    - `m`: Middleware function that wraps the next handler

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
- **`Use(m middleware.MiddlewareHandler) *Handler`** - Adds route-specific middleware. Returns `*Handler` for chaining.

**Example**:
```go
server.AddHandler("/api/users", handler).GET()
server.AddHandler("/api/users", createHandler).POST()

// With route-specific middleware
server.AddHandler("/api/protected", handler).
    Use(authMiddleware).
    Use(loggingMiddleware).
    GET()
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

### Global Middleware

Global middleware applies to all routes in the order they are registered. A middleware function takes the next handler in the chain and returns a wrapped handler.

```go
import (
    "fmt"
    "github.com/noelw19/tcptohttp/internal/middleware.go"
    "github.com/noelw19/tcptohttp/internal/request"
    "github.com/noelw19/tcptohttp/internal/response"
    "github.com/noelw19/tcptohttp/internal/server"
)

func main() {
    srv := server.Serve(8080)
    
    // Add global middleware - executes for ALL routes
    // The function signature matches middleware.MiddlewareHandler:
    // func(next MiddlewareFunc) MiddlewareFunc
    srv.Use(func(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
        return func(w *response.Writer, req *request.Request) {
            fmt.Println("Global middleware 1 - before")
            next(w, req)  // Continue to next middleware/handler
            fmt.Println("Global middleware 1 - after")
        }
    })
    
    srv.Use(func(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
        return func(w *response.Writer, req *request.Request) {
            fmt.Println("Global middleware 2 - before")
            next(w, req)
            fmt.Println("Global middleware 2 - after")
        }
    })
    
    // Routes will execute: Global 1 → Global 2 → Handler
    srv.AddHandler("/", homeHandler).GET()
    srv.Listen()
}
```

**Execution order for a request:**
1. Global middleware 1 (before)
2. Global middleware 2 (before)
3. Route handler
4. Global middleware 2 (after)
5. Global middleware 1 (after)

### Route-Specific Middleware

Route-specific middleware applies only to specific routes and executes after global middleware.

```go
func main() {
    srv := server.Serve(8080)
    
    // Global middleware
    srv.Use(loggingMiddleware)
    
    // Public route - only global middleware
    srv.AddHandler("/public", publicHandler).GET()
    
    // Protected route - global + route-specific middleware
    srv.AddHandler("/api/users", userHandler).
        Use(authMiddleware).      // Route-specific: authentication
        Use(rateLimitMiddleware). // Route-specific: rate limiting
        GET()
    
    // Another protected route with different middleware
    srv.AddHandler("/api/admin", adminHandler).
        Use(authMiddleware).
        Use(adminOnlyMiddleware). // Different middleware for admin
        GET()
}
```

**Execution order for `/api/users`:**
1. Global logging middleware (before)
2. Route auth middleware (before)
3. Route rate limit middleware (before)
4. Handler
5. Route rate limit middleware (after)
6. Route auth middleware (after)
7. Global logging middleware (after)

### Common Middleware Patterns

#### Logging Middleware

```go
func loggingMiddleware(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
    return func(w *response.Writer, req *request.Request) {
        start := time.Now()
        method := req.RequestLine.Method
        path := req.RequestLine.RequestTarget
        
        fmt.Printf("[%s] %s %s - Started\n", time.Now().Format(time.RFC3339), method, path)
        
        next(w, req)  // Execute handler
        
        duration := time.Since(start)
        fmt.Printf("[%s] %s %s - Completed in %v\n", 
            time.Now().Format(time.RFC3339), method, path, duration)
    }
}

// Usage
srv.Use(loggingMiddleware)
```

#### Authentication Middleware

```go
func authMiddleware(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
    return func(w *response.Writer, req *request.Request) {
        token := req.Headers.Get("authorization")
        
        if token == "" || !isValidToken(token) {
            body := []byte(`{"error": "Unauthorized"}`)
            headers := response.GetDefaultHeaders(len(body))
            headers.Replace("content-type", "application/json")
            w.Respond(401, headers, body)
            return  // Don't call next() - short-circuit the request
        }
        
        // Token is valid, continue to handler
        next(w, req)
    }
}

// Usage - only on protected routes
srv.AddHandler("/api/protected", protectedHandler).
    Use(authMiddleware).
    GET()
```

#### CORS Middleware

```go
func corsMiddleware(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
    return func(w *response.Writer, req *request.Request) {
        // Handle preflight requests
        if req.RequestLine.Method == "OPTIONS" {
            headers := response.GetDefaultHeaders(0)
            headers.Set("Access-Control-Allow-Origin", "*")
            headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
            headers.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            w.Respond(200, headers, []byte{})
            return
        }
        
        // Execute handler
        next(w, req)
        
        // Add CORS headers to response (if response writer supports modification)
        // Note: This is a simplified example
    }
}

// Usage
srv.Use(corsMiddleware)
```

#### Request ID Middleware

```go
import (
    "crypto/rand"
    "encoding/hex"
)

func requestIDMiddleware(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
    return func(w *response.Writer, req *request.Request) {
        // Generate unique request ID
        id := make([]byte, 16)
        rand.Read(id)
        requestID := hex.EncodeToString(id)
        
        // Store in request context (you'd need to add a context field to Request)
        // For now, we can add it as a header
        req.Headers.Set("X-Request-ID", requestID)
        
        // Add to response headers
        next(w, req)
        
        // Response headers would be set in the handler or another middleware
    }
}

// Usage
srv.Use(requestIDMiddleware)
```

#### Rate Limiting Middleware

```go
import (
    "sync"
    "time"
)

type rateLimiter struct {
    requests map[string][]time.Time
    mu       sync.Mutex
    limit    int
    window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
    return &rateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *rateLimiter) middleware(next middleware.MiddlewareFunc) middleware.MiddlewareFunc {
    return func(w *response.Writer, req *request.Request) {
        // Get client IP (simplified - you'd extract from connection)
        clientIP := req.Headers.Get("X-Forwarded-For")
        if clientIP == "" {
            clientIP = "unknown"
        }
        
        rl.mu.Lock()
        now := time.Now()
        
        // Clean old requests outside the window
        if times, ok := rl.requests[clientIP]; ok {
            valid := []time.Time{}
            for _, t := range times {
                if now.Sub(t) < rl.window {
                    valid = append(valid, t)
                }
            }
            rl.requests[clientIP] = valid
        }
        
        // Check limit
        if len(rl.requests[clientIP]) >= rl.limit {
            rl.mu.Unlock()
            body := []byte(`{"error": "Rate limit exceeded"}`)
            headers := response.GetDefaultHeaders(len(body))
            headers.Replace("content-type", "application/json")
            w.Respond(429, headers, body)
            return
        }
        
        // Add current request
        rl.requests[clientIP] = append(rl.requests[clientIP], now)
        rl.mu.Unlock()
        
        next(w, req)
    }
}

// Usage
limiter := newRateLimiter(100, time.Minute) // 100 requests per minute
srv.AddHandler("/api/endpoint", handler).
    Use(limiter.middleware).
    GET()
```

### Combining Global and Route-Specific Middleware

```go
func main() {
    srv := server.Serve(8080)
    
    // Global middleware - applies to all routes
    srv.Use(loggingMiddleware)
    srv.Use(requestIDMiddleware)
    
    // Public routes - only global middleware
    srv.AddHandler("/", homeHandler).GET()
    srv.AddHandler("/about", aboutHandler).GET()
    
    // Protected routes - global + authentication
    srv.AddHandler("/api/users", getUserHandler).
        Use(authMiddleware).
        GET()
    
    // Admin routes - global + auth + admin check
    srv.AddHandler("/api/admin", adminHandler).
        Use(authMiddleware).
        Use(adminOnlyMiddleware).
        GET()
    
    // Rate-limited API - global + rate limiting
    srv.AddHandler("/api/data", dataHandler).
        Use(rateLimitMiddleware).
        GET()
    
    srv.Listen()
}
```

**Execution flow for `/api/admin`:**
1. Request ID middleware (global)
2. Logging middleware (global)
3. Auth middleware (route-specific)
4. Admin-only middleware (route-specific)
5. Admin handler
6. Admin-only middleware (after)
7. Auth middleware (after)
8. Logging middleware (after)
9. Request ID middleware (after)

### Middleware Best Practices

1. **Always call `next()`** unless you're intentionally short-circuiting the request
2. **Order matters**: Middleware executes in registration order
3. **Global vs Route-specific**: Use global for cross-cutting concerns (logging, CORS), route-specific for conditional logic (auth, rate limiting)
4. **Error handling**: If middleware fails, return an error response and don't call `next()`
5. **Performance**: Keep middleware lightweight; expensive operations should be async or cached

---

## Improvement Tips

Here are some suggestions to enhance this HTTP server package:

### 1. **Connection Keep-Alive Support**
   - HTTP/1.1 supports persistent connections via the `Connection: keep-alive` header
   - Currently, the server closes connections after each request
   - Implement connection pooling and reuse for better performance

### 2. **Middleware Enhancements**
   - ✅ Basic middleware support is implemented
   - Consider adding middleware groups/prefixes
   - Consider adding conditional middleware execution
   - Consider adding middleware context/state passing

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
