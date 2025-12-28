package response

import "bytes"

func isHTML(body []byte) bool {
	if bytes.Contains(body, []byte("<html>")) && bytes.Contains(body, []byte("</html>")) {
		return true
	}
	return false
}