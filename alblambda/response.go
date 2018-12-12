package alblambda

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// ResponseOptions holds the options for responses.
type ResponseOptions struct {
	// Multi value Headers must be explicitly enabled
	// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/lambda-functions.html#multi-value-headers
	MultiValueHeaders bool
}

// Response represents a response sent to the load balancer.
type Response struct {
	IsBase64Encoded   bool        `json:"isBase64Encoded"`
	StatusCode        int         `json:"statusCode"`
	StatusDescription string      `json:"statusDescription"`
	Headers           Headers     `json:"headers"`
	MultiValueHeaders http.Header `json:"multiValueHeaders"`
	Body              string      `json:"body"`
}

func newLambdaResponseWriter(opts ResponseOptions) *responseWriter {
	rw := &responseWriter{
		opts:          opts,
		header:        make(http.Header),
		handlerHeader: make(http.Header),
	}
	return rw
}

type responseWriter struct {
	opts              ResponseOptions
	writeHeaderCalled bool

	// handlerHeader is the Header that Handlers get access to,
	// which may be retained and mutated even after WriteHeader.
	// handlerHeader is copied into header at WriteHeader
	// time. Not strictly necessary but ensures consistency with
	// http.Response
	handlerHeader       http.Header
	handlerHeaderCalled bool
	header              http.Header
	body                bytes.Buffer
	statusCode          int
}

// Header returns the header map that will be sent by
// WriteHeader. The Header map also is the mechanism with which
// Handlers can set HTTP trailers.
//
// Changing the header map after a call to WriteHeader (or
// Write) has no effect.
func (rw *responseWriter) Header() http.Header {
	rw.handlerHeaderCalled = true
	return rw.handlerHeader
}

// we write Status header
// once WriteHeader is called, further mutations to handlerHeader are ineffective

func (rw *responseWriter) cloneHeader() {
	h2 := make(http.Header, len(rw.handlerHeader))
	for k, vv := range rw.handlerHeader {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	rw.header = h2
}

// Write writes Response data.
//
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType.
func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.writeHeaderCalled {
		rw.WriteHeader(http.StatusOK)
	}

	return rw.body.Write(data)
}

// WriteHeader sets the Status header with the provided status code.
// Only one header is set, additional calls are no-op.
//
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.writeHeaderCalled {
		fmt.Println("multiple WriteHeader calls")
		return
	}
	rw.writeHeaderCalled = true

	// https://github.com/golang/go/blob/a1aafd8b28ada0d40e2cb25fb0762ae171eec558/src/net/http/server.go#L1093
	if statusCode < 199 || statusCode > 599 {
		panic(fmt.Sprintf("invalid WriteHeader code %v", statusCode))
	}

	if rw.handlerHeaderCalled {
		rw.cloneHeader()
	}

	rw.statusCode = statusCode
}

// Should return type be a string/[]byte, using interface{} as it's assumed to be a struct with json tags.
func (rw *responseWriter) AsLambdaResponse() Response {
	if !rw.writeHeaderCalled {
		rw.WriteHeader(http.StatusOK)
	}

	resp := Response{
		StatusCode:        rw.statusCode,
		StatusDescription: http.StatusText(rw.statusCode),
		Body:              rw.body.String(),
	}

	// Ensure we've got a Content-Type header
	bodyLen := len(resp.Body)
	if bodyLen > 0 && rw.header.Get("Content-Type") == "" {
		max := 512
		if bodyLen < max {
			max = bodyLen
		}
		rw.header.Set("Content-Type", http.DetectContentType([]byte(resp.Body[:max])))
	}

	// multi/single valued Headers
	if rw.opts.MultiValueHeaders {
		resp.MultiValueHeaders = rw.header
	} else {
		resp.Headers = make(Headers)
		for k := range rw.header {
			resp.Headers[http.CanonicalHeaderKey(k)] = rw.header.Get(k)
		}
	}

	if useB64InResponseBody(rw.header.Get("Content-Type")) {
		resp.IsBase64Encoded = true
		resp.Body = base64.StdEncoding.EncodeToString([]byte(resp.Body))
	}

	return resp
}

var notB64 = map[string]bool{
	"application/json":       true,
	"application/javascript": true,
	"application/xml":        true,
}

func useB64InResponseBody(contentType string) bool {
	// this is the test that the alb applies to the request body so it should be
	// reasonable for the Response
	if strings.HasPrefix(contentType, "text/") || notB64[contentType] {
		return false
	}

	return true
}
