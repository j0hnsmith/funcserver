package funcserver

import (
	"context"
	"net/http"
)

// RequestHandler is the interface that receives an incoming request and returns a struct that can be serialised as json.
type RequestHandler func(context.Context, map[string]interface{}) (resp interface{}, err error)

// RequestConverter is the interface to convert to an incoming request to a stdlib *http.Request.
type RequestConverter interface {
	AsHTTPRequest(ctx context.Context) (*http.Request, error)
}

// ContextKey is type used to avoid context name clashes.
type ContextKey string
