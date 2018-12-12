package funcserver

import (
	"context"
	"net/http"
)

// RequestConverter is the interface to convert to an incoming request to a stdlib *http.Request.
type RequestConverter interface {
	AsHTTPRequest(ctx context.Context) (*http.Request, error)
}

// ContextKey is type used to avoid context name clashes.
type ContextKey string
