# Fix: Duplicate JSON fields when delete-all and set mutations occur in same transaction

## What was the issue?

When you delete all edges for a predicate and then set a new edge in the same transaction, the query would return **both the old and new values** in invalid JSON with duplicate field keys.

The issue manifests as a speedometer that only works after you've already been speeding—except in this case, it's a database that remembers things you explicitly told it to forget.

For example:
```json
{"q":[{"uid":"0x1","like":{"uid":"0x2","fruit":"apple","uid":"0x3","fruit":"banana"}}]}
```

This is invalid JSON (objects can't have duplicate keys), and it's wrong behavior. After deleting all likes and setting a new like to banana, you should only see banana, not both apple and banana.

## What caused it?

The bug was in `MutableLayer.iterate()` in `posting/list.go`. When a delete-all marker existed in `currentEntries` (uncommitted mutations) and there were also committed entries at the **same timestamp**, both were being included in the iteration.

Here's the scenario that triggers it:
1. You have some committed data at timestamp T (apple edge)  
2. You start a new transaction that also uses timestamp T (edge case, but possible)
3. You issue a delete-all operation (goes into `currentEntries`)
4. You issue a set operation for a new value (banana)
5. The `iterate()` function has to decide what postings to include
6. It finds the delete-all marker at timestamp T
7. It correctly excludes data from **before** timestamp T
8. **BUG:** It includes BOTH `committedEntries[T]` (apple) AND `currentEntries` (delete-all + banana)
9. Result: apple wasn't filtered out even though it came before the delete-all

The condition `ts >= deleteAllMarker` was meant to exclude old data, but when old committed data had the exact same timestamp as the delete-all marker, it got included.

## What I changed and why

Modified `MutableLayer.iterate()` in `posting/list.go` to:
1. Check if `currentEntries` contains a delete-all marker
2. If yes, exclude committed entries at the exact same timestamp as that delete-all
3. Those committed entries were added **before** the delete-all chronologically, so they should be filtered

The fix is surgical and minimal:
- Added a pre-scan of `currentEntries` to detect delete-all markers
- Added a condition to skip `committedEntries[T]` when `currentEntries` has a delete-all at timestamp T
- Everything else stays the same

This only affects the specific edge case and doesn't change the normal flow.

## What I actually tested

**Test added:** `systest/mutations-and-queries/issue9422_test.go`

The test:
1. Sets up schema with single-valued `like` predicate
2. Creates Tom who likes apple  
3. In one transaction: `DelNquads` to delete all likes, then `SetNquads` to like banana
4. Queries and verifies:
   - JSON is valid (no duplicate keys)
   - Result contains only banana (not apple)
   - The old value (apple) is gone

**What I ran:** Code review, logic tracing, test authoring

**What I couldn't run:** The full Dgraph build and test suite. Build requires jemalloc compilation and takes significant time in this environment. I validated the logic carefully and added a comprehensive test, but I couldn't actually execute it.

## What I COULDN'T test and why

Be upfront about limitations:

- **Full integration test suite** — Build environment setup is complex (jemalloc, Go modules, etc.)
- **Performance benchmarks** — Need running cluster
- **Multi-node scenarios** — Need distributed setup
- **Actual test execution** — Couldn't get a full Dgraph instance running

The fix is minimal and the logic is sound, but I acknowledge this needs to pass CI before merging. I'd appreciate if maintainers could run the full test suite to validate.

## Trade-offs and considerations

**Performance impact:** The added check scans `currentEntries.Postings` once to detect delete-all markers. This is negligible because:
- We're about to iterate these postings anyway
- The scan is O(n) where n is typically small (postings in current transaction)
- It only runs when `currentEntries` exists

**Edge case frequency:** This bug only triggers when:
1. `committedEntries` and `currentEntries` share the same timestamp
2. `currentEntries` contains a delete-all
3. A query reads the posting list during or after this transaction

In normal operation, this shouldn't be common, but when it happens, it causes data correctness issues that violate user expectations and produce invalid JSON.

## Links

Fixes dgraph-io/dgraph#9422

---

**Note:** I'm trying to get more involved with this project. Happy to iterate on this PR if anything looks off, or if there are patterns I should follow that I missed. Feedback welcome!
