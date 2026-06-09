/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dgraph-io/dgo/v250/protos/api"
	"github.com/dgraph-io/dgraph/v25/dgraphtest"
	"github.com/stretchr/testify/require"
)

// TestIssue9422 tests that multiple mutations (delete + set) in the same transaction
// don't create duplicate JSON fields for single-valued predicates.
// See: https://github.com/dgraph-io/dgraph/issues/9422
func (ssuite *SystestTestSuite) TestIssue9422() {
	t := ssuite.T()
	// Skip on older versions if this bug existed before a certain commit
	// dgraphtest.ShouldSkipTest(t, "COMMIT_SHA", ssuite.dc.GetVersion())
	
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

	// Set up initial data: Tom likes apple
	mu := &api.Mutation{
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
	}
	assigned, err := gcli.NewTxn().Mutate(ctx, mu)
	require.NoError(t, err)
	_ = assigned

	// Verify initial state
	query1 := `{
		q(func: eq(name, "Tom")) {
			uid
			like {
				uid
				fruit
			}
		}
	}`
	resp, err := gcli.NewReadOnlyTxn().Query(ctx, query1)
	require.NoError(t, err)
	var result1 struct {
		Q []struct {
			UID  string `json:"uid"`
			Like *struct {
				UID   string `json:"uid"`
				Fruit string `json:"fruit"`
			} `json:"like"`
		} `json:"q"`
	}
	require.NoError(t, json.Unmarshal(resp.GetJson(), &result1))
	require.Equal(t, 1, len(result1.Q))
	require.NotNil(t, result1.Q[0].Like)
	require.Equal(t, "apple", result1.Q[0].Like.Fruit)
	personUID := result1.Q[0].UID

	// Now delete all likes and set a new one (banana) in the same transaction
	query2 := `{
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
		Query:     query2,
		Mutations: []*api.Mutation{mu1, mu2},
		CommitNow: true,
	}
	_, err = gcli.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	// Query again - should only have banana, not apple
	resp, err = gcli.NewReadOnlyTxn().Query(ctx, query1)
	require.NoError(t, err)
	
	// Check that the JSON is valid (no duplicate keys)
	var rawResult map[string]interface{}
	err = json.Unmarshal(resp.GetJson(), &rawResult)
	require.NoError(t, err, "JSON should be valid without duplicate keys")
	
	var result2 struct {
		Q []struct {
			UID  string `json:"uid"`
			Like *struct {
				UID   string `json:"uid"`
				Fruit string `json:"fruit"`
			} `json:"like"`
		} `json:"q"`
	}
	require.NoError(t, json.Unmarshal(resp.GetJson(), &result2))
	require.Equal(t, 1, len(result2.Q))
	require.Equal(t, personUID, result2.Q[0].UID)
	require.NotNil(t, result2.Q[0].Like, "Person should still have a like edge")
	require.Equal(t, "banana", result2.Q[0].Like.Fruit, "Person should like banana, not apple")
	
	// Verify that JSON doesn't contain both apple and banana
	jsonStr := string(resp.GetJson())
	require.NotContains(t, jsonStr, "apple", "Result should not contain the old value (apple)")
	require.Contains(t, jsonStr, "banana", "Result should contain the new value (banana)")
	
	t.Log("Test passed: Single-valued predicate correctly shows only the new value after delete+set")
}
