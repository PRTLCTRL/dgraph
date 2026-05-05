/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package posting

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgraph/v25/protos/pb"
	"github.com/dgraph-io/dgraph/v25/x"
)

// TestDeleteAllFollowedBySetInSameTransaction tests the scenario where a delete-all
// is followed by a set operation in the same transaction. This reproduces issue #9422.
func TestDeleteAllFollowedBySetInSameTransaction(t *testing.T) {
	key := x.DataKey(x.AttrInRootNamespace("like"), 1)
	ol, err := GetNoStore(key, math.MaxUint64)
	require.NoError(t, err)
	ol.mutationMap.setTs(1)

	// First, add an edge to uid 2 (representing "apple")
	edge := &pb.DirectedEdge{
		Entity: 1,
		Attr:   "like",
		Value:  nil,
		ValueId: 2,
		Op:     pb.DirectedEdge_SET,
	}
	txn := &Txn{StartTs: 1}
	require.NoError(t, ol.addMutation(context.Background(), txn, edge))
	require.NoError(t, ol.commitMutation(1, 2))

	// Verify we can read the edge
	l, err := ol.Uids(ListOptions{ReadTs: 3})
	require.NoError(t, err)
	require.Equal(t, []uint64{2}, l.Uids)

	// Now in a new transaction, delete all edges and add a new edge to uid 3 (representing "banana")
	ol.mutationMap.setTs(3)
	txn = &Txn{StartTs: 3}
	
	// Delete all
	deleteAllEdge := &pb.DirectedEdge{
		Entity: 1,
		Attr:   "like",
		Value:  []byte(x.Star),
		Op:     pb.DirectedEdge_DEL,
	}
	require.NoError(t, ol.addMutation(context.Background(), txn, deleteAllEdge))

	// Set new edge
	newEdge := &pb.DirectedEdge{
		Entity: 1,
		Attr:   "like",
		Value:  nil,
		ValueId: 3,
		Op:     pb.DirectedEdge_SET,
	}
	require.NoError(t, ol.addMutation(context.Background(), txn, newEdge))

	// Read within the same transaction (before commit)
	// This should only return uid 3, not both 2 and 3
	l, err = ol.Uids(ListOptions{ReadTs: 3})
	require.NoError(t, err)
	require.Equal(t, []uint64{3}, l.Uids, "Should only see the new edge (uid 3), not the deleted one (uid 2)")

	// Commit the transaction
	require.NoError(t, ol.commitMutation(3, 4))

	// Read after commit
	l, err = ol.Uids(ListOptions{ReadTs: 5})
	require.NoError(t, err)
	require.Equal(t, []uint64{3}, l.Uids, "After commit, should only see the new edge (uid 3)")
}

// TestDeleteAllFollowedByMultipleSetsInSameTransaction tests the scenario where a delete-all
// is followed by multiple set operations in the same transaction.
func TestDeleteAllFollowedByMultipleSetsInSameTransaction(t *testing.T) {
	key := x.DataKey(x.AttrInRootNamespace("like"), 2)
	ol, err := GetNoStore(key, math.MaxUint64)
	require.NoError(t, err)
	ol.mutationMap.setTs(1)

	// Add initial edges to uid 2 and 4
	txn := &Txn{StartTs: 1}
	for _, uid := range []uint64{2, 4} {
		edge := &pb.DirectedEdge{
			Entity: 2,
			Attr:   "like",
			Value:  nil,
			ValueId: uid,
			Op:     pb.DirectedEdge_SET,
		}
		require.NoError(t, ol.addMutation(context.Background(), txn, edge))
	}
	require.NoError(t, ol.commitMutation(1, 2))

	// Verify initial state
	l, err := ol.Uids(ListOptions{ReadTs: 3})
	require.NoError(t, err)
	require.Equal(t, []uint64{2, 4}, l.Uids)

	// Now delete all and add new edges to uid 3 and 5
	ol.mutationMap.setTs(3)
	txn = &Txn{StartTs: 3}
	
	// Delete all
	deleteAllEdge := &pb.DirectedEdge{
		Entity: 2,
		Attr:   "like",
		Value:  []byte(x.Star),
		Op:     pb.DirectedEdge_DEL,
	}
	require.NoError(t, ol.addMutation(context.Background(), txn, deleteAllEdge))

	// Set new edges
	for _, uid := range []uint64{3, 5} {
		newEdge := &pb.DirectedEdge{
			Entity: 2,
			Attr:   "like",
			Value:  nil,
			ValueId: uid,
			Op:     pb.DirectedEdge_SET,
		}
		require.NoError(t, ol.addMutation(context.Background(), txn, newEdge))
	}

	// Read within the same transaction
	// Should only see the new edges (3, 5), not the old ones (2, 4)
	l, err = ol.Uids(ListOptions{ReadTs: 3})
	require.NoError(t, err)
	require.Equal(t, []uint64{3, 5}, l.Uids, "Should only see new edges (3, 5), not deleted ones (2, 4)")
}
