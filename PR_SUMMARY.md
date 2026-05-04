# PR Summary: Fix Duplicate JSON Fields in Multi-Mutation Transactions

## Quick Links
- **Issue**: https://github.com/dgraph-io/dgraph/issues/9422
- **Branch**: `cursor/fix-duplicate-json-fields-9422-433d`
- **Upstream PR Target**: `dgraph-io/dgraph` (main branch)

## PR Title
```
fix: prevent duplicate JSON fields in multi-mutation transactions
```

## Description (for PR body)

When multiple mutations in a single transaction delete and then set the same non-list uid predicate, the query response was generating invalid JSON with duplicate field names. This happened because both the old (deleted) and new (set) UIDs were being included in the result.

For example, with a `like: uid` predicate (non-list), after running:
```
uid(person) <like> * .       // delete
uid(person) <like> uid(banana) .  // set
```

The query result would produce:
```json
"like": {"uid":"0x2","fruit":"apple","uid":"0x3","fruit":"banana"}
```

This is invalid JSON because objects cannot have duplicate keys.

### Root Cause

The `preTraverse` method in `query/outputnode.go` iterates through all UIDs in the `uidMatrix` when building the JSON response, regardless of whether the predicate is a list or not. When a transaction contains both delete and set mutations for the same non-list edge, the `uidMatrix` temporarily holds both UIDs (old and new), causing the encoder to output both as separate child nodes with identical field names.

The issue only manifests when:
1. A non-list uid predicate is involved
2. Multiple mutations in the *same* transaction operate on it
3. The mutations include at least one deletion and one set

### Solution

For non-list predicates (`pc.List == false`), only process the last UID in the `uidMatrix`, which represents the final state after all transaction mutations are applied. This matches the expected behavior: non-list predicates can only point to a single UID.

The fix is minimal and surgical — just 9 lines added to handle the non-list case before the existing iteration loop.

### Testing

**What I tested:**
- ✅ Code compiles successfully: `make dgraph` and `go build ./query`  
- ✅ Added regression test `TestMultipleMutationsNoDuplicateJSONFields` that:
  - Sets up the exact schema and data from issue #9422
  - Performs delete + set mutations in a single transaction
  - Verifies the result contains only the new edge (banana), not both edges
  - Validates that the JSON is well-formed (no duplicate keys)
- ✅ Test compiles: `go test -tags=integration -c ./query`

**What I couldn't test:**
- ❌ Running the full integration test suite requires a live Dgraph cluster, which I don't have access to in this environment
- ❌ Manual reproduction with actual `dgraph alpha` and `dgo` client

The test should pass once CI runs it with a proper Dgraph instance. The logic is straightforward: for non-list predicates, we now skip all but the last UID.

### Checklist

- [x] The PR title follows Conventional Commits (`fix:`)
- [x] Code compiles correctly
- [x] Regression test added for the bug fix

### Notes

This is my first contribution to Dgraph — I'm working on getting more involved with the project. The fix itself is a targeted change that shouldn't affect list predicates or any other query paths. I'd appreciate maintainer review to make sure I haven't missed any edge cases around facets, normalization, or other uidMatrix consumers.

If there are other scenarios I should test or if the approach needs adjustment, happy to iterate!

Fixes dgraph-io/dgraph#9422

---

## Files Changed

1. **query/outputnode.go** (9 lines added)
   - Modified `preTraverse` method to only process the last UID for non-list predicates when multiple UIDs are present

2. **query/mutation_test.go** (new test)
   - Added `TestMultipleMutationsNoDuplicateJSONFields` test case that reproduces the issue and validates the fix

## How to Create the PR

Since the PR tool couldn't create it directly (fork permissions), you'll need to create it manually:

1. Go to: https://github.com/dgraph-io/dgraph/compare/main...PRTLCTRL:dgraph:cursor/fix-duplicate-json-fields-9422-433d
2. Click "Create pull request"
3. Use the title: `fix: prevent duplicate JSON fields in multi-mutation transactions`
4. Copy the description from above
5. Mark as draft initially if you'd like maintainer feedback before full review
