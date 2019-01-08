package main

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gorilla/mux"
	"github.com/j0hnsmith/funcserver/alblambda"
)

// main is run by AWS Lambda.
func main() {
	router := Router()

	// wrap handler to automatically convert requests/responses
	lambda.Start(alblambda.WrapHTTPHandler(router, alblambda.ResponseOptions{}))
}

//func main1() {
//	// use the same router for development/local testing
//	router := Router()
//
//	http.ListenAndServe(":8080", router)
//}

func Router() http.Handler {
	router := mux.NewRouter()
	links := `<a href="/">Home</a><br/><a href="/products">Products</a><br/><a href="/articles">Articles</a><br/>`

	router.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(fmt.Sprintf("<h1>Home</h1>%s", links)))
	})
	router.HandleFunc("/products", func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(fmt.Sprintf("<h1>Products</h1>%s", links)))
	})
	router.HandleFunc("/articles", func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(fmt.Sprintf("<h1>Articles</h1>%s", links)))
	})

	return router
}
