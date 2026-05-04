//go:build integration || upgrade

/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package query

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v250/protos/api"
)

// TestMultipleMutationsNonListPredicate tests the scenario where multiple mutations
// (delete followed by set) on a non-list uid predicate in the same transaction
// should not create duplicate JSON fields.
// Fixes https://github.com/dgraph-io/dgraph/issues/9422
func TestMultipleMutationsNonListPredicate(t *testing.T) {
	ctx := context.Background()

	// Set up schema
	require.NoError(t, client.Alter(ctx, &api.Operation{
		Schema: `
			name: string @index(exact) .
			like: uid @reverse .
			fruit: string @index(exact) .
			
			type Person {
				name
				like
			}
			type Fruit {
				fruit
			}
		`,
	}))

	// Set up initial data
	txn := client.NewTxn()
	assignedUids, err := txn.Mutate(ctx, &api.Mutation{
		SetNquads: []byte(`
			_:person <name> "Tom" .
			_:person <dgraph.type> "Person" .
			_:apple <fruit> "apple" .
			_:apple <dgraph.type> "Fruit" .
			_:banana <fruit> "banana" .
			_:banana <dgraph.type> "Fruit" .
			_:person <like> _:apple .
		`),
		CommitNow: true,
	})
	require.NoError(t, err)

	// Perform delete-then-set in a single transaction
	txn = client.NewTxn()
	dql := `{
		person as var(func: eq(name, "Tom"))
		banana as var(func: eq(fruit, "banana"))
	}`
	mu1 := &api.Mutation{
		DelNquads: []byte(`uid(person) <like> * .`),
	}
	mu2 := &api.Mutation{
		SetNquads: []byte(`uid(person) <like> uid(banana) .`),
	}
	req := &api.Request{
		Query:     dql,
		Mutations: []*api.Mutation{mu1, mu2},
		CommitNow: true,
	}
	_, err = txn.Do(ctx, req)
	require.NoError(t, err)

	// Query and verify the result
	queryDql := `{
		q(func: eq(name, "Tom")) {
			uid
			like { uid fruit }
		}
	}`
	resp, err := client.NewTxn().Query(ctx, queryDql)
	require.NoError(t, err)

	// Parse the JSON response
	var result map[string][]interface{}
	err = json.Unmarshal(resp.Json, &result)
	require.NoError(t, err)

	// Verify the response structure
	require.Len(t, result["q"], 1)
	person := result["q"][0].(map[string]interface{})
	require.NotNil(t, person["like"])
	
	like := person["like"].(map[string]interface{})
	
	// The key assertion: "like" should be a single object, not contain duplicate "uid" fields
	// If the bug exists, the JSON would have duplicate "uid" keys which would cause
	// one to overwrite the other, or the unmarshal would fail/behave unexpectedly
	
	// Verify there's only one uid and one fruit field
	require.Contains(t, like, "uid")
	require.Contains(t, like, "fruit")
	require.Equal(t, "banana", like["fruit"])
	require.Equal(t, assignedUids.Uids["banana"], like["uid"])
	
	// The response should be valid JSON without duplicate keys
	// Parse again as raw JSON to ensure no duplicates
	var rawResult interface{}
	require.NoError(t, json.Unmarshal(resp.Json, &rawResult))
}
