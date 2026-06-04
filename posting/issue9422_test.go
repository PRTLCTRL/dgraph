/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package posting

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgraph/v25/protos/pb"
	"github.com/dgraph-io/dgraph/v25/x"
)

// TestDeleteAllThenSetInSameTransaction reproduces issue #9422
// When a transaction deletes all edges (uid(person) <like> *) and then
// sets a new edge (uid(person) <like> uid(banana)), the query should only
// return the new edge, not both old and new edges.
func TestDeleteAllThenSetInSameTransaction(t *testing.T) {
	key := x.DataKey(x.AttrInRootNamespace("like"), 1)
	
	// Setup: Add initial edge (person likes apple) and commit it
	txn1 := NewTxn(1)
	l, err := txn1.Get(key)
	require.NoError(t, err)
	
	edge := &pb.DirectedEdge{
		Entity: 1,
		Attr:   "like",
		Value:  []byte{},
		ValueId: 2, // apple
	}
	addMutationHelper(t, l, edge, Set, txn1)
	require.NoError(t, l.commitMutation(1, 2))
	
	// Verify apple is there after commit
	uids := listToArray(t, 0, l, 3)
	require.Equal(t, []uint64{2}, uids, "Should see apple before deletion")

	// Transaction 3: Delete all edges, then set new edge in same transaction
	txn3 := NewTxn(5)
	
	// Delete all edges
	edge.Value = []byte(x.Star)
	addMutationHelper(t, l, edge, Del, txn3)
	
	// Set new edge (person likes banana)
	edge.Value = []byte{}
	edge.ValueId = 3 // banana
	addMutationHelper(t, l, edge, Set, txn3)

	// Query within the same transaction (before commit)
	// Should only see banana (3), not apple (2)
	uids = listToArray(t, 0, l, 5)
	t.Logf("UIDs in transaction (before commit): %v", uids)
	require.Equal(t, []uint64{3}, uids, "Within transaction, should only see banana")

	// Commit and query after commit
	require.NoError(t, l.commitMutation(5, 6))

	uids = listToArray(t, 0, l, 7)
	t.Logf("UIDs after commit: %v", uids)
	require.Equal(t, []uint64{3}, uids, "After commit, should only see banana, not apple")
}

// TestDeleteAllThenSetReturnsOnlyNewValue ensures that after committing
// a delete-all + set in the same transaction, the posting list correctly
// reflects only the new value.
func TestDeleteAllThenSetReturnsOnlyNewValue(t *testing.T) {
	key := x.DataKey(x.AttrInRootNamespace("like2"), 1)
	
	// Add apple and commit
	txn1 := NewTxn(1)
	l, err := txn1.Get(key)
	require.NoError(t, err)
	
	edge := &pb.DirectedEdge{Entity: 1, Attr: "like2", ValueId: 2}
	addMutationHelper(t, l, edge, Set, txn1)
	require.NoError(t, l.commitMutation(1, 2))
	
	// Verify apple is there
	uids := listToArray(t, 0, l, 3)
	require.Equal(t, []uint64{2}, uids)

	// In a new transaction, delete all and add banana
	txn2 := NewTxn(5)
	
	// Delete all
	edge.Value = []byte(x.Star)
	addMutationHelper(t, l, edge, Del, txn2)
	
	// Add banana
	edge.Value = []byte{}
	edge.ValueId = 3
	addMutationHelper(t, l, edge, Set, txn2)
	
	require.NoError(t, l.commitMutation(5, 6))

	// Query after commit - should only see banana
	uids = listToArray(t, 0, l, 7)
	t.Logf("UIDs after delete-all + set: %v", uids)
	require.Equal(t, []uint64{3}, uids, "Should only see banana (3), not apple (2)")
}
