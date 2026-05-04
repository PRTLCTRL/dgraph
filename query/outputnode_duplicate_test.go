/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package query

import (
	"testing"

	"github.com/dgraph-io/dgraph/v25/testutil"
	"github.com/dgraph-io/dgraph/v25/types"
	"github.com/stretchr/testify/require"
)

func TestAddMapChildReplacesNotMerges(t *testing.T) {
	enc := newEncoder()
	defer func() {
		arenaPool.Put(enc.arena)
		enc.alloc.Release()
	}()

	root := enc.newNode(enc.idForAttr("root"))

	likeAttr := enc.idForAttr("like")
	uidAttr := enc.idForAttr("uid")

	firstChild := enc.newNode(likeAttr)
	err := enc.SetUID(firstChild, 0x2, uidAttr)
	require.NoError(t, err)
	err = enc.AddValue(firstChild, enc.idForAttr("fruit"), types.Val{
		Tid:   types.StringID,
		Value: "apple",
	})
	require.NoError(t, err)

	secondChild := enc.newNode(likeAttr)
	err = enc.SetUID(secondChild, 0x3, uidAttr)
	require.NoError(t, err)
	err = enc.AddValue(secondChild, enc.idForAttr("fruit"), types.Val{
		Tid:   types.StringID,
		Value: "banana",
	})
	require.NoError(t, err)

	enc.AddMapChild(root, firstChild)
	enc.AddMapChild(root, secondChild)

	enc.fixOrder(root)
	enc.buf.Reset()
	require.NoError(t, enc.encode(root))

	testutil.CompareJSON(t, `{
		"like": {
			"uid": "0x3",
			"fruit": "banana"
		}
	}`, enc.buf.String())
}
