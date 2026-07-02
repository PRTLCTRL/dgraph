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

// TestDeleteAllThenSetWithCommit tests the scenario where:
// 1. Initial edge exists (committed)
// 2. Delete-all followed by set in a new transaction (committed)
// 3. Query in a third transaction should only see the new edge
// This more closely reproduces issue #9422.
func TestDeleteAllThenSetWithCommit(t *testing.T) {
	ctx := context.Background()
	
	// Set up schema for a non-list uid predicate
	err := schema.ParseBytes([]byte(`friend: uid .`), 1)
	require.NoError(t, err)
	
	key := x.DataKey(x.AttrInRootNamespace("friend"), 1)
	
	// Create a new posting list
	ol := &List{
		key:   key,
		plist: &pb.PostingList{},
	}
	
	// Transaction 1: Add an initial edge (1 -> 2, representing person -> apple)
	txn1 := &Txn{StartTs: 1}
	edge1 := &pb.DirectedEdge{
		Entity:    1,
		Attr:      "friend",
		Value:     nil,
		ValueId:   2,
		ValueType: pb.Posting_UID,
		Op:        pb.DirectedEdge_SET,
	}
	err = ol.addMutation(ctx, txn1, edge1)
	require.NoError(t, err)
	require.NoError(t, ol.commitMutation(1, 2))
	
	// Verify we can read the edge
	uids, err := ol.Uids(ListOptions{ReadTs: 3})
	require.NoError(t, err)
	require.Equal(t, []uint64{2}, uids.Uids, "Should have edge to UID 2 (apple)")
	
	// Transaction 2: Delete all edges (explicit), then add a new edge (1 -> 3, representing person -> banana)
	txn2 := &Txn{StartTs: 4}
	
	// First: Explicit delete-all with Value="*"
	delEdge := &pb.DirectedEdge{
		Entity:    1,
		Attr:      "friend",
		Value:     []byte(x.Star), // Delete-all marker
		ValueType: pb.Posting_DEFAULT,
		Op:        pb.DirectedEdge_DEL,
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
		Op:        pb.DirectedEdge_SET,
	}
	err = ol.addMutation(ctx, txn2, setEdge)
	require.NoError(t, err)
	
	// Commit transaction 2
	require.NoError(t, ol.commitMutation(4, 5))
	
	// Transaction 3: Query after both transactions are committed
	// This should ONLY return UID 3 (banana), not UID 2 (apple)
	uids, err = ol.Uids(ListOptions{ReadTs: 6})
	require.NoError(t, err)
	
	// This is the bug! We expect [3] but might get [2, 3]
	require.Equal(t, []uint64{3}, uids.Uids, "After commit, should only have edge to UID 3 (banana), not 2 (apple)")
}
