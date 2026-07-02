/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package posting

import (
	"context"
	"testing"

	"github.com/dgraph-io/dgraph/v25/protos/pb"
	"github.com/dgraph-io/dgraph/v25/schema"
	"github.com/dgraph-io/dgraph/v25/x"
	"github.com/stretchr/testify/require"
)

// TestIssue9422 reproduces and tests the fix for issue #9422:
// When using multiple mutations in a single transaction where the first mutation 
// deletes an edge and the second mutation creates a new edge, the result should
// only contain the new edge, not both the old and new edges.
func TestIssue9422(t *testing.T) {
	ctx := context.Background()
	
	// Set up schema for a non-list uid predicate (like <like>: uid @reverse .)
	err := schema.ParseBytes([]byte(`like: uid .`), 1)
	require.NoError(t, err)
	
	personUID := uint64(1)
	appleUID := uint64(2)
	bananaUID := uint64(3)
	
	key := x.DataKey(x.AttrInRootNamespace("like"), personUID)
	
	// Create a new posting list
	ol := &List{
		key:   key,
		plist: &pb.PostingList{},
	}
	
	// Initial setup: person likes apple
	txn1 := &Txn{StartTs: 1}
	initialEdge := &pb.DirectedEdge{
		Entity:    personUID,
		Attr:      "like",
		ValueId:   appleUID,
		ValueType: pb.Posting_UID,
		Op:        pb.DirectedEdge_SET,
	}
	err = ol.addMutation(ctx, txn1, initialEdge)
	require.NoError(t, err)
	require.NoError(t, ol.commitMutation(1, 2))
	
	// Verify initial state: person likes apple
	uids, err := ol.Uids(ListOptions{ReadTs: 3})
	require.NoError(t, err)
	require.Equal(t, []uint64{appleUID}, uids.Uids, "Initially, person should like apple")
	
	// Transaction with two mutations:
	// 1. Delete all likes: uid(person) <like> * .
	// 2. Set new like: uid(person) <like> uid(banana) .
	txn2 := &Txn{StartTs: 4}
	
	// Mutation 1: Explicit delete-all (uid(person) <like> * .)
	delAllEdge := &pb.DirectedEdge{
		Entity:    personUID,
		Attr:      "like",
		Value:     []byte(x.Star), // This represents the * in "uid(person) <like> * ."
		ValueType: pb.Posting_DEFAULT,
		Op:        pb.DirectedEdge_DEL,
	}
	err = ol.addMutation(ctx, txn2, delAllEdge)
	require.NoError(t, err)
	
	// Mutation 2: Set new edge (uid(person) <like> uid(banana) .)
	setNewEdge := &pb.DirectedEdge{
		Entity:    personUID,
		Attr:      "like",
		ValueId:   bananaUID,
		ValueType: pb.Posting_UID,
		Op:        pb.DirectedEdge_SET,
	}
	err = ol.addMutation(ctx, txn2, setNewEdge)
	require.NoError(t, err)
	
	// Commit the transaction
	require.NoError(t, ol.commitMutation(4, 5))
	
	// Query after commit: should ONLY return banana, NOT apple
	// This is the bug that was reported in issue #9422
	uids, err = ol.Uids(ListOptions{ReadTs: 6})
	require.NoError(t, err)
	
	// The fix ensures that the explicit delete-all is preserved,
	// so the result should only contain bananaUID
	require.Equal(t, []uint64{bananaUID}, uids.Uids, 
		"After delete-all + set, person should only like banana, not apple")
	require.NotContains(t, uids.Uids, appleUID, 
		"Apple should not be in the result after being deleted")
}
