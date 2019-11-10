package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"

	graphql "github.com/graph-gophers/graphql-go"
	_ "github.com/lib/pq"
)

// This version uses a Postgres database with mock data.
// The root resolvers actually reads from the database and
// the root mutations actually writes to the database.
//
// This version relies on some setup:
//
// $ psql -d postgres
// postgres=# create database graph_gophers; # Create database.
// postgres=# \c graph_gohpers               # Connect to database.
// graph_gohpers=# begin;                    # Begin transaction.
// graph_gohpers=# \i main-6-schema.sql      # Load tables and mock data (\i is interactive mode).
// graph_gohpers=# commit;                   # Commit transaction.
// graph_gophers=# select * from users;      # Check users.
//
//  user_id  | username
// ----------+----------
//  u-f4ff7e | nyxerys
//  u-260753 | rdnkta
//  u-33e723 | zaydek
//
// graph_gophers=# select * from notes;      # Check notes.
//
//  user_id  | note_id  |         data
// ----------+----------+-----------------------
//  u-f4ff7e | n-b2c043 | Olá Mundo!
//  u-f4ff7e | n-95d818 | Olá novamente, mundo!
//  u-f4ff7e | n-80a459 | Olá, escuridão!
//  u-260753 | n-5f06d0 | Привіт Світ!
//  u-260753 | n-1c4bbe | Привіт ще раз, світ!
//  u-260753 | n-453ff5 | Привіт, темрява!
//  u-33e723 | n-81e59b | Hello, world!
//  u-33e723 | n-b8b326 | Hello again, world!
//  u-33e723 | n-7fdd0c | Hello, darkness!
//
// Now we can connect to the graph_gophers database, parse
// the schema, then query and mutate as per usual, but this
// time against a real database.

type User struct {
	UserID   graphql.ID
	Username string
	Notes    []*Note
}

type Note struct {
	NoteID graphql.ID
	Data   string
}

type NoteInput struct{ Data string }

/*
 * RootResolver
 */

type RootResolver struct{}

