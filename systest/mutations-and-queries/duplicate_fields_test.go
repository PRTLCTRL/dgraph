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

func (ssuite *SystestTestSuite) TestMultipleMutationsNoDuplicateFields() {
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

	q1 := result1["q"].([]interface{})
	require.Len(t, q1, 1)
	person := q1[0].(map[string]interface{})
	like := person["like"].(map[string]interface{})
	require.Equal(t, "apple", like["fruit"])

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

	resp2, err := gcli.NewReadOnlyTxn().Query(ctx, query1)
	require.NoError(t, err)

	var result2 map[string]interface{}
	err = json.Unmarshal(resp2.Json, &result2)
	require.NoError(t, err)

	q2 := result2["q"].([]interface{})
	require.Len(t, q2, 1)
	person2 := q2[0].(map[string]interface{})
	like2 := person2["like"].(map[string]interface{})

	keys := make([]string, 0, len(like2))
	for k := range like2 {
		keys = append(keys, k)
	}
	
	uidCount := 0
	fruitCount := 0
	for _, k := range keys {
		if k == "uid" {
			uidCount++
		}
		if k == "fruit" {
			fruitCount++
		}
	}
	
	require.Equal(t, 1, uidCount, "Expected exactly one 'uid' field in like object")
	require.Equal(t, 1, fruitCount, "Expected exactly one 'fruit' field in like object")
	require.Equal(t, "banana", like2["fruit"], "Expected fruit to be banana after mutation")

	jsonBytes, err := json.Marshal(like2)
	require.NoError(t, err)
	
	var testDecode map[string]interface{}
	err = json.Unmarshal(jsonBytes, &testDecode)
	require.NoError(t, err, "JSON should be valid and decodable without duplicate keys")
}
