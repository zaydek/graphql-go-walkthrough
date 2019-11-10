package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	graphql "github.com/graph-gophers/graphql-go"
)

// This schema defines a note-taking application with two
// simple graphs: users and notes.

const schemaString = `
	schema {
		query: Query
	}
	# Define users:
	type User {
		userID: ID!
		username: String!
		emoji: String!
		notes: [Note!]!
	}
	# Define notes:
	type Note {
		noteID: ID!
		data: String!
	}
	type Query {
		# List users:
		users: [User!]!
		# Get user:
		user(userID: ID!): User!
		# List notes:
		notes(userID: ID!): [Note!]!
		# Get note:
		note(noteID: ID!): Note!
	}
`

type User struct {
	UserID   graphql.ID
	Username string
	Emoji    string
	Notes    []Note
}

type Note struct {
	NoteID graphql.ID
	Data   string
}

// Define mock data:
var users = []User{
	{
		UserID:   graphql.ID("u-001"),
		Username: "nyxerys",
		Emoji:    "🇵🇹",
		Notes: []Note{
			{NoteID: "n-001", Data: "Olá Mundo!"},
			{NoteID: "n-002", Data: "Olá novamente, mundo!"},
			{NoteID: "n-003", Data: "Olá, escuridão!"},
		},
	}, {
		UserID:   graphql.ID("u-002"),
		Username: "rdnkta",
		Emoji:    "🇺🇦",
		Notes: []Note{
			{NoteID: "n-004", Data: "Привіт Світ!"},
			{NoteID: "n-005", Data: "Привіт ще раз, світ!"},
			{NoteID: "n-006", Data: "Привіт, темрява!"},
		},
	}, {
		UserID:   graphql.ID("u-003"),
		Username: "username_ZAYDEK",
		Emoji:    "🇺🇸",
		Notes: []Note{
			{NoteID: "n-007", Data: "Hello, world!"},
			{NoteID: "n-008", Data: "Hello again, world!"},
			{NoteID: "n-009", Data: "Hello, darkness!"},
		},
	},
}

type RootResolver struct{}

func (r *RootResolver) Users() ([]User, error) {
	return users, nil
}

func (r *RootResolver) User(args struct{ UserID graphql.ID }) (User, error) {
	// Find user:
	for _, user := range users {
		if args.UserID == user.UserID {
			// Found user:
			return user, nil
		}
	}
	// Didn’t find user:
	return User{}, nil
}

func (r *RootResolver) Notes(args struct{ UserID graphql.ID }) ([]Note, error) {
	// Find user to find notes:
	user, err := r.User(args) // We can reuse resolvers.
	if reflect.ValueOf(user).IsZero() || err != nil {
		// Didn’t find user:
		return nil, err
	}
	// Found user; return notes:
	return user.Notes, nil
}

func (r *RootResolver) Note(args struct{ NoteID graphql.ID }) (Note, error) {
	// Find note:
	for _, user := range users {
		for _, note := range user.Notes {
			if args.NoteID == note.NoteID {
				// Found note:
				return note, nil
			}
		}
	}
	// Didn’t find note:
	return Note{}, nil
}

var (
	// We can pass an option to the schema so we don’t need to
	// write a method to access each type’s field:
	opts   = []graphql.SchemaOpt{graphql.UseFieldResolvers()}
	Schema = graphql.MustParseSchema(schemaString, &RootResolver{}, opts...)
)

func main() {
	ctx := context.Background()

	type ClientQuery struct {
		OpName    string
		Query     string
		Variables map[string]interface{}
	}

	q1 := ClientQuery{
		OpName: "Users",
		Query: `query Users {
			users {
				userID
				username
				emoji
			}
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
	// 		"users": [
	// 			{
	// 				"userID": "u-001",
	// 				"username": "nyxerys",
	// 				"emoji": "🇵🇹"
	// 			},
	// 			{
	// 				"userID": "u-002",
	// 				"username": "rdnkta",
	// 				"emoji": "🇺🇦"
	// 			},
	// 			{
	// 				"userID": "u-003",
	// 				"username": "username_ZAYDEK",
	// 				"emoji": "🇺🇸"
	// 			}
	// 		]
	// 	}
	// }

	q2 := ClientQuery{
		OpName: "User",
		Query: `query User($userID: ID!) {
			user(userID: $userID) {
				userID
				username
				emoji
			}
		}`,
		Variables: map[string]interface{}{
			"userID": "u-001",
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
	// 		"user": {
	// 			"userID": "u-001",
	// 			"username": "nyxerys",
	// 			"emoji": "🇵🇹"
	// 		}
	// 	}
	// }

	q3 := ClientQuery{
		OpName: "Notes",
		Query: `query Notes($userID: ID!) {
			notes(userID: $userID) {
				noteID
				data
			}
		}`,
		Variables: map[string]interface{}{
			"userID": "u-001",
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
	// 		"notes": [
	// 			{
	// 				"noteID": "n-001",
	// 				"data": "Olá Mundo!"
	// 			},
	// 			{
	// 				"noteID": "n-002",
	// 				"data": "Olá novamente, mundo!"
	// 			},
	// 			{
	// 				"noteID": "n-003",
	// 				"data": "Olá, escuridão!"
	// 			}
	// 		]
	// 	}
	// }

	q4 := ClientQuery{
		OpName: "Note",
		Query: `query Note($noteID: ID!) {
			note(noteID: $noteID) {
				noteID
				data
			}
		}`,
		Variables: map[string]interface{}{
			"noteID": "n-001",
		},
	}
	resp4 := Schema.Exec(ctx, q4.Query, q4.OpName, q4.Variables)
	json4, err := json.MarshalIndent(resp4, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(json4))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"note": {
	// 			"noteID": "n-001",
	// 			"data": "Olá Mundo!"
	// 		}
	// 	}
	// }
}
