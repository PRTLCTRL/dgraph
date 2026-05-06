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
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v250/protos/api"
)

func TestMultipleMutationsNoDuplicateJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	schema := `
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
	`
	op := &api.Operation{Schema: schema}
	require.NoError(t, client.Alter(ctx, op))

	txn := client.NewTxn()
	defer func() {
		_ = txn.Discard(ctx)
	}()

	mutation := &api.Mutation{
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

	assigned, err := txn.Mutate(ctx, mutation)
	require.NoError(t, err)
	_ = assigned

	txn = client.NewTxn()
	defer func() {
		_ = txn.Discard(ctx)
	}()

	req := &api.Request{
		Query: `{
			person as var(func: eq(name, "Tom"))
			banana as var(func: eq(fruit, "banana"))
		}`,
		Mutations: []*api.Mutation{
			{
				DelNquads: []byte(`uid(person) <like> * .`),
			},
			{
				SetNquads: []byte(`uid(person) <like> uid(banana) .`),
			},
		},
		CommitNow: true,
	}

	_, err = txn.Do(ctx, req)
	require.NoError(t, err)

	txn = client.NewReadOnlyTxn()
	defer func() {
		_ = txn.Discard(ctx)
	}()

	resp, err := txn.Query(ctx, `{
		q(func: eq(name, "Tom")) {
			uid
			like { uid fruit }
		}
	}`)
	require.NoError(t, err)

	type likeNode struct {
		Uid   string `json:"uid"`
		Fruit string `json:"fruit"`
	}

	type personNode struct {
		Uid  string   `json:"uid"`
		Like likeNode `json:"like"`
	}

	type result struct {
		Q []personNode `json:"q"`
	}

	var res result
	require.NoError(t, json.Unmarshal(resp.Json, &res))

	require.Len(t, res.Q, 1, "Expected one person")
	require.Equal(t, "banana", res.Q[0].Like.Fruit, "Expected person to like banana after mutation")

	require.NotEmpty(t, res.Q[0].Like.Uid, "Expected like to have uid")

	var rawMap map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Json, &rawMap))

	q := rawMap["q"].([]interface{})
	person := q[0].(map[string]interface{})
	like := person["like"].(map[string]interface{})

	uidCount := 0
	for key := range like {
		if key == "uid" {
			uidCount++
		}
	}

	require.Equal(t, 1, uidCount, "Expected exactly one 'uid' field in like object, got duplicate fields")
}
