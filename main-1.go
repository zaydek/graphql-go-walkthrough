package main

import (
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
)

// This is a simple program that tests whether or not we
// downloaded and imported graphql-go correctly.
//
// It is meant to be run like this:
//
// $ go get github.com/graph-gophers/graphql-go
// $ go run main.go
//
func main() {
	fmt.Println(&graphql.Schema{})
	// Ignore the output.
}
