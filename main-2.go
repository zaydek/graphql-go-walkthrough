package main

import (
	"context"
	"encoding/json"
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
)

// Define a schema string:
const schemaString = `
	# Define what the schema is capable of:
	schema {
		query: Query
	}
	# Define what the queries are capable of:
	type Query {
		# Generic greeting, e.g. "Hello, world!":
		greet: String!
	}
`

// Define a root resolver to hook queries onto:
type RootResolver struct{}

// Define the greet: String! query:
func (*RootResolver) Greet() string {
	return "Hello, world!"
}

// There are two ways we can define a schema:
//
// - graphql.MustParseSchema(...) *graphql.Schema // Panics on error.
// - graphql.ParseSchema(...) (*graphql.Schema, error)
//
// Define a schema:
var Schema = graphql.MustParseSchema(schemaString, &RootResolver{})

func main() {
	query := `{
		greet
	}`
	//
	// You can also use these syntax forms if you prefer:
	//
	// descriptiveQuery := `query {
	// 	greet
	// }`
	//
	// moreDescriptiveQuery := `query Greet {
	// 	greet
	// }`

	ctx := context.Background()
	resp := Schema.Exec(ctx, query, "", nil)
	json, err := json.MarshalIndent(resp, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(json))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"greet": "Hello, world!"
	// 	}
	// }
}
