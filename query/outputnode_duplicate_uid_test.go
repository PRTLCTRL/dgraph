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

func TestAddMapChildDuplicateUID(t *testing.T) {
	enc := newEncoder()
	root := enc.newNode(enc.idForAttr("root"))

	likeAttr := enc.idForAttr("like")
	fruitAttr := enc.idForAttr("fruit")

	// First add an apple node (simulating the initial edge)
	appleNode := enc.newNode(likeAttr)
	require.NoError(t, enc.SetUID(appleNode, 0x2, enc.uidAttr))
	require.NoError(t, enc.AddValue(appleNode, fruitAttr, types.Val{Tid: types.StringID, Value: "apple"}))
	enc.AddMapChild(root, appleNode)

	// Then add a banana node (simulating the new edge after delete+set mutation)
	bananaNode := enc.newNode(likeAttr)
	require.NoError(t, enc.SetUID(bananaNode, 0x3, enc.uidAttr))
	require.NoError(t, enc.AddValue(bananaNode, fruitAttr, types.Val{Tid: types.StringID, Value: "banana"}))
	enc.AddMapChild(root, bananaNode)

	enc.fixOrder(root)
	enc.buf.Reset()
	require.NoError(t, enc.encode(root))

	jsonOutput := enc.buf.String()

	var result map[string]interface{}
	err := json.Unmarshal(enc.buf.Bytes(), &result)
	require.NoError(t, err, "JSON should be valid")

	likeVal, ok := result["like"]
	require.True(t, ok, "should have 'like' field")

	likeMap, ok := likeVal.(map[string]interface{})
	require.True(t, ok, "like should be a map")

	require.Equal(t, "0x3", likeMap["uid"], "should have the most recent uid (banana)")
	require.Equal(t, "banana", likeMap["fruit"], "should have the most recent fruit (banana)")
	require.Len(t, likeMap, 2, "should only have two fields: uid and fruit")

	require.NotContains(t, jsonOutput, "0x2", "should not contain old uid (apple)")
	require.NotContains(t, jsonOutput, "apple", "should not contain old fruit value")
}
