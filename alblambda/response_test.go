package alblambda

import (
	"context"
	"encoding/base64"
	"net/http"
	"testing"
)

func TestResponse(t *testing.T) { // nolint: gocyclo
	t.Run("body & default status", func(t *testing.T) {
		expectedBody := "Hello World!"
		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			_, _ = res.Write([]byte(expectedBody))
		})

		resp := callHandlerReturnResp(t, h, ResponseOptions{})

		if resp.Body != expectedBody {
			t.Errorf(`resp.Body = %q, want: "%s"`, resp.Body, expectedBody)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf(`resp.StatusCode = %q, want: %d`, resp.StatusCode, http.StatusOK)
		}
		if resp.StatusDescription != http.StatusText(http.StatusOK) {
			t.Errorf(`resp.StatusDescription = %q, want: "%s"`, resp.StatusDescription, http.StatusText(http.StatusOK))
		}
	})

	t.Run("single value header", func(t *testing.T) {
		key1 := "Some-Header1"
		value1 := "some value1"
		key2 := "Some-Header1"
		value2 := "some value1"
		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.Header().Set(key1, value1)
			res.Header().Set(key2, value2)
		})

		resp := callHandlerReturnResp(t, h, ResponseOptions{})

		if resp.Headers[key1] != value1 {
			t.Errorf(`resp.Headers[key1] = %q, want: "%s"`, resp.Headers[key1], value1)
		}
		if resp.Headers[key2] != value2 {
			t.Errorf(`resp.Headers[key2] = %q, want: "%s"`, resp.Headers[key2], value2)
		}
	})

	t.Run("multi value header", func(t *testing.T) {
		key1 := "Some-Header1"
		values1 := []string{"some value1_1", "some value1_2"}
		key2 := "Some-Header2"
		values2 := []string{"some value2_1", "some value2_2"}
		h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.Header()[key1] = values1
			res.Header()[key2] = values2
		})

		resp := callHandlerReturnResp(t, h, ResponseOptions{MultiValueHeaders: true})

		if len(resp.MultiValueHeaders) != 2 {
			t.Errorf(`len(resp.MultiValueHeaders) = %d, want: %d`, len(resp.MultiValueHeaders), 2)
		}

		for i, val := range resp.MultiValueHeaders[key1] {
			if val != values1[i] {
				t.Errorf(`resp.MultiValueHeaders[key1][i] = %q, want: "%s"`, val, values1[i])
			}
		}
		for i, val := range resp.MultiValueHeaders[key2] {
			if val != values2[i] {
				t.Errorf(`resp.MultiValueHeaders[key2][i] = %q, want: "%s"`, val, values2[i])
			}
		}
	})

	t.Run("body encoding & detected content type", func(t *testing.T) {
		bodyTests := []struct {
			name                string
			body                string
			expectedContentType string
			expectBase64Encoded bool
		}{
			{
				name:                "text/html not b64 encoded",
				body:                "<h1>Hello World!</h1>",
				expectedContentType: "text/html; charset=utf-8",
			},
			{
				name:                "image/png b64 encoded",
				body:                "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABAQMAAAAl21bKAAAAA1BMVEUAAACnej3aAAAAAXRSTlMAQObYZgAAAApJREFUCNdjYAAAAAIAAeIhvDMAAAAASUVORK5CYII=",
				expectBase64Encoded: true,
				expectedContentType: "image/png",
			},
		}

		for _, tc := range bodyTests {
			t.Run(tc.name, func(t *testing.T) {
				h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
					body := []byte(tc.body)
					if tc.expectBase64Encoded {
						var err error
						body, err = base64.StdEncoding.DecodeString(tc.body)
						if err != nil {
							t.Error(err)
						}
					}
					_, _ = res.Write(body)
				})

				resp := callHandlerReturnResp(t, h, ResponseOptions{})

				if resp.Body != tc.body {
					t.Errorf(`resp.Body = %q, want: %q`, resp.Body, tc.body)
				}
				if resp.IsBase64Encoded != tc.expectBase64Encoded {
					t.Errorf(`resp.IsBase64Encoded = %t, want: %t`, resp.IsBase64Encoded, tc.expectBase64Encoded)
				}
				if resp.Headers["Content-Type"] != tc.expectedContentType {
					t.Errorf(`resp.Headers["Content-Type"] = %s, want: %s`, resp.Headers["Content-Type"], tc.expectedContentType)
				}
			})
		}
	})
}

func callHandlerReturnResp(t *testing.T, h http.Handler, opts ResponseOptions) Response {
	f := WrapHTTPHandler(h, opts)

	albr := aLBRequest{}
	r, err := f(context.Background(), albrToMapStringInterface(albr))
	if err != nil {
		t.Error(err)
	}
	return r.(Response)
}
