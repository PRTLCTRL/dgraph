# Fix for Issue #9422: Multiple Mutations Create Invalid JSON

## Status: ✅ Code Complete, Ready for PR

All development work is complete and pushed to the fork. The only remaining step is creating the PR through GitHub's web interface.

---

## What Was Fixed

**Problem:** When multiple mutations in a single transaction delete then set the same non-list predicate, the JSON response contained duplicate fields, producing invalid JSON.

**Root Cause:** The `AddMapChild` method was merging children instead of replacing them for non-list predicates.

**Solution:** Modified `AddMapChild` to replace the entire existing child node when a duplicate attribute is found.

---

## Changes Made

### 1. Code Fix
**File:** `query/outputnode.go`
**Function:** `AddMapChild`

Changed the behavior to track the previous sibling and properly replace duplicate children in the linked list structure instead of merging them.

### 2. Test Added
**File:** `systest/mutations-and-queries/mutations_test.go`
**Function:** `TestMultipleMutationsNonListPredicate`

Comprehensive integration test that:
- Sets up the exact scenario from the issue
- Performs a delete + set mutation in one transaction
- Validates the JSON is valid and contains only the new value

---

## Testing Performed

✅ **Build:** `make dgraph` completed successfully
✅ **Unit Tests:** `go test ./query/...` passed
⚠️ **Integration Tests:** Cannot run locally (requires Docker)

The integration test follows existing patterns and should work in CI.

---

## Commits

1. `b0ac3b167` - Fix duplicate JSON fields when mutations delete then set same predicate
2. `cc30e3d04` - Add test for issue #9422: multiple mutations on non-list predicate

Branch: `cursor/fix-issue-9422-d54c`
Fork: `PRTLCTRL/dgraph`

---

## To Create the PR

### Option 1: GitHub Web Interface (Recommended)
Visit: https://github.com/PRTLCTRL/dgraph/pull/new/cursor/fix-issue-9422-d54c

- **Title:** `fix: prevent duplicate JSON fields in multi-mutation transactions`
- **Body:** Use the content from `PR_DESCRIPTION.md` (in workspace root)
- **Base:** `dgraph-io/dgraph:main`
- **Draft:** Yes

### Option 2: GitHub CLI (if you have proper permissions)
```bash
gh pr create \
  --repo dgraph-io/dgraph \
  --head PRTLCTRL:cursor/fix-issue-9422-d54c \
  --base main \
  --title "fix: prevent duplicate JSON fields in multi-mutation transactions" \
  --body-file PR_DESCRIPTION.md \
  --draft
```

---

## Example: Before and After

### Before (Invalid JSON)
```json
{
  "q": [{
    "uid": "0x1",
    "like": {"uid": "0x2", "fruit": "apple", "uid": "0x3", "fruit": "banana"}
  }]
}
```
☝️ Duplicate `uid` and `fruit` fields in the same object

### After (Valid JSON)
```json
{
  "q": [{
    "uid": "0x1",
    "like": {"uid": "0x3", "fruit": "banana"}
  }]
}
```
☝️ Clean, valid JSON with only the new value

---

## Files Changed
- `query/outputnode.go` (19 lines changed)
- `systest/mutations-and-queries/mutations_test.go` (105 lines added)

## References
- Issue: https://github.com/dgraph-io/dgraph/issues/9422
- Branch: https://github.com/PRTLCTRL/dgraph/tree/cursor/fix-issue-9422-d54c
