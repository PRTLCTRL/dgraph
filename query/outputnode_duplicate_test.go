/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package query

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgraph/v25/types"
)

// TestAddMapChildReplacement tests that AddMapChild replaces existing children
// with the same attribute instead of merging them. This prevents duplicate fields
// in JSON when mutations delete and set an edge in the same transaction.
// See issue: https://github.com/dgraph-io/dgraph/issues/9422
func TestAddMapChildReplacement(t *testing.T) {
	enc := newEncoder()

	// Create a parent node representing a person
	person := enc.newNode(enc.idForAttr("person"))

	// Create first child representing "like" -> apple
	like1 := enc.newNode(enc.idForAttr("like"))
	require.NoError(t, enc.SetUID(like1, 0x2, enc.uidAttr))
	fruitVal1 := types.Val{Tid: types.StringID, Value: "apple"}
	require.NoError(t, enc.AddValue(like1, enc.idForAttr("fruit"), fruitVal1))

	// Add first "like" child
	enc.AddMapChild(person, like1)

	// Create second child representing "like" -> banana
	like2 := enc.newNode(enc.idForAttr("like"))
	require.NoError(t, enc.SetUID(like2, 0x3, enc.uidAttr))
	fruitVal2 := types.Val{Tid: types.StringID, Value: "banana"}
	require.NoError(t, enc.AddValue(like2, enc.idForAttr("fruit"), fruitVal2))

	// Add second "like" child - this should replace the first one
	enc.AddMapChild(person, like2)

	// Encode to JSON
	enc.fixOrder(person)
	require.NoError(t, enc.encode(person))

	// Parse the JSON to verify it's valid and has only one uid field
	var result map[string]interface{}
	err := json.Unmarshal(enc.buf.Bytes(), &result)
	require.NoError(t, err, "JSON should be valid")

	// Check that "like" exists
	like, ok := result["like"].(map[string]interface{})
	require.True(t, ok, "like should be an object")

	// Check that there's only one uid (the latest one)
	uid, ok := like["uid"].(string)
	require.True(t, ok, "uid should exist and be a string")
	require.Equal(t, "0x3", uid, "uid should be the latest value (banana)")

	// Check that fruit is "banana"
	fruit, ok := like["fruit"].(string)
	require.True(t, ok, "fruit should exist and be a string")
	require.Equal(t, "banana", fruit, "fruit should be banana")

	// Verify JSON doesn't contain apple
	jsonStr := enc.buf.String()
	require.NotContains(t, jsonStr, "apple", "JSON should not contain the old value 'apple'")
	require.NotContains(t, jsonStr, "0x2", "JSON should not contain the old uid '0x2'")
}

// TestAddMapChildMergeScenario tests that AddMapChild works correctly for
// scenarios where we add children to an existing object (not replacing).
func TestAddMapChildMergeScenario(t *testing.T) {
	enc := newEncoder()

	// Create a parent node
	parent := enc.newNode(enc.idForAttr("parent"))

	// Create first child "name"
	child1 := enc.newNode(enc.idForAttr("name"))
	nameVal := types.Val{Tid: types.StringID, Value: "Alice"}
	require.NoError(t, enc.AddValue(child1, enc.idForAttr("value"), nameVal))
	enc.AddMapChild(parent, child1)

	// Create second child "age" (different attribute)
	child2 := enc.newNode(enc.idForAttr("age"))
	ageVal := types.Val{Tid: types.IntID, Value: int64(30)}
	require.NoError(t, enc.AddValue(child2, enc.idForAttr("value"), ageVal))
	enc.AddMapChild(parent, child2)

	// Encode to JSON
	enc.fixOrder(parent)
	require.NoError(t, enc.encode(parent))

	// Parse the JSON
	var result map[string]interface{}
	err := json.Unmarshal(enc.buf.Bytes(), &result)
	require.NoError(t, err, "JSON should be valid")

	// Both children should exist
	require.Contains(t, result, "name", "name should exist")
	require.Contains(t, result, "age", "age should exist")
}
