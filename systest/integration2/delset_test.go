//go:build integration2

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
	"github.com/dgraph-io/dgraph/v25/dgraphtest"
)

func TestMultipleMutationsDelSetSameTransaction(t *testing.T) {
	conf := dgraphtest.NewClusterConfig().WithNumAlphas(1).WithNumZeros(1).WithReplicas(1)
	c, err := dgraphtest.NewLocalCluster(conf)
	require.NoError(t, err)
	defer func() { c.Cleanup(t.Failed()) }()
	require.NoError(t, c.Start())

	gc, cleanup, err := c.Client()
	require.NoError(t, err)
	defer cleanup()

	// Set schema
	schema := `
		<name>: string @index(exact) .
		<like>: uid @reverse .
		<fruit>: string @index(exact) .

		type Person {
			name
			like
		}
		type Fruit {
			fruit
		}
	`
	require.NoError(t, gc.SetupSchema(schema))

	// Set initial data
	mutation := `
		_:person <name> "Tom" .
		_:person <dgraph.type> "Person" .
		_:apple <fruit> "apple" .
		_:apple <dgraph.type> "Fruit" .
		_:banana <fruit> "banana" .
		_:banana <dgraph.type> "Fruit" .
		_:person <like> _:apple .
	`
	txn := gc.NewTxn()
	_, err = txn.Mutate(context.Background(), &api.Mutation{
		SetNquads: []byte(mutation),
		CommitNow: true,
	})
	require.NoError(t, err)

	// Query to verify initial state
	query := `{
		q(func: eq(name, "Tom")) {
			uid
			like {
				uid
				fruit
			}
		}
	}`

	resp, err := gc.NewReadOnlyTxn().Query(context.Background(), query)
	require.NoError(t, err)

	type Result struct {
		Q []struct {
			UID  string `json:"uid"`
			Like struct {
				UID   string `json:"uid"`
				Fruit string `json:"fruit"`
			} `json:"like"`
		} `json:"q"`
	}

	var initialResult Result
	require.NoError(t, json.Unmarshal(resp.Json, &initialResult))
	require.Len(t, initialResult.Q, 1)
	require.Equal(t, "apple", initialResult.Q[0].Like.Fruit)

	// Now perform delete and set in the same transaction
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
	txn = gc.NewTxn()
	_, err = txn.Do(context.Background(), req)
	require.NoError(t, err)

	// Query after mutation
	resp, err = gc.NewReadOnlyTxn().Query(context.Background(), query)
	require.NoError(t, err)

	var finalResult Result
	require.NoError(t, json.Unmarshal(resp.Json, &finalResult))
	require.Len(t, finalResult.Q, 1)
	
	// The bug would cause multiple uid fields in the like object,
	// which is invalid JSON. After the fix, we should only see banana.
	require.Equal(t, "banana", finalResult.Q[0].Like.Fruit)
	
	// Verify that the JSON is valid by checking there's only one UID in the like object
	var rawResult map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Json, &rawResult))
	q := rawResult["q"].([]interface{})
	require.Len(t, q, 1)
	like := q[0].(map[string]interface{})["like"].(map[string]interface{})
	
	// Count the number of "uid" keys - there should be only one
	uidCount := 0
	for key := range like {
		if key == "uid" {
			uidCount++
		}
	}
	require.Equal(t, 1, uidCount, "Expected only one 'uid' field in the like object")
}
