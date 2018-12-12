package alblambda

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
)

// WrapHTTPHandler is wrapper around a http.Handler to convert requests & responses for use in a AWS Lambda function
// with requests coming from a ALB. Subject to a few caveats (max payload 1mb, no streaming requests/responses, possible
// slow start delay), a vanilla http.Handler can be easily used with Lambda.
func WrapHTTPHandler(h http.Handler, opts ResponseOptions) func(context.Context, ALBRequest) (interface{}, error) {
	return func(ctx context.Context, r ALBRequest) (resp interface{}, err error) {

		req, err := r.AsHTTPRequest(ctx)
		if err != nil {
			return
		}

		res := newLambdaResponseWriter(opts)

		defer func() {
			if r := recover(); r != nil {
				switch e := r.(type) {
				case string:
					err = errors.New(e)
					return
				default:
					err = errors.New("panic: unknown cause")
					return
				}
			}
		}()

		h.ServeHTTP(res, req)

		// Response written, convert to alblambda format
		resp = res.AsLambdaResponse()
		return resp, err
	}
}
