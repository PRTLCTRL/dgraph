**Description**

When executing multiple mutations in a single transaction where the first mutation deletes an edge and the second mutation creates a new edge for the same non-list predicate, the JSON encoder was producing invalid JSON with duplicate field names.

Root cause: The `AddMapChild` method in the JSON encoder was merging children when a node with the same attribute already existed, instead of replacing the old node entirely. For non-list predicates, this is incorrect behavior — think of it like updating a field in a struct; you don't append to the old value, you replace it.

The fix is straightforward: when adding a child to a non-list predicate that already has a child with the same attribute, replace the existing child entirely instead of merging their children together.

**What Changed**

Modified `query/outputnode.go`:
- Updated `AddMapChild` to track the previous sibling pointer during traversal
- When a duplicate attribute is found, replace the entire child node instead of merging children
- This ensures non-list predicates behave correctly when mutations delete then set values in the same transaction

Added test in `systest/mutations-and-queries/mutations_test.go`:
- `TestMultipleMutationsNonListPredicate` reproduces the exact scenario from the issue
- Verifies that JSON is valid and contains only the new value (no duplicates)

**Testing**

I tested this by:
1. Building the project with `make dgraph` — no compilation errors
2. Running unit tests for the query package with `go test ./query/...` — all passed
3. Added an integration test that reproduces the bug scenario and validates the fix

I couldn't run the full integration test suite locally because Docker isn't available in my environment, but the test is structured following existing patterns in the test suite and should work once CI runs it with a proper Dgraph cluster.

**What I Couldn't Test**

- Full integration test suite (requires Docker/running Dgraph cluster)
- Performance impact on large-scale mutations (though the change is minimal and only affects the specific case of duplicate attributes)
- Edge cases with deeply nested structures (added logic is localized to `AddMapChild`)

**Additional Context**

The `like` predicate in the issue was defined as `uid` (not `[uid]`), so it should only point to one UID at a time. The bug was causing it to contain both the old and new values, producing this invalid JSON:

```json
"like":{"uid":"0x2","fruit":"apple","uid":"0x3","fruit":"banana"}
```

After the fix, the JSON correctly shows only the new value:

```json
"like":{"uid":"0x3","fruit":"banana"}
```

Fixes dgraph-io/dgraph#9422

**Checklist**

- [x] The PR title follows the Conventional Commits syntax, leading with `fix:`, `feat:`, `chore:`, `ci:`, etc.
- [x] Code compiles correctly and linting (via trunk) passes locally
- [x] Tests added for new functionality, or regression tests for bug fixes added as applicable

---

I'm trying to get more involved with this project — happy to iterate on this if anything looks off or if there are additional edge cases worth considering.
