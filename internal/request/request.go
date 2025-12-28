package request

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/noelw19/tcptohttp/internal/headers"
)

type parserState string

const (
	parserInit    parserState = "initialised"
	parserDone    parserState = "done"
	parserHeaders parserState = "parsingHeaders"
	parserBody    parserState = "parsingBody"
)

type Request struct {
	RequestLine RequestLine
	state       parserState
	Headers     headers.Headers
	Body        []byte
	Vars        map[string]string // Path parameters from dynamic routes
	Params      map[string]string // Query string parameters
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

var ErrBadStartLine = fmt.Errorf("bad start line")
var SEPARATOR = []byte("\r\n")

func newRequest() *Request {
	return &Request{
		state:   parserInit,
		Headers: headers.NewHeaders(),
		Vars:    make(map[string]string),
		Params:  make(map[string]string),
	}
}

func parseRequestLine(req []byte) (*RequestLine, int, error) {
	idx := bytes.Index(req, SEPARATOR)
	if idx == -1 {
		return nil, 0, nil
	}

	startLine := req[:idx]
	read := idx + len(SEPARATOR)

	parts := bytes.Split(startLine, []byte(" "))
	if len(parts) != 3 {
		return nil, len(startLine), ErrBadStartLine
	}

	method := parts[0]
	target := parts[1]

	httpParts := bytes.Split(parts[2], []byte("/"))
	capMethod := strings.ToUpper(string(method))
	if string(method) != capMethod && string(httpParts[0]) != "HTTP" && string(httpParts[1]) != "1.1" {
		return nil, read, ErrBadStartLine
	}

	return &RequestLine{
		Method:        string(method),
		RequestTarget: string(target),
		HttpVersion:   string(httpParts[1]),
	}, read, nil
}

// parseParams extracts query string parameters from the RequestTarget
// and stores them in r.Params
func (r *Request) parseParams() {
	target := r.RequestLine.RequestTarget
	
	// Split path and query string (separated by ?)
	parts := strings.SplitN(target, "?", 2)
	if len(parts) < 2 {
		// No query string
		return
	}
	
	queryString := parts[1]
	if queryString == "" {
		return
	}
	
	// Parse query string using net/url
	values, err := url.ParseQuery(queryString)
	if err != nil {
		// If parsing fails, just return (don't break the request)
		return
	}
	
	// Store parameters in the Params map
	// If a parameter appears multiple times, we'll use the last value
	for key, val := range values {
		if len(val) > 0 {
			r.Params[key] = val[len(val)-1]
		}
	}
}

func (r *Request) parseBody(data []byte) (int, error) {
	cl := r.Headers.Get("content-length")
	if cl == "" {
		r.state = parserDone
	}

	clength, ok := r.Headers.HasContentLength()
	if !ok {
		return 0, nil
	}

	if clength != len(data) {
		return 0, fmt.Errorf("content length and body length mismatch")
	}

	r.Body = data
	return len(data), nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {

	bufferSize := 1024
	buffer := make([]byte, bufferSize)
	idx := 0

	request := newRequest()

	for !request.done() {

		n, err := reader.Read(buffer[idx:])
		if err == io.EOF {
			request.state = parserDone
		} else if err != nil {
			return nil, err
		}

		idx += n
		readN, err := request.parse(buffer[:idx])
		if err != nil {
			return nil, err
		}

		copy(buffer, buffer[readN:idx])
		idx -= readN

	}

	return request, nil
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
outer:
	for {
		switch r.state {
		case parserInit:
			rl, n, err := parseRequestLine(data[read:])
			if err != nil {
				return 0, err
			}

			if n == 0 {
				break outer
			}

			r.RequestLine = *rl
			read += n
			
			// Parse query string parameters
			r.parseParams()

			r.state = parserHeaders

		case parserHeaders:
			n, done, err := r.Headers.Parse(data[read:])
			if err != nil {
				return read, err
			}

			if n == 0 {
				break outer
			}

			read += n

			if done {
				r.state = parserBody
			}
		case parserBody:
			n, err := r.parseBody(data[read:])
			if err != nil {
				return read, err
			}

			if n == 0 {
				break outer
			}

			r.state = parserDone

		case parserDone:
			break outer
		}
	}
	return read, nil
}

func (r *Request) done() bool {
	return r.state == parserDone
}

// Path returns just the path portion of the RequestTarget, without the query string
func (r *Request) Path() string {
	target := r.RequestLine.RequestTarget
	// Split path and query string (separated by ?)
	parts := strings.SplitN(target, "?", 2)
	return parts[0]
}
