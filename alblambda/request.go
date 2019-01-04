package alblambda

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/j0hnsmith/funcserver"
)

// An aLBRequest represents an http request received by an application load balancer and forwarded to a alblambda function.
// It's container to marshal json into that can be converted to a *http.Request.
// https://docs.aws.amazon.com/lambda/latest/dg/services-alb.html
type aLBRequest struct {
	RequestContext                  requestContext          `json:"requestContext"`
	HTTPMethod                      string                  `json:"httpMethod"`
	Path                            string                  `json:"path"`
	QueryStringParameters           queryStringParameters   `json:"queryStringParameters,omitempty"`
	MultiValueQueryStringParameters mVQueryStringParameters `json:"mVQueryStringParameters,omitempty"`
	Headers                         Headers                 `json:"Headers,omitempty"`
	MultiValueHeaders               http.Header             `json:"multiValueHeaders,omitempty"`
	IsBase64Encoded                 bool                    `json:"isBase64Encoded"`

	// limited to 1mb in size
	// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/lambda-functions.html
	Body string `json:"body"`
}

var _ funcserver.RequestConverter = aLBRequest{}

// ELB holds information about the elastic load balancer that received the http request.
// This is accessed via the context on a http.Request, eg ctx.Get("elb"), then type assert.
type ELB struct {
	TargetGroupArn string `json:"targetGroupArn"`
}

// requestContext holds information pertinent to the request.
type requestContext struct {
	ELB `json:"elb"`
}

// AsHTTPRequest converts to the equivalent *http.Request so that the request can be processed via standard net/http
// functionality.
func (albr aLBRequest) AsHTTPRequest(ctx context.Context) (*http.Request, error) {
	var qp string
	if len(albr.MultiValueQueryStringParameters) > 0 {
		qp = albr.MultiValueQueryStringParameters.AsQueryString()

	} else {
		qp = albr.QueryStringParameters.AsQueryString()
	}

	var headers http.Header
	if len(albr.MultiValueHeaders) > 0 {
		headers = albr.MultiValueHeaders
	} else {
		headers = albr.Headers.AsHTTPHeader()
	}

	bodyStr := albr.Body
	if albr.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(albr.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to decode body as base64: %s", albr.Body)
		}
		bodyStr = string(decoded)
	}

	r := &http.Request{
		Method: albr.HTTPMethod,
		URL: &url.URL{
			Path:     albr.Path,
			RawQuery: qp,
		},
		Header: headers,
		Body:   ioutil.NopCloser(strings.NewReader(bodyStr)),
	}

	ctx = context.WithValue(ctx, funcserver.ContextKey("elb"), albr.RequestContext.ELB)
	r = r.WithContext(ctx)

	return r, nil
}

// queryStringParameters is a container for query params.
type queryStringParameters map[string]string

// AsQueryString converts to a querystring as it's not possible to pass url.Values into a net.URL.
func (qsp queryStringParameters) AsQueryString() string {
	b := new(strings.Builder)
	first := true
	for k, v := range qsp {
		if first {
			first = false
		} else {
			b.WriteString("&") // nolint: gosec
		}
		_, _ = fmt.Fprintf(b, "%s=%s", k, v) // nolint: gosec
	}
	return b.String()
}

// mVQueryStringParameters is a container for multi value query params. Must be explicity enabled, mutually
// exclusive with queryStringParameters.
// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/lambda-functions.html#multi-value-headers
type mVQueryStringParameters map[string][]string

// AsQueryString converts to a querystring with multiple values as it's not possible to pass
// url.Values into a net.URL.
func (mqsp mVQueryStringParameters) AsQueryString() string {
	b := new(strings.Builder)
	first := true
	for k, items := range mqsp {
		for _, v := range items {
			if first {
				first = false
			} else {
				b.WriteString("&") // nolint: gosec
			}
			_, _ = fmt.Fprintf(b, "%s=%s", k, v)
		}
	}
	return b.String()
}
