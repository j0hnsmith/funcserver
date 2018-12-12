package alblambda

import (
	"net/http"
)

// Headers is a container for single value HTTP Headers.
type Headers map[string]string

// AsHTTPHeader converts to a http.Header
func (h Headers) AsHTTPHeader() http.Header {
	out := make(http.Header)
	for k, v := range h {
		out.Set(k, v)
	}
	return out
}
