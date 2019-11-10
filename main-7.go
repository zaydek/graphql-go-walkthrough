package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	graphql "github.com/graph-gophers/graphql-go"
)

// This example builds on main-2.go. The intent of this
// example is to demonstrate how to serve and respond to
// GraphQL queries over HTTP.

/*
 * Responders
 *
 * Responders are a clever pattern I developed that makes
 * responding to HTTP requests simpler.
 */

// stripe.com/docs/api/errors
const (
	StatusCodeOK              = 200
	StatusCodeBadRequest      = 400
	StatusCodeUnauthorized    = 401
	StatusCodeRequestFailed   = 402
	StatusCodeNotFound        = 404
	StatusCodeConflict        = 409
	StatusCodeTooManyRequests = 429
	StatusCodeServerError     = 500
)

var Statuses = map[int]string{
	StatusCodeOK:              "OK",
	StatusCodeBadRequest:      "Bad Request",
	StatusCodeUnauthorized:    "Unauthorized",
	StatusCodeRequestFailed:   "Request Failed",
	StatusCodeNotFound:        "Not Found",
	StatusCodeConflict:        "Conflict",
	StatusCodeTooManyRequests: "Too Many Requests",
	StatusCodeServerError:     "Server Error",
}

var (
	RespondOK              = NewResponder(StatusCodeOK)
	RespondBadRequest      = NewResponder(StatusCodeBadRequest)
	RespondUnauthorized    = NewResponder(StatusCodeUnauthorized)
	RespondRequestFailed   = NewResponder(StatusCodeRequestFailed)
	RespondNotFound        = NewResponder(StatusCodeNotFound)
	RespondConflict        = NewResponder(StatusCodeConflict)
	RespondTooManyRequests = NewResponder(StatusCodeTooManyRequests)
	RespondServerError     = NewResponder(StatusCodeServerError)
)

func NewResponder(statusCode int) func(http.ResponseWriter) {
	respond := func(w http.ResponseWriter) {
		if statusCode >= 200 && statusCode <= 299 {
			w.WriteHeader(statusCode)
			return
		}
		status := Statuses[statusCode]
		http.Error(w, status, statusCode)
	}
	return respond
}

/*
 * main
 */
const schemaString = `
	schema {
		query: Query
	}
	type Query {
		greet: String!
	}
`

type RootResolver struct{}

func (*RootResolver) Greet() (string, error) {
	return "Hello, world!", nil
}

var Schema = graphql.MustParseSchema(schemaString, &RootResolver{})

func main() {
	// Client-side request; this goroutine simulates a client-
	// side service, e.g. an app or a service that consumes
	// this API.
	//
	// The reason we’re using a goroutine is so we don’t block
	// the server from responding to the request.
	go func() {
		// To perform a query over HTTP, we can use a GET
		// request and concatenate the query to the ?query= URL
		// parameter. This is common practice for getting
		// started.
		queryParam := url.QueryEscape(`{ greet }`)
		resp, err := http.Get("http://localhost:8000/graphql?query=" + queryParam)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		bstr, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(bstr))
		// Expected output:
		//
		// {
		// 	"data": {
		// 		"greet": "Hello, world!"
		// 	}
		// }
	}()

	http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		// This is the GraphQL endpoint (/graphql). It has
		// several responsibilities:
		//
		// - Ignore non non-GET request.
		// - Get the URL’s parameters (to access ?query=...).
		// - Perform the query against the schema.
		// - Respond to errors with HTTP status codes.
		//
		if r.Method != http.MethodGet {
			RespondNotFound(w)
			return
		}
		params := r.URL.Query()
		resp := Schema.Exec(context.Background(), params.Get("query"), "", nil)
		if len(resp.Errors) > 0 {
			RespondServerError(w)
			log.Printf("Schema.Exec: %+v", resp.Errors)
			return
		}
		json, err := json.MarshalIndent(resp, "", "\t")
		if err != nil {
			RespondServerError(w)
			log.Printf("json.MarshalIndent: %s", err)
			return
		}
		fmt.Fprint(w, string(json))
	})
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		panic(err)
	}
}
