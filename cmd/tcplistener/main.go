package main

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/noelw19/tcptohttp/internal/request"
)

func getLinesChannel(f io.ReadCloser) <-chan string {

	lineChannel := make(chan string, 1)

	go func() {
		defer f.Close()
		defer close(lineChannel)

		line := ""
		for {
			data := make([]byte, 8)

			n, err := f.Read(data)
			if err != nil {
				break
			}

			data = data[:n]
			if i := bytes.IndexByte(data, '\n'); i != -1 {
				line += string(data[:i])
				data = data[i+1:]
				lineChannel <- line
				line = ""
			}

			line += string(data)

		}

		if len(line) != 0 {
			lineChannel <- line
		}
	}()

	return lineChannel
}

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		req, err := request.RequestFromReader(conn)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Request line:\n- Method: %s\n- Target: %s\n- Version: %s\nHeaders:\n", req.RequestLine.Method, req.RequestLine.RequestTarget, req.RequestLine.HttpVersion)
		for header := range req.Headers {
			fmt.Printf("- %s: %s\n", header, req.Headers.Get(header))
		}
		fmt.Printf("Body:\n%s", string(req.Body))
	}

}
