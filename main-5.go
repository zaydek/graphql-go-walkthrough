package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	graphql "github.com/graph-gophers/graphql-go"
)

// In the previous example, we used:
//
//  opts   = []graphql.SchemaOpt{graphql.UseFieldResolvers()}
//  Schema = graphql.MustParseSchema(schemaString, &RootResolver{}, opts...)
//
// This time we won’t use graphql.UseFieldResolvers.
// UseFieldResolvers creates a 1:1 relationship with a
// type’s fields.
//
// Instead, we’ll define a custom resolver per type and use
// methods as accessors. This affords us more power at the
// cost of some convenience.
//
// This is a powerful pattern; by using methods as accessors
// to a type’s fields, we afford more degrees of freedom.
// For example, we can delegate resolvers to resolvers (see
// the last line of RootResolver.Notes).
//
// We’re also going to parse the schema from an actual
// GraphQL file (see main-5-schema.graphql).
//
// Next, we’ll prefer using pointers instead of structs so
// we can return nil instead of User{}, for example. This is
// just a personal preference.
//
// Last, we’ll add a createNote mutation and query all users
// and all of their notes to confirm we created a note.

type User struct {
	UserID   graphql.ID
	Username string
	Emoji    string
	Notes    []*Note
}

type Note struct {
	NoteID graphql.ID
	Data   string
}

type NoteInput struct{ Data string }

var users = []*User{
	{
		UserID:   graphql.ID("u-001"),
		Username: "nyxerys",
		Emoji:    "🇵🇹",
		Notes: []*Note{
			{NoteID: "n-001", Data: "Olá Mundo!"},
			{NoteID: "n-002", Data: "Olá novamente, mundo!"},
			{NoteID: "n-003", Data: "Olá, escuridão!"},
		},
	}, {
		UserID:   graphql.ID("u-002"),
		Username: "rdnkta",
		Emoji:    "🇺🇦",
		Notes: []*Note{
			{NoteID: "n-004", Data: "Привіт Світ!"},
			{NoteID: "n-005", Data: "Привіт ще раз, світ!"},
			{NoteID: "n-006", Data: "Привіт, темрява!"},
		},
	}, {
		UserID:   graphql.ID("u-003"),
		Username: "username_ZAYDEK",
		Emoji:    "🇺🇸",
		Notes: []*Note{
			{NoteID: "n-007", Data: "Hello, world!"},
			{NoteID: "n-008", Data: "Hello again, world!"},
			{NoteID: "n-009", Data: "Hello, darkness!"},
		},
	},
}

/*
 * RootResolver
 */

type RootResolver struct{}

func (r *RootResolver) Users() ([]*UserResolver, error) {
	var userRxs []*UserResolver
	for _, u := range users {
		userRxs = append(userRxs, &UserResolver{u})
	}
	return userRxs, nil
}

func (r *RootResolver) User(args struct{ UserID graphql.ID }) (*UserResolver, error) {
	// Find user:
	for _, user := range users {
		if args.UserID == user.UserID {
			// Found user:
			return &UserResolver{user}, nil
		}
	}
	// Didn’t find user:
	return nil, nil
}

func (r *RootResolver) Notes(args struct{ UserID graphql.ID }) ([]*NoteResolver, error) {
	// Find user to find notes:
	user, err := r.User(args)
	if user == nil || err != nil {
		// Didn’t find user:
		return nil, err
	}
	// Found user; return notes:
	return user.Notes(), nil // We can reuse resolvers on resolvers, oh my.
}

func (r *RootResolver) Note(args struct{ NoteID graphql.ID }) (*NoteResolver, error) {
	// Find note:
	for _, user := range users {
		for _, note := range user.Notes {
			if args.NoteID == note.NoteID {
				// Found note:
				return &NoteResolver{note}, nil
			}
		}
	}
	// Didn’t find note:
	return nil, nil
}

type CreateNoteArgs struct {
	UserID graphql.ID
	Note   NoteInput
}

