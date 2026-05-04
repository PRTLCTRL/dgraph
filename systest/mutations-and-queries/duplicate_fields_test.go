//go:build integration || upgrade

/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v250/protos/api"
)

// TestMultipleMutationsNoduplicateFields tests that when multiple mutations
// in a single transaction delete and create edges, the resulting JSON doesn't
// contain duplicate fields for non-list predicates.
// This is a regression test for issue https://github.com/dgraph-io/dgraph/issues/9422
func (ssuite *SystestTestSuite) TestMultipleMutationsNoDuplicateFields() {
	t := ssuite.T()

	gcli, cleanup, err := doGrpcLogin(ssuite)
	defer cleanup()
	require.NoError(t, err)

	ctx := context.Background()

	// Set up schema
	op := &api.Operation{
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
	}
	require.NoError(t, gcli.Alter(ctx, op))

	// Initial data: person likes apple
	txn := gcli.NewTxn()
	_, err = txn.Mutate(ctx, &api.Mutation{
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

	// Query to verify initial state
	query1 := `{
		q(func: eq(name, "Tom")) {
			uid
			like { uid fruit }
		}
	}`
	resp, err := gcli.NewReadOnlyTxn().Query(ctx, query1)
	require.NoError(t, err)

	var result1 map[string]interface{}
	err = json.Unmarshal(resp.Json, &result1)
	require.NoError(t, err)

	// Verify Tom likes apple
	qResults := result1["q"].([]interface{})
	require.Len(t, qResults, 1)
	person := qResults[0].(map[string]interface{})
	like := person["like"].(map[string]interface{})
	require.Equal(t, "apple", like["fruit"])

	// Now perform the problematic mutations: delete all likes, then add banana
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
	txn2 := gcli.NewTxn()
	_, err = txn2.Do(ctx, req)
	require.NoError(t, err)

	// Query again to check the result
	query2 := `{
		q(func: eq(name, "Tom")) {
			uid
			like { uid fruit }
		}
	}`
	resp2, err := gcli.NewReadOnlyTxn().Query(ctx, query2)
	require.NoError(t, err)

	// Parse JSON to validate structure
	var result2 map[string]interface{}
	err = json.Unmarshal(resp2.Json, &result2)
	require.NoError(t, err)

	// Verify only banana is present, not apple
	qResults2 := result2["q"].([]interface{})
	require.Len(t, qResults2, 1)
	person2 := qResults2[0].(map[string]interface{})
	like2 := person2["like"].(map[string]interface{})
	
	// The bug manifests as duplicate "uid" and "fruit" keys in the like object
	// Valid JSON can't have duplicate keys, but Go's json.Unmarshal will only
	// keep the last value. We need to check the raw JSON string for duplicates.
	jsonStr := string(resp2.Json)
	t.Logf("Response JSON: %s", jsonStr)
	
	// Check that the like object only has ONE uid field
	require.Equal(t, "banana", like2["fruit"], "Expected person to like banana after mutation")
	
	// Additional validation: ensure apple is not in the response
	require.NotContains(t, jsonStr, "apple", "Apple should not appear in response after deletion")
}