func (r *RootResolver) Users() ([]*UserResolver, error) {
	var userRxs []*UserResolver
	rows, err := DB.Query(`
		SELECT
			user_id,
			username
		FROM users
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.UserID, &user.Username)
		if err != nil {
			return nil, err
		}
		userRxs = append(userRxs, &UserResolver{user})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return userRxs, nil
}

func (r *RootResolver) User(args struct{ UserID graphql.ID }) (*UserResolver, error) {
	user := &User{}
	err := DB.QueryRow(`
		SELECT
			user_id,
			username
		FROM users
		WHERE user_id = $1
	`, args.UserID).Scan(&user.UserID, &user.Username)
	if err != nil {
		return nil, err
	}
	return &UserResolver{user}, nil
}

func (r *RootResolver) Notes(args struct{ UserID graphql.ID }) ([]*NoteResolver, error) {
	var noteRxs []*NoteResolver
	rows, err := DB.Query(`
		SELECT
			note_id,
			data
		FROM notes
		WHERE user_id = $1
	`, args.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		note := &Note{}
		err := rows.Scan(&note.NoteID, &note.Data)
		if err != nil {
			return nil, err
		}
		noteRxs = append(noteRxs, &NoteResolver{note})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return noteRxs, nil
}

func (r *RootResolver) Note(args struct{ NoteID graphql.ID }) (*NoteResolver, error) {
	note := &Note{}
	err := DB.QueryRow(`
		SELECT
			note_id,
			data
		FROM notes
		WHERE note_id = $1
	`, args.NoteID).Scan(&note.NoteID, &note.Data)
	if err != nil {
		return nil, err
	}
	return &NoteResolver{note}, nil
}

type CreateNoteArgs struct {
	UserID graphql.ID
	Note   NoteInput
}

func (r *RootResolver) CreateNote(args CreateNoteArgs) (*NoteResolver, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var noteID string
	err = tx.QueryRow(`
		INSERT INTO notes (
			user_id,
			data )
		VALUES ($1, $2)
		RETURNING note_id
	`, args.UserID, args.Note.Data).Scan(&noteID)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return r.Note(struct{ NoteID graphql.ID }{graphql.ID(noteID)})
}

/*
 * UserResolver
 */

type UserResolver struct{ u *User }

func (r *UserResolver) UserID() graphql.ID {
	return r.u.UserID
}

func (r *UserResolver) Username() string {
	return r.u.Username
}

func (r *UserResolver) Notes() ([]*NoteResolver, error) {
	rootRx := &RootResolver{}
	return rootRx.Notes(struct{ UserID graphql.ID }{UserID: r.u.UserID})
}

/*
 * NoteResolver
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

var DB *sql.DB

var Schema *graphql.Schema

func check(err error, desc string) {
	if err == nil {
		return
	}
	errStr := fmt.Sprintf("%s: %s", desc, err)
	panic(errStr)
}

func main() {
	// Connect to database:
	var err error
	DB, err = sql.Open("postgres", "postgres://zaydek@localhost/graph_gophers?sslmode=disable")
	check(err, "sql.Open")
	err = DB.Ping()
	check(err, "DB.Ping")
	defer DB.Close()

	// Parse schema:
	bstr, err := ioutil.ReadFile("./main-6-schema.graphql")
	check(err, "ioutil.ReadFile")
	schemaString := string(bstr)
	Schema, err = graphql.ParseSchema(schemaString, &RootResolver{})
	check(err, "graphql.ParseSchema")

	ctx := context.Background()

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
			}
		}`,
		Variables: nil,
	}
	resp1 := Schema.Exec(ctx, q1.Query, q1.OpName, q1.Variables)
	json1, err := json.MarshalIndent(resp1, "", "\t")
	check(err, "json.MarshalIndent")
	fmt.Println(string(json1))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"users": [
	// 			{
	// 				"userID": "u-4f122c",
	// 				"username": "nyxerys"
	// 			},
	// 			{
	// 				"userID": "u-03f744",
	// 				"username": "rdnkta"
	// 			},
	// 			{
	// 				"userID": "u-93ecba",
	// 				"username": "zaydek"
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
			}
		}`,
		Variables: JSON{
			"userID": "u-f4ff7e",
		},
	}
	resp2 := Schema.Exec(ctx, q2.Query, q2.OpName, q2.Variables)
	json2, err := json.MarshalIndent(resp2, "", "\t")
	check(err, "json.MarshalIndent")
	fmt.Println(string(json2))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"user": {
	// 			"userID": "u-f4ff7e",
	// 			"username": "nyxerys"
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
			"userID": "u-f4ff7e",
		},
	}
	resp3 := Schema.Exec(ctx, q3.Query, q3.OpName, q3.Variables)
	json3, err := json.MarshalIndent(resp3, "", "\t")
	check(err, "json.MarshalIndent")
	fmt.Println(string(json3))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"notes": [
	// 			{
	// 				"noteID": "n-b2c043",
	// 				"data": "Olá Mundo!"
	// 			},
	// 			{
	// 				"noteID": "n-95d818",
	// 				"data": "Olá novamente, mundo!"
	// 			},
	// 			{
	// 				"noteID": "n-80a459",
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
			"noteID": "n-b2c043",
		},
	}
	resp4 := Schema.Exec(ctx, q4.Query, q4.OpName, q4.Variables)
	json4, err := json.MarshalIndent(resp4, "", "\t")
	check(err, "json.MarshalIndent")
	fmt.Println(string(json4))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"note": {
	// 			"noteID": "n-b2c043",
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
			"userID": "u-33e723",
			"note": JSON{
				"data": "We created a note!",
			},
		},
	}
	resp5 := Schema.Exec(ctx, q5.Query, q5.OpName, q5.Variables)
	json5, err := json.MarshalIndent(resp5, "", "\t")
	check(err, "json.MarshalIndent")
	fmt.Println(string(json5))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"createNote": {
	// 			"noteID": "n-fdb06b",
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
				notes {
					noteID
					data
				}
			}
		}`,
		Variables: nil,
	}
	resp6 := Schema.Exec(ctx, q6.Query, q6.OpName, q6.Variables)
	json6, err := json.MarshalIndent(resp6, "", "\t")
	check(err, "json.MarshalIndent")
	fmt.Println(string(json6))
	// Expected output:
	//
	// {
	// 	"data": {
	// 		"users": [
	// 			// ...
	// 			{
	// 				"userID": "u-33e723",
	// 				"username": "zaydek",
	// 				"notes": [
	// 					// ...
	// 					{
	// 						"noteID": "n-7ccbdf",
	// 						"data": "We created a note!"
	// 					}
	// 				]
	// 			}
	// 		]
	// 	}
	// }
}
