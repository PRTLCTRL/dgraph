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

// TestIssue9422 reproduces the bug where multiple mutations (delete-all followed by set)
// in the same transaction result in duplicate UIDs being returned.
func TestIssue9422(t *testing.T) {
	require.NoError(t, pstore.DropAll())

	key := x.DataKey(x.AttrInRootNamespace("like"), 1)
	
	// Initial state: person (uid=1) likes apple (uid=2)
	txn1 := NewTxn(1)
	l, err := txn1.Get(key)
	require.NoError(t, err)
	
	edge := &pb.DirectedEdge{
		Entity:  1,
		Attr:    x.AttrInRootNamespace("like"),
		ValueId: 2,
		Op:      pb.DirectedEdge_SET,
	}
	
	err = l.addMutation(context.Background(), txn1, edge)
	require.NoError(t, err)
	require.NoError(t, l.commitMutation(1, 2)) // commit at ts=2
	
	// Write to disk
	kvs, err := l.Rollup(nil, math.MaxUint64)
	require.NoError(t, err)
	require.NotNil(t, kvs)
	writer := NewTxnWriter(pstore)
	for _, kv := range kvs {
		require.NoError(t, writer.SetAt(kv.Key, kv.Value, BitCompletePosting, kv.Version))
	}
	require.NoError(t, writer.Flush())
	
	// Now in transaction 3: DELETE * followed by SET banana (uid=3)
	txn2 := NewTxn(3)
	
	// Re-load list from disk to simulate a fresh read
	l2, err := readPostingListFromDisk(key, pstore, math.MaxUint64)
	require.NoError(t, err)
	l2.minTs = 2 // Set minTs to the commit timestamp
	
	// Add mutations to the reloaded list
	if l2.mutationMap == nil {
		l2.mutationMap = newMutableLayer()
	}
	
	// DELETE *
	delEdge := &pb.DirectedEdge{
		Entity: 1,
		Attr:   x.AttrInRootNamespace("like"),
		Value:  []byte(x.Star),
		Op:     pb.DirectedEdge_DEL,
	}
	err = l2.addMutation(context.Background(), txn2, delEdge)
	require.NoError(t, err)
	
	// SET banana (uid=3)
	setEdge := &pb.DirectedEdge{
		Entity:  1,
		Attr:    x.AttrInRootNamespace("like"),
		ValueId: 3,
		Op:      pb.DirectedEdge_SET,
	}
	err = l2.addMutation(context.Background(), txn2, setEdge)
	require.NoError(t, err)
	
	// Commit transaction 3
	require.NoError(t, l2.commitMutation(3, 4)) // commit at ts=4
	
	// Query at ts=4 should show ONLY banana (uid=3), not both apple and banana
	uidList, err := l2.Uids(ListOptions{ReadTs: 4})
	require.NoError(t, err)
	
	t.Logf("UIDs returned after delete-all and set: %v", uidList.Uids)
	
	// This is the bug: it returns [2, 3] instead of just [3]
	require.Equal(t, []uint64{3}, uidList.Uids, "After delete-all and set, should only have UID 3 (banana), not both 2 and 3")
}
