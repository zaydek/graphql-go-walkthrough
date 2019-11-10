package main

import (
	"context"
	"encoding/json"
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
)

// Define a more capable schema string:
const schemaString = `
	schema {
		query: Query
	}
	type Query {
		# Generic greeting, e.g. "Hello, world!":
		greet: String!
		# Customized greeting, e.g. "Hello, Johan!":
		greetPerson(person: String!): String!
		# More customized greeting, e.g. "Good morning, Johan!":
		greetPersonTimeOfDay(person: String!, timeOfDay: TimeOfDay!): String!
	}
	# Enumerate times of day:
	enum TimeOfDay {
		MORNING
		AFTERNOON
		EVENING
	}
`

type RootResolver struct{}

func (*RootResolver) Greet() string {
	return "Hello, world!"
}

func (*RootResolver) GreetPerson(args struct{ Person string }) string {
	return fmt.Sprintf("Hello, %s!", args.Person)
}

// We don’t have to use type literals; we can also write a
// convenience type:
type PersonTimeOfDayArgs struct {
	Person    string // Note that fields need to be exported.
	TimeOfDay string
}

var TimesOfDay = map[string]string{
	"MORNING":   "Good morning",
	"AFTERNOON": "Good afternoon",
	"EVENING":   "Good evening",
}

// Resolvers can also be passed a context (first argument):
func (*RootResolver) GreetPersonTimeOfDay(ctx context.Context, args PersonTimeOfDayArgs) string {
	timeOfDay, ok := TimesOfDay[args.TimeOfDay]
	if !ok {
		timeOfDay = "Go to bed"
	}
	return fmt.Sprintf("%s, %s!", timeOfDay, args.Person)
}

var Schema = graphql.MustParseSchema(schemaString, &RootResolver{})

func main() {
	ctx := context.Background()

	// We can use ClientQuery to aggregate the query’s values:
	type ClientQuery struct {
		OpName    string                 // Operation name.
		Query     string                 // Query string.
		Variables map[string]interface{} // Query variables (untyped).
	}

	q1 := ClientQuery{
		OpName: "Greet",
		Query: `query Greet {
			greet
		}`,
		Variables: nil,
	}
	resp1 := Schema.Exec(ctx, q1.Query, q1.OpName, q1.Variables)
	json1, err := json.MarshalIndent(resp1, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(json1))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"greet": "Hello, world!"
	// 	}
	// }

	q2 := ClientQuery{
		OpName: "GreetPerson",
		// GraphQL queries and mutations can be parameterized:
		Query: `query GreetPerson($person: String!) {
			greetPerson(person: $person)
		}`,
		Variables: map[string]interface{}{
			"person": "Johan",
		},
	}

	resp2 := Schema.Exec(ctx, q2.Query, q2.OpName, q2.Variables)
	json2, err := json.MarshalIndent(resp2, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(json2))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"greetPerson": "Hello, Johan!"
	// 	}
	// }

	q3 := ClientQuery{
		OpName: "GreetPersonTimeOfDay",
		Query: `query GreetPersonTimeOfDay($person: String!, $timeOfDay: TimeOfDay!) {
			greetPersonTimeOfDay(person: $person, timeOfDay: $timeOfDay)
		}`,
		Variables: map[string]interface{}{
			"person":    "Johan",
			"timeOfDay": "MORNING",
		},
	}
	resp3 := Schema.Exec(ctx, q3.Query, q3.OpName, q3.Variables)
	json3, err := json.MarshalIndent(resp3, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(json3))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"greetPersonTimeOfDay": "Good morning, Johan!"
	// 	}
	// }
}
