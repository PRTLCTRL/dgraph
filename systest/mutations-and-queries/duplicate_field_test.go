// +build integration || upgrade

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
	"github.com/dgraph-io/dgraph/v25/dgraphapi"
	"github.com/stretchr/testify/require"
)

func (ssuite *SystestTestSuite) TestMultipleMutationsNoInvalidJSON() {
	t := ssuite.T()

	gcli, cleanup, err := doGrpcLogin(ssuite)
	defer cleanup()
	require.NoError(t, err)

	ctx := context.Background()
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

	ssuite.Upgrade()

	gcli, cleanup, err = doGrpcLogin(ssuite)
	defer cleanup()
	require.NoError(t, err)
	ctx = context.Background()

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

	txn = gcli.NewTxn()
	_, err = txn.Do(ctx, req)
	require.NoError(t, err)

	queryDQL := `{
	  q(func: eq(name, "Tom")) {
		uid
		like { uid fruit }
	  }
	}`

	resp, err := gcli.NewReadOnlyTxn().Query(ctx, queryDQL)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(resp.Json, &result)
	require.NoError(t, err, "JSON response should be valid")

	q, ok := result["q"].([]interface{})
	require.True(t, ok)
	require.Len(t, q, 1)

	person, ok := q[0].(map[string]interface{})
	require.True(t, ok)

	like, ok := person["like"].(map[string]interface{})
	require.True(t, ok, "like should be a single object, not a list")

	fruit, ok := like["fruit"].(string)
	require.True(t, ok)
	require.Equal(t, "banana", fruit)

	dgraphapi.CompareJSON(`{"uid":"banana", "fruit":"banana"}`, string(resp.Json))
}
