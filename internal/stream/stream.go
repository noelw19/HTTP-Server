package stream

import (
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/noelw19/tcptohttp/internal/headers"
	"github.com/noelw19/tcptohttp/internal/response"
)

func bytesToStr(bs []byte) string {
	out := ""

	for _, i := range bs {
		out += fmt.Sprintf("%02x", i)
	}
	return out
}

func Streamer(w *response.Writer, h headers.Headers, reader io.ReadCloser) {
	w.WriteStatusLine(response.StatusOK)
	h.Delete("content-length")
	h.Set("transfer-encoding", "chunked")
	h.Set("trailer", "X-Content-SHA256, X-Content-Length")
	w.WriteHeaders(h)

	rawBody := []byte{}

	for {
		data := make([]byte, 32)
		n, err := reader.Read(data)
		defer reader.Close()
		if err != nil {
			break
		}
		_, err = w.WriteChunkedBody(data[:n])
		if err != nil {
			break
		}
		rawBody = append(rawBody, data[:n]...)
	}

	trailers := headers.NewHeaders()
	hash := sha256.Sum256(rawBody)
	trailers.Set("X-Content-SHA256", bytesToStr(hash[:]))
	trailers.Set("X-Content-Length", fmt.Sprintf("%d", len(rawBody)))

	w.WriteChunkedBodyDone(trailers)
	fmt.Println("Request successfully actioned and response sent")
}
