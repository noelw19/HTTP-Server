package headers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test: Valid single header
func TestSingleHeader(t *testing.T) {

	headers := NewHeaders()
	data := []byte("Host: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "localhost:42069", headers["Host"])
	assert.Equal(t, 23, n)
	assert.False(t, done)

	headers = NewHeaders()
	data = []byte("       Host : localhost:42069       \r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)
}

// Test: Invalid spacing header
func TestValidSingleHeader(t *testing.T) {
	headers := NewHeaders()

	data := []byte("H©st: localhost:42069\r\n\r\n")
	_, _, err := headers.Parse(data)
	require.Error(t, err)
}

func TestInvalidSingleHeader(t *testing.T) {
	headers := NewHeaders()

	data := []byte("H©st: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)
}

func TestMultipleHeaders(t *testing.T) {
	headers := NewHeaders()

	data := []byte("Set-Person: lane-loves-go\r\n\r\n")
	data1 := []byte("Set-Person: prime-loves-zig\r\n\r\n")
	data2 := []byte("Set-Person: tj-loves-ocaml\r\n\r\n")
	_, done, err := headers.Parse(data)
	_, _, _ = headers.Parse(data1)
	_, _, _ = headers.Parse(data2)
	fmt.Println(headers)
	require.NoError(t, err)
	assert.Equal(t, "lane-loves-go, prime-loves-zig, tj-loves-ocaml", headers["set-person"])
	assert.False(t, done)
}
