package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/noelw19/tcptohttp/internal/headers"
	"github.com/noelw19/tcptohttp/internal/request"
	"github.com/noelw19/tcptohttp/internal/response"
	"github.com/noelw19/tcptohttp/internal/stream"
)

func wakandaHandler(w *response.Writer, req *request.Request) {
	w.WriteStatusLine(200)
	res := []byte("wakanda to you too")
	w.WriteHeaders(response.GetDefaultHeaders(len(res)))
	w.WriteBody(res)
}

func wakandaPOSTHandler(w *response.Writer, req *request.Request) {
	fmt.Println(string(req.Body))
	body := []byte("its working!!!!")
	w.Respond(200, response.GetDefaultHeaders(len(body)), body)
}

func wakandaIDHandler(w *response.Writer, req *request.Request) {
	// Access the dynamic route parameters
	id := req.Vars["id"]
	lala := req.Vars["lala"]

	// You can also access query string parameters
	// Example: /wakanda/123/abc?filter=active&sort=name
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Wakanda ID: %s, Lala: %s", id, lala))

	if len(req.Params) > 0 {
		result.WriteString("\nQuery params: ")
		var params []string
		for key, value := range req.Params {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
		result.WriteString(strings.Join(params, ", "))
	}

	body := []byte(result.String())
	w.Respond(200, response.GetDefaultHeaders(len(body)), body)
}

func queryHandler(w *response.Writer, req *request.Request) {
	// Access query string parameters
	// Example: /query?name=John&age=30
	var params []string
	for key, value := range req.Params {
		params = append(params, fmt.Sprintf("%s=%s", key, value))
	}

	queryStr := "No query parameters provided"
	if len(params) > 0 {
		queryStr = fmt.Sprintf("Query parameters: %s", strings.Join(params, ", "))
	}

	body := []byte(queryStr)
	w.Respond(200, response.GetDefaultHeaders(len(body)), body)
}

func streamHandler(w *response.Writer, req *request.Request) {

	target := req.RequestLine.RequestTarget
	var body []byte
	var status response.StatusCode
	h := response.GetDefaultHeaders(0)

	res, err := http.Get("https://httpbin.org/" + target[len("/httpbin/"):])
	if err != nil {
		body = respond500()
		status = response.StatusInternalServerError
		w.Respond(status, h, body)

		return
	}
	h.Replace("content-type", "text/plain")
	stream.Streamer(w, h, res.Body)
}

func videoHandler(w *response.Writer, req *request.Request) {
	f, err := os.Open("./assets/vim.mp4")
	if err != nil {
		h := headers.NewHeaders()
		body := respond500()
		w.Respond(response.StatusInternalServerError, h, body)
	} else {
		defer f.Close()
		h := headers.NewHeaders()
		h.Replace("content-type", "video/mp4")
		stream.Streamer(w, h, f)
	}
}

func respond400() []byte {
	return []byte(`<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>`)
}

func respond200() []byte {
	return []byte(`<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`)
}

func respond500() []byte {
	return []byte(`<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>`)
}
