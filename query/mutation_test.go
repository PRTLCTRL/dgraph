//go:build integration || upgrade

/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v250/protos/api"
	"github.com/dgraph-io/dgraph/v25/dgraphtest"
)

func TestReservedPredicateForMutation(t *testing.T) {
	err := addTriplesToCluster(`_:x <dgraph.graphql.schema> "df"`)
	require.Error(t, err, "Cannot mutate graphql reserved predicate dgraph.graphql.schema")
}

func TestAlteringReservedTypesAndPredicatesShouldFail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	op := &api.Operation{Schema: `
		type dgraph.Person {
			name: string
			age: int
		}
		name: string .
		age: int .
	`}
	err := client.Alter(context.Background(), op)
	require.Error(t, err, "altering type in dgraph namespace shouldn't have succeeded")
	require.Contains(t, err.Error(), "Can't alter type `dgraph.Person` as it is prefixed with "+
		"`dgraph.` which is reserved as the namespace for dgraph's internal types/predicates.")

	op = &api.Operation{Schema: `
		type Person {
			dgraph.name
			age
		}
		dgraph.name: string .
		age: int .
	`}
	err = client.Alter(ctx, op)
	require.Error(t, err, "altering predicate in dgraph namespace shouldn't have succeeded")
	require.Contains(t, err.Error(), "Can't alter predicate `dgraph.name` as it is prefixed with "+
		"`dgraph.` which is reserved as the namespace for dgraph's internal types/predicates.")

	_, err = client.NewTxn().Mutate(ctx, &api.Mutation{
		SetNquads: []byte(`_:new <dgraph.name> "Alice" .`),
	})
	require.Error(t, err, "storing predicate in dgraph namespace shouldn't have succeeded")
	require.Contains(t, err.Error(), "Can't store predicate `dgraph.name` as it is prefixed with "+
		"`dgraph.` which is reserved as the namespace for dgraph's internal types/predicates.")
}

func TestUnreservedPredicateForDeletion(t *testing.T) {
	// This bug was fixed in commit 8631dab37c951b288f839789bbabac5e7088b58f
	dgraphtest.ShouldSkipTest(t, "8631dab37c951b288f839789bbabac5e7088b58f", dc.GetVersion())

	grootUserQuery := `
	{
		grootUser(func:eq(dgraph.xid, "groot")){
			uid
		}
	}`
	type userQryResp struct {
		GrootUser []struct {
			Uid string `json:"uid"`
		} `json:"grootUser"`
	}
	resp, err := client.Query(grootUserQuery)
	require.NoError(t, err, "groot user query failed")

	var userResp userQryResp
	require.NoError(t, json.Unmarshal(resp.GetJson(), &userResp))
	grootsUidStr := userResp.GrootUser[0].Uid
	grootsUidInt, err := strconv.ParseUint(grootsUidStr, 0, 64)
	require.NoError(t, err)
	require.Greater(t, uint64(grootsUidInt), uint64(0))

	const testSchema = `
		type Star {
			name
		}

		name : string @index(term, exact, trigram) @count @lang .`
	setSchema(testSchema)

	triple := fmt.Sprintf(`<%s> <dgraphcore_355> "Betelgeuse" .`, grootsUidStr)
	require.NoError(t, addTriplesToCluster(triple))
	deleteTriplesInCluster(triple)
}

func TestMultipleMutationsNoDuplicateJSON(t *testing.T) {
	const schema = `
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
	setSchema(schema)
	
	ctx := context.Background()
	
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
	
	txn := client.NewTxn()
	assigned, err := txn.Mutate(ctx, mu)
	require.NoError(t, err)
	
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
	
	txn = client.NewTxn()
	_, err = txn.Do(ctx, req)
	require.NoError(t, err)
	
	queryDql := `{
		q(func: eq(name, "Tom")) {
			uid
			like { uid fruit }
		}
	}`
	
	resp, err := client.NewReadOnlyTxn().Query(ctx, queryDql)
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
	
	var result Result
	err = json.Unmarshal(resp.Json, &result)
	require.NoError(t, err, "JSON should be valid without duplicate fields")
	require.Len(t, result.Q, 1)
	require.Equal(t, "banana", result.Q[0].Like.Fruit, "Should only have banana, not apple")
	require.Equal(t, assigned.Uids["banana"], result.Q[0].Like.UID, "UID should match banana")
	
	var rawResult map[string]interface{}
	err = json.Unmarshal(resp.Json, &rawResult)
	require.NoError(t, err, "JSON should be valid")
	
	q := rawResult["q"].([]interface{})
	require.Len(t, q, 1)
	qItem := q[0].(map[string]interface{})
	like := qItem["like"].(map[string]interface{})
	
	require.Contains(t, like, "uid", "like should have uid field")
	require.Contains(t, like, "fruit", "like should have fruit field")
	require.Len(t, like, 2, "like should have exactly 2 fields, not duplicates")
}
