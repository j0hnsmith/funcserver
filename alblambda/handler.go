/*
Package alblambda provides a conversion wrapper so that a http.Handler (think a generic router) can be used in an AWS
lambda function, incoming requests are routed via an application load balancer. This essentially means, subject to
some caveats (more on those later), your http server can now be serverless (according so some definitions of
serverless at least).

Application load balancers can now send incoming requests to lambda functions, the lambda is called with a json
payload that gets marshalled into a struct. This package takes the payload and turns it into a http.Request then \
calls your http.Handler with it and an appropriate http.ResponseWriter (conveniently an interface) before returning
an appropriate response to the load balancer.

Here's some background info
https://docs.aws.amazon.com/elasticloadbalancing/latest/application/lambda-functions.html.

Caveats:

- request & response bodies are limited to 1mb in size (headers have separate size limits)

- no streaming requests/responses, request is received in full before the lambda is invoked, handler must return before
response is sent to the load balancer

- lambda functions are subject to 'cold start'. Can be as little as 100-200ms, if you use lambdas
in private vpcs it can be ten+ seconds (being fixed in 2019 https://www.nuweba.com/AWS-Lambda-in-a-VPC-will-soon-be-faster).
https://medium.freecodecamp.org/lambda-vpc-cold-starts-a-latency-killer-5408323278dd

Example usage:
	package main

	import (
		"net/http"

		"github.com/aws/aws-lambda-go/lambda"
		"github.com/gorilla/mux"
		"github.com/j0hnsmith/funcserver/alblambda"
	)

	func main() {

		// any http.Handler, let's use a gorilla/mux router
		router := mux.NewRouter()
		router.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) { resp.Write([]byte("<h1>Home</h1>")) })
		router.HandleFunc("/products", func(resp http.ResponseWriter, req *http.Request) { resp.Write([]byte("<h1>Products</h1>")) })
		router.HandleFunc("/articles", func(resp http.ResponseWriter, req *http.Request) { resp.Write([]byte("<h1>Articles</h1>")) })

		// wrap handler to automatically convert requests/responses
		lambda.Start(alblambda.WrapHTTPHandler(router, alblambda.ResponseOptions{}))
	}

*/
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
