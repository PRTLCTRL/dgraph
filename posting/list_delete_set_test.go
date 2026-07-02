/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package posting

import (
	"context"
	"testing"

	"github.com/dgraph-io/dgraph/v25/protos/pb"
	"github.com/dgraph-io/dgraph/v25/x"
	"github.com/stretchr/testify/require"
)

// TestDeleteAllFollowedBySet tests the scenario where a delete-all is followed by a set
// in the same transaction. This reproduces issue #9422.
func TestDeleteAllFollowedBySet(t *testing.T) {
	ctx := context.Background()
	key := x.DataKey(x.AttrInRootNamespace("friend"), 1)
	
	// Create a new posting list
	ol := &List{
		key:   key,
		plist: &pb.PostingList{},
	}
	
	// Transaction 1: Add an initial edge (1 -> 2)
	txn1 := &Txn{StartTs: 1}
	edge1 := &pb.DirectedEdge{
		Entity:    1,
		Attr:      "friend",
		Value:     nil,
		ValueId:   2,
		ValueType: pb.Posting_UID,
	}
	err := ol.addMutation(ctx, txn1, edge1)
	require.NoError(t, err)
	require.NoError(t, ol.commitMutation(1, 2))
	
	// Verify we can read the edge
	uids, err := ol.Uids(ListOptions{ReadTs: 3})
	require.NoError(t, err)
	require.Equal(t, []uint64{2}, uids.Uids, "Should have edge to UID 2")
	
	// Transaction 2: Delete all edges, then add a new edge (1 -> 3)
	txn2 := &Txn{StartTs: 4}
	
	// First: Delete all
	delEdge := &pb.DirectedEdge{
		Entity: 1,
		Attr:   "friend",
		Value:  []byte(x.Star), // Delete-all marker
		Op:     pb.DirectedEdge_DEL,
	}
	err = ol.addMutation(ctx, txn2, delEdge)
	require.NoError(t, err)
	
	// Second: Set new edge
	setEdge := &pb.DirectedEdge{
		Entity:    1,
		Attr:      "friend",
		Value:     nil,
		ValueId:   3,
		ValueType: pb.Posting_UID,
	}
	err = ol.addMutation(ctx, txn2, setEdge)
	require.NoError(t, err)
	
	// Read at transaction 2's timestamp (before commit)
	uids, err = ol.Uids(ListOptions{ReadTs: 4})
	require.NoError(t, err)
	require.Equal(t, []uint64{3}, uids.Uids, "Should only have edge to UID 3, not 2")
	
	// Commit transaction 2
	require.NoError(t, ol.commitMutation(4, 5))
	
	// Read after commit
	uids, err = ol.Uids(ListOptions{ReadTs: 6})
	require.NoError(t, err)
	require.Equal(t, []uint64{3}, uids.Uids, "After commit, should only have edge to UID 3, not 2")
}
