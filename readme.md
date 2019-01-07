# funcserver

[![GoDoc](https://godoc.org/github.com/j0hnsmith/funcserver?status.svg)](https://godoc.org/github.com/j0hnsmith/funcserver)
[![Go Report Card](https://goreportcard.com/badge/j0hnsmith/funcserver)](https://goreportcard.com/report/j0hnsmith/funcserver)

This project provides conversion wrappers to make a `http.Handler`, such as a router, work with function-as-a-service (faas) providers (currently only AWS Lambda via an ALB implemented).

## Why?
faas means you don't have to keep the server running - no monitoring, upgrades, patching etc etc.  

### Caveats
Of course sending requests to a `http.Handler` without the usual server will have some caveats

* no streaming requests/responses
* request/response body size restrictions (1mb for lambda via ALB)
* faas suffers from slow starts ([AWS VPC very slow starts to be solved in 2019](https://twitter.com/jeremy_daly/status/1068272580556087296))

Private VPC aside, whether this is a problem for you depends on your use case, it shouldn't be for the majority.

## ALB + Lambda example
This example uses a gorilla mux router.

```go

package main

import (
    "net/http"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/gorilla/mux"
    "github.com/j0hnsmith/funcserver/alblambda"
)

func main() {
    router := mux.NewRouter()
    router.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) { resp.Write([]byte("<h1>Home</h1>")) })
    router.HandleFunc("/products", func(resp http.ResponseWriter, req *http.Request) { resp.Write([]byte("<h1>Products</h1>")) })
    router.HandleFunc("/articles", func(resp http.ResponseWriter, req *http.Request) { resp.Write([]byte("<h1>Articles</h1>")) })

    // wrap handler to automatically convert requests/responses
    lambda.Start(alblambda.WrapHTTPHandler(router, alblambda.ResponseOptions{}))
}
```	