func (r *RootResolver) CreateNote(args CreateNoteArgs) (*NoteResolver, error) {
	// Find user:
	var note *Note
	for _, user := range users {
		if args.UserID == user.UserID {
			// Create a note with a note ID of n-010:
			note = &Note{NoteID: "n-010", Data: args.Note.Data}
			user.Notes = append(user.Notes, note) // Push note.
		}
	}
	// Return note:
	return &NoteResolver{note}, nil
}

/*
 * UserResolver
 *
 * type User {
 * 	userID: ID!
 * 	username: String!
 * 	emoji: String!
 * 	notes: [Note!]!
 * }
 */

type UserResolver struct{ u *User }

func (r *UserResolver) UserID() graphql.ID {
	return r.u.UserID
}

func (r *UserResolver) Username() string {
	return r.u.Username
}

func (r *UserResolver) Emoji() string {
	return r.u.Emoji
}

// Opt to return []*NoteResolver instead of []*Note:
func (r *UserResolver) Notes() []*NoteResolver {
	var noteRxs []*NoteResolver
	for _, note := range r.u.Notes {
		noteRxs = append(noteRxs, &NoteResolver{note})
	}
	return noteRxs
}

/*
 * NoteResolver
 *
 * type Note {
 * 	noteID: ID!
 * 	data: String!
 * }
 */

type NoteResolver struct{ n *Note }

func (r *NoteResolver) NoteID() graphql.ID {
	return r.n.NoteID
}

func (r *NoteResolver) Data() string {
	return r.n.Data
}

/*
 * main
 */

func main() {
	ctx := context.Background()

	// Read and parse the schema:
	bstr, err := ioutil.ReadFile("./main-5-schema.graphql")
	if err != nil {
		panic(err)
	}
	schemaString := string(bstr)
	schema, err := graphql.ParseSchema(schemaString, &RootResolver{})
	if err != nil {
		panic(err)
	}

	// We can use a type alias for convenience.
	//
	// NOTE: It’s not recommended to use a true type because
	// you’ll need to implement MarshalJSON and UnmarshalJSON.
	type JSON = map[string]interface{}

	type ClientQuery struct {
		OpName    string
		Query     string
		Variables JSON
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
	resp1 := schema.Exec(ctx, q1.Query, q1.OpName, q1.Variables)
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
		Variables: JSON{
			"userID": "u-001",
		},
	}
	resp2 := schema.Exec(ctx, q2.Query, q2.OpName, q2.Variables)
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
		Variables: JSON{
			"userID": "u-001",
		},
	}
	resp3 := schema.Exec(ctx, q3.Query, q3.OpName, q3.Variables)
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
		Variables: JSON{
			"noteID": "n-001",
		},
	}
	resp4 := schema.Exec(ctx, q4.Query, q4.OpName, q4.Variables)
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

	q5 := ClientQuery{
		OpName: "CreateNote",
		Query: `mutation CreateNote($userID: ID!, $note: NoteInput!) {
			createNote(userID: $userID, note: $note) {
				noteID
				data
			}
		}`,
		Variables: JSON{
			"userID": "u-003",
			"note": JSON{
				"data": "We created a note!",
			},
		},
	}
	resp5 := schema.Exec(ctx, q5.Query, q5.OpName, q5.Variables)
	json5, err := json.MarshalIndent(resp5, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(json5))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"createNote": {
	// 			"noteID": "n-010",
	// 			"data": "We created a note!"
	// 		}
	// 	}
	// }

	q6 := ClientQuery{
		OpName: "Users",
		Query: `query Users {
			users {
				userID
				username
				emoji
				notes {
					noteID
					data
				}
			}
		}`,
		Variables: nil,
	}
	resp6 := schema.Exec(ctx, q6.Query, q6.OpName, q6.Variables)
	json6, err := json.MarshalIndent(resp6, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(json6))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"users": [
	// 			// ...
	// 			{
	// 				"userID": "u-003",
	// 				"username": "username_ZAYDEK",
	// 				"emoji": "🇺🇸",
	// 				"notes": [
	// 					// ...
	// 					{
	// 						"noteID": "n-010",
	// 						"data": "We created a note!"
	// 					}
	// 				]
	// 			}
	// 		]
	// 	}
	// }
}
