package alblambda

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/j0hnsmith/funcserver"
	"github.com/pkg/errors"
)

// WrapHTTPHandler is wrapper around a http.Handler to convert requests & responses for use in a AWS Lambda function
// with requests coming from a ALB. Subject to a few caveats (max payload 1mb, no streaming requests/responses, possible
// slow start delay), a vanilla http.Handler can be easily used with Lambda.
func WrapHTTPHandler(h http.Handler, opts ResponseOptions) funcserver.RequestHandler {
	return func(ctx context.Context, r map[string]interface{}) (resp interface{}, err error) {

		// this is ugly and slow, it's done to meet the funcserver.RequestHandler interface and remove the
		// need to export the type aLBRequest and everything that comes with it
		data, err := json.Marshal(r)
		if err != nil {
			return
		}
		albr := new(aLBRequest)
		err = json.Unmarshal(data, albr)
		if err != nil {
			return
		}
		// end uglyness

		req, err := albr.AsHTTPRequest(ctx)
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
