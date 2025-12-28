package response

import (
	"fmt"
	"io"
	"strconv"

	"github.com/noelw19/tcptohttp/internal/headers"
)

type writerState int

const (
	writerStateNotStarted writerState = 1
	writerStateStatusLine writerState = 2
	writerStateHeaders    writerState = 3
	writerStateBody       writerState = 4
)

type Writer struct {
	Writer      io.Writer
	writerState writerState
}

func NewResponseWriter(w io.Writer) *Writer {
	return &Writer{
		Writer:      w,
		writerState: writerStateNotStarted,
	}
}

func (w *Writer) isCorrectState(expected writerState) error {
	if expected == w.writerState {
		return nil
	}
	return fmt.Errorf("you have executed the writers in the wrong order: current: %d, expected: %d", w.writerState, expected)
}

func (w *Writer) Respond(status StatusCode, h headers.Headers, body []byte) {
	err := w.WriteStatusLine(status)
	if err != nil {
		fmt.Println(err, status, string(body))
		return
	}
	h.Replace("content-length", fmt.Sprintf("%d", len(body)))

	if isHTML(body) {
		h.Replace("content-type", "text/html")
	}

	err = w.WriteHeaders(h)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = w.WriteBody(body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Request successfully actioned and response sent")
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	err := w.isCorrectState(writerStateNotStarted)
	if err != nil {
		return err
	}

	version := "HTTP/1.1"
	reason := GetStatusReason(statusCode)

	statusLine := fmt.Appendf(nil, "%s %d %s\r\n", version, statusCode, reason)
	_, err = w.Writer.Write(statusLine)

	w.writerState = writerStateStatusLine
	return err
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	err := w.isCorrectState(writerStateStatusLine)
	if err != nil {
		return err
	}

	hasBody := false

	if _, ok := headers.HasContentLength(); ok {
		hasBody = true
	}

	if len(headers) == 0 || headers == nil {
		headers = GetDefaultHeaders(0)
	}

	for key := range headers {

		headerLine := fmt.Sprintf("%s:%s\r\n", key, headers.Get(key))
		_, err := w.Writer.Write([]byte(headerLine))
		if err != nil {
			return err
		}
	}
	// write the final \r\n if there is a body
	if hasBody {
		_, err := w.Writer.Write([]byte("\r\n"))
		if err != nil {
			return err
		}
	}

	w.writerState = writerStateHeaders
	return nil
}
func (w *Writer) WriteBody(p []byte) (int, error) {
	err := w.isCorrectState(writerStateHeaders)
	if err != nil {
		return 0, err
	}

	bodyString := string(p) + "\r\n"
	n, err := w.Writer.Write([]byte(bodyString))
	if err != nil {
		return n, err
	}

	w.writerState = writerStateBody
	return n, err
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()

	h.Set("content-length", fmt.Sprintf("%d", contentLen))
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")

	return h
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	length := strconv.FormatInt(int64(len(p)), 16)
	read := 0
	n, err := w.Writer.Write([]byte(length + "\r\n"))
	if err != nil {
		return n, err
	}
	read += n
	n, err = w.Writer.Write(fmt.Appendf(p, "\r\n"))
	if err != nil {
		return n, err
	}
	read += n

	return read, nil
}

func (w *Writer) WriteChunkedBodyDone(trailers headers.Headers) (int, error) {
	n, err := w.Writer.Write([]byte("0\r\n"))
	if err != nil {
		return n, err
	}

	if len(trailers) > 0 {
		err = w.WriteTrailers(trailers)
		if err != nil {
			return n, err
		}
	}

	n, err = w.Writer.Write([]byte("\r\n"))
	if err != nil {
		return n, err
	}
	return 0, nil
}

func (w *Writer) WriteTrailers(trailers headers.Headers) error {
	for key := range trailers {

		headerLine := fmt.Sprintf("%s:%s\r\n", key, trailers.Get(key))
		_, err := w.Writer.Write([]byte(headerLine))
		if err != nil {
			return err
		}
	}
	return nil
}
