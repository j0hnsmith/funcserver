package alblambda_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/j0hnsmith/funcserver"
	"github.com/j0hnsmith/funcserver/alblambda"
)

func TestRequest(t *testing.T) { // nolint: gocyclo
	t.Run("method", func(t *testing.T) {
		expectedMethod := http.MethodGet
		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			if req.Method != expectedMethod {
				t.Errorf(`resp.Method = %q, want: "%s"`, req.Method, expectedMethod)
			}
		})

		f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})

		albr := alblambda.ALBRequest{HTTPMethod: expectedMethod}
		_, err := f(context.Background(), albr)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("path", func(t *testing.T) {
		expectedPath := "/some/path"
		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			if req.URL.Path != expectedPath {
				t.Errorf(`resp.URL.Path = %q, want: "%s"`, req.Method, expectedPath)
			}
		})

		f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})
		albr := alblambda.ALBRequest{Path: expectedPath}
		_, err := f(context.Background(), albr)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("single value query params", func(t *testing.T) {
		key1 := "someKey1"
		val1 := "someVal1"
		key2 := "someKey2"
		val2 := "someVal2"
		qp := make(alblambda.QueryStringParameters)
		qp[key1] = val1
		qp[key2] = val2

		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			vals := req.URL.Query()
			if vals.Get(key1) != val1 {
				t.Errorf(`req.URL.Query().Get(key1) = %q, want: "%s"`, vals.Get(key1), val1)
			}
			if vals.Get(key2) != val2 {
				t.Errorf(`req.URL.Query().Get(key2) = %q, want: "%s"`, vals.Get(key2), val2)
			}
		})

		f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})
		albr := alblambda.ALBRequest{QueryStringParameters: qp}
		_, err := f(context.Background(), albr)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("multiple value query params", func(t *testing.T) {
		key1 := "someKey1"
		val1 := []string{"someVal1-1", "someVal1-2"}
		key2 := "someKey2"
		val2 := []string{"someVal2-1", "someVal2-2"}

		qp := make(alblambda.MVQueryStringParameters)
		qp[key1] = val1
		qp[key2] = val2

		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			vals := req.URL.Query()
			if !reflect.DeepEqual(vals[key1], val1) {
				t.Errorf(`vals[key1] = %q, want: %s`, vals[key1], val1)
			}
			if !reflect.DeepEqual(vals[key2], val2) {
				t.Errorf(`vals[key2] = %q, want: %s`, vals[key2], val2)
			}
		})

		f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})
		albr := alblambda.ALBRequest{MultiValueQueryStringParameters: qp}
		_, err := f(context.Background(), albr)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("single value headers", func(t *testing.T) {
		key1 := "Content-Type"
		val1 := "application/json"
		key2 := "Accept"
		val2 := "text/html"
		headers := make(alblambda.Headers)
		headers[key1] = val1
		headers[key2] = val2

		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			if req.Header.Get(key1) != val1 {
				t.Errorf(`req.Header.Get(key1) = %q, want: "%s"`, req.Header.Get(key1), val1)
			}
			if req.Header.Get(key2) != val2 {
				t.Errorf(`req.Header.Get(key2) = %q, want: "%s"`, req.Header.Get(key2), val2)
			}
		})

		f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})
		albr := alblambda.ALBRequest{Headers: headers}
		_, err := f(context.Background(), albr)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("multi value headers", func(t *testing.T) {
		key1 := http.CanonicalHeaderKey("cookie")
		val1 := []string{"name1-1", "name1-2"}
		key2 := http.CanonicalHeaderKey("another-header")
		val2 := []string{"someVal2-1", "someVal2-2"}
		mvh := make(http.Header)
		mvh[key1] = val1
		mvh[key2] = val2

		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			if !reflect.DeepEqual(req.Header[key1], val1) {
				t.Errorf(`req.Header[key1] = %q, want: %s`, req.Header[key1], val1)
			}
			if !reflect.DeepEqual(req.Header[key2], val2) {
				t.Errorf(`req.Header[key2Header] = %q, want: %s`, req.Header[key2], val2)
			}
		})

		f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})
		albr := alblambda.ALBRequest{MultiValueHeaders: mvh}
		_, err := f(context.Background(), albr)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("body", func(t *testing.T) {
		bodyTests := []struct {
			name            string
			rawBody         string
			isBase64Encoded bool
			expectedReqBody string
		}{
			{
				name:            "not b64 encoded",
				rawBody:         "request body",
				expectedReqBody: "request body",
			},
			{
				name:            "b64 encoded",
				rawBody:         "cmVxdWVzdCBib2R5",
				isBase64Encoded: true,
				expectedReqBody: "request body",
			},
		}

		for _, tc := range bodyTests {
			t.Run(tc.name, func(t *testing.T) {
				h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
					body, err := ioutil.ReadAll(req.Body)
					if err != nil {
						t.Error(err)
						return
					}

					if string(body) != tc.expectedReqBody {
						t.Errorf(`body = %q, want: %q`, body, tc.expectedReqBody)
					}
				})

				f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})
				albr := alblambda.ALBRequest{Body: tc.rawBody, IsBase64Encoded: tc.isBase64Encoded}
				_, err := f(context.Background(), albr)
				if err != nil {
					t.Error(err)
				}
			})
		}
	})

	t.Run("context", func(t *testing.T) {
		rc := alblambda.RequestContext{
			ELB: alblambda.ELB{
				TargetGroupArn: "arn:aws:elasticloadbalancing:region:123456789012:targetgroup/my-target-group/6d0ecf831eec9f09",
			},
		}

		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			elb := ctx.Value(funcserver.ContextKey("elb")).(alblambda.ELB)
			if elb.TargetGroupArn != rc.ELB.TargetGroupArn {
				t.Errorf(`elb.TargetGroupArn = %q, want: %q`, elb.TargetGroupArn, rc.ELB.TargetGroupArn)
			}
		})

		f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})

		albr := alblambda.ALBRequest{RequestContext: rc}
		_, err := f(context.Background(), albr)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid status panic recover", func(t *testing.T) {
		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(999)
		})

		f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})

		albr := alblambda.ALBRequest{}
		_, err := f(context.Background(), albr)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("req body encoding error", func(t *testing.T) {
		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {})

		f := alblambda.WrapHTTPHandler(h, alblambda.ResponseOptions{})

		albr := alblambda.ALBRequest{Body: "not base64", IsBase64Encoded: true}
		_, err := f(context.Background(), albr)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
