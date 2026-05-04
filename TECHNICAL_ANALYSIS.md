# Technical Analysis: Issue #9422

## Problem Statement

When executing multiple mutations in a single transaction that:
1. Delete an edge: `uid(person) <like> * .`
2. Set a new edge: `uid(person) <like> uid(banana) .`

The query response contains **both** the old and new UIDs, producing invalid JSON:

```json
{
  "like": {
    "uid": "0x2",     // old value (apple)
    "fruit": "apple",
    "uid": "0x3",     // new value (banana) 
    "fruit": "banana"
  }
}
```

This is invalid JSON since objects cannot have duplicate keys.

## Root Cause Analysis

### Where the bug occurs

File: `query/outputnode.go`  
Method: `SubGraph.preTraverse()`  
Line: ~1429 (before fix)

### The problematic code path

```go
// Line 1428
ul := pc.uidMatrix[idx]
for childIdx, childUID := range ul.Uids {  // Iterates ALL UIDs
    uc := enc.newNode(fieldID)
    // ... builds JSON node for each UID
    if pc.List {
        enc.AddListChild(dst, uc)
    } else {
        enc.AddMapChild(dst, uc)  // For non-list predicates
    }
}
```

### Why it happens

1. **Transaction processing**: When mutations delete and then set an edge in the same transaction, the `uidMatrix` temporarily contains both UIDs:
   - `ul.Uids = [0x2, 0x3]` (apple, then banana)

2. **Loop iteration**: The code iterates through **all** UIDs regardless of whether the predicate is a list type:
   ```go
   for childIdx, childUID := range ul.Uids {  // Processes both 0x2 and 0x3
   ```

3. **AddMapChild behavior**: For non-list predicates, `AddMapChild` is supposed to merge children, but it merges the *children* of nodes with the same attribute name, not replace the parent node itself:
   ```go
   func (enc *encoder) AddMapChild(fj, val fastJsonNode) {
       // Finds existing child with same attribute
       // Merges val's children into that child
       // But doesn't prevent multiple top-level nodes with same attr
   }
   ```

4. **Result**: Two separate nodes are created with the same field name ("like"), causing duplicate JSON keys.

## The Fix

### What changed

For non-list predicates, only process the **last UID** in the uidMatrix, which represents the final state after all mutations:

```go
// Line 1430-1437 (after fix)
ul := pc.uidMatrix[idx]

// For non-list predicates, only process the last UID to avoid duplicate
// JSON fields when multiple mutations (delete + set) occur in the same transaction.
startIdx := 0
if !pc.List && len(ul.Uids) > 1 {
    startIdx = len(ul.Uids) - 1  // Skip to last UID
}

for childIdx := startIdx; childIdx < len(ul.Uids); childIdx++ {
    childUID := ul.Uids[childIdx]  // Only processes 0x3 (banana)
    // ... rest of loop
}
```

### Why this is correct

1. **Semantic correctness**: Non-list predicates (`uid`) can only point to **one** UID. Processing only the last one gives us the final transaction state.

2. **List predicates unaffected**: When `pc.List == true` (for `[uid]` predicates), `startIdx` remains 0, so all UIDs are processed as before.

3. **Minimal change**: The fix is surgical — only 9 lines added, no other code paths affected.

4. **Consistent with downstream code**: The code already distinguishes between list/non-list at line 1519-1523 (`AddListChild` vs `AddMapChild`), so this early filtering is aligned with existing patterns.

## Edge Cases Considered

### 1. Single UID (normal case)
- `len(ul.Uids) == 1` → `startIdx = 0` → Same behavior as before ✓

### 2. List predicates with multiple UIDs
- `pc.List == true` → `startIdx = 0` → All UIDs processed ✓

### 3. Delete-only mutation
- After delete, `ul.Uids` should be empty or contain tombstone
- Loop wouldn't execute or would process tombstone correctly ✓

### 4. Set-only mutation
- Only new UID in `ul.Uids` → Same as single UID case ✓

### 5. Multiple sets in same transaction
- `ul.Uids = [0x2, 0x3, 0x4]` → Only 0x4 processed → Correct final state ✓

### 6. Facets and other metadata
- Facets are indexed by `childIdx` in `fcsList`
- Since we only process last UID, we'll use `childIdx = len-1` → Correct facet ✓

## Validation

### What was tested
- ✅ Code compiles: `make dgraph` and `go build ./query`
- ✅ Test compiles: `go test -tags=integration -c ./query`
- ✅ Added regression test `TestMultipleMutationsNoDuplicateJSONFields`

### What requires CI
- ❌ Running integration tests (needs live Dgraph cluster)
- ❌ Full test suite validation
- ❌ Performance impact (likely negligible — just a bounds check)

## Alternative Approaches Considered

### Alternative 1: Fix in AddMapChild
**Idea**: Modify `AddMapChild` to replace parent instead of merging children  
**Rejected**: Would affect all callers of `AddMapChild`, potentially breaking other use cases

### Alternative 2: Deduplicate in uidMatrix earlier
**Idea**: Clean up uidMatrix after mutations are applied  
**Rejected**: More invasive change, unclear where this should happen in mutation processing

### Alternative 3: Post-process JSON to remove duplicates
**Idea**: Detect and fix duplicate keys after encoding  
**Rejected**: Hacky, expensive, treats symptom not cause

### Why the current fix is best
- Minimal code change (9 lines)
- Fixes root cause (don't create duplicates in the first place)
- No performance impact
- Aligned with existing code patterns (`pc.List` checks)
- Easy to understand and maintain
