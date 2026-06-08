# Fix for Issue #9422: Duplicate JSON Fields in Mutation Transactions

## Summary

Fixed a bug where combining delete and set mutations on a single-value predicate within the same transaction would produce invalid JSON with duplicate field names.

## Problem

When executing these mutations in a single transaction:
```
mu1: uid(person) <like> * .     (delete)
mu2: uid(person) <like> uid(banana) .  (set)
```

The subsequent query would return:
```json
{"uid":"0x2","fruit":"apple","uid":"0x3","fruit":"banana"}
```

This is invalid JSON because objects cannot have duplicate keys.

## Root Cause

The issue was in the JSON encoder (`query/outputnode.go`). When a single-value predicate had multiple UIDs in the uidMatrix (due to transaction-level visibility of both old and new values):

1. The code looped through each UID (line 1429)
2. For non-list predicates, it called `AddMapChild()` (line 1513)
3. `AddMapChild()` would merge children of nodes with the same attribute
4. This resulted in duplicate `uid` fields in the same JSON object

## The Fix

**Files Changed:**
- `query/outputnode.go` - Added `ReplaceMapChild()` function
- `query/outputnode_test.go` - Added test case

**Changes:**

1. **New function `ReplaceMapChild()`**: Replaces an existing child node instead of merging
2. **Updated `AddMapChild()`**: Refactored to use internal function `addMapChildInternal()` with a `replace` parameter
3. **Updated calling code**: Changed line 1513 to use `ReplaceMapChild()` for single-value predicates

The key insight: For single-value predicates (`!pc.List`), when we encounter a duplicate attribute, we should replace the entire node, not merge its children.

## Testing Done

✅ **Unit Test**: Created `TestSingleValuePredicateReplace` that verifies:
   - No duplicate JSON keys
   - Valid JSON output
   - Latest value is retained

✅ **Package Tests**: All query package tests pass (`go test ./query/`)

✅ **Build**: Successfully built dgraph binary (`make dgraph`)

✅ **JSON Tests**: All JSON encoding tests pass

## Testing NOT Done (Requires Running Dgraph Instance)

❌ **End-to-end test**: Cannot verify the exact scenario from the issue:
   - Start Dgraph instance
   - Load schema and test data
   - Execute mutations and query in same transaction
   - Verify JSON response

This would require a running Dgraph server with proper setup, which I don't have access to in this environment.

## Code Quality

- ✅ No linting errors
- ✅ Follows existing code patterns
- ✅ No drive-by changes
- ✅ Minimal, targeted fix
- ✅ Added appropriate comments

## Branch and Commit

- **Branch**: `cursor/fix-duplicate-json-fields-9ad9`
- **Commit**: `3b85cb496`
- **Remote**: Pushed to `PRTLCTRL/dgraph`

## Next Steps

1. Create PR from fork to upstream `dgraph-io/dgraph`
2. Get maintainer review
3. If approved, merge to main

## Notes for Reviewers

- The fix is in the JSON encoding layer, not the mutation/transaction layer
- Single-value predicates now use `ReplaceMapChild()` instead of `AddMapChild()`
- List predicates continue to use `AddListChild()` (no change in behavior)
- The test demonstrates the fix works at the encoding level
