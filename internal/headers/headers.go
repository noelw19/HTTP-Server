package headers

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Headers map[string]string

func NewHeaders() Headers {
	return map[string]string{}
}

var ErrInvalidHeader = fmt.Errorf("invalid header in request")

const CRLF = "\r\n"

var numberRegexp = regexp.MustCompile("^[a-zA-Z0-9!#$%&'*+-.^_|~`]+$")

func (h Headers) Get(key string) string {
	return h[strings.ToLower(key)]
}

func (h Headers) Set(key, value string) {
	if h.Get(key) == "" {
		h[strings.ToLower(key)] = value
		return
	}

	h[strings.ToLower(key)] = h[strings.ToLower(key)] + ", " + value
}

func (h Headers) Replace(key, value string) {
	h[strings.ToLower(key)] = value
}

func (h Headers) Delete(key string) {
	delete(h, strings.ToLower(key))
}

func (h Headers) HasContentLength() (int, bool) {
	cl := h.Get("content-length")
	te := h.Get("transfer-encoding")
	lengthInt, err := strconv.Atoi(cl)
	if err != nil {
		if te == "chunked" {
			return 0, true
		}
		return 0, false
	}
	return lengthInt, true
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	if !bytes.Contains(data, []byte(CRLF)) {
		return 0, false, nil
	}

	if string(data[:len(CRLF)]) == CRLF {
		return len(CRLF), true, nil
	}

	read := 0
	headers := bytes.Split(data, []byte(CRLF))

	header := headers[0]
	read += len(header) + len(CRLF)

	before, after, ok := bytes.Cut(header, []byte(":"))
	if !ok {
		return read, false, ErrInvalidHeader
	}

	key := string(before)
	value := string(after)

	if !numberRegexp.Match(before) {
		fmt.Println("includes invalid")
		return 0, false, ErrInvalidHeader
	}

	if string(key[len(key)-1]) == " " {
		return 0, false, ErrInvalidHeader
	}

	key = strings.ToLower(strings.Trim(key, " "))
	value = strings.Trim(value, " ")

	if _, ok := h[key]; ok {
		h.Set(key, h.Get(key)+", "+value)
	} else {
		h.Set(key, value)
	}

	return read, false, nil
}
