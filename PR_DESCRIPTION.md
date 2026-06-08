## PR Title
fix: prevent duplicate JSON fields for single-value predicates in transactions

## PR Description

**Description**

Fixes dgraph-io/dgraph#9422

The query engine was returning invalid JSON when a transaction combined delete and set mutations on a single-value predicate. The result would look like:

```json
{"uid":"0x2","fruit":"apple","uid":"0x3","fruit":"banana"}
```

...which is a bit like having two steering wheels in a car. Technically interesting, legally questionable, definitely not JSON-spec compliant.

**Root Cause**

When mutations delete an edge (`uid(person) <like> * .`) and then add a new one (`uid(person) <like> uid(banana) .`) within the same transaction, the query sees both the old and new values in the uidMatrix. The JSON encoder was calling `AddMapChild()` for each UID, which merged the children into the same parent node, creating duplicate keys.

**The Fix**

Introduced `ReplaceMapChild()` that replaces the existing child node instead of merging for single-value predicates (`!List`). For list-type predicates, the behavior remains unchanged (merge via `AddListChild()`).

The fix is straightforward: when we know we're dealing with a single-value predicate, we replace the old child node entirely rather than trying to merge its children. This ensures only the most recent value appears in the output.

**What I Tested**

- Added `TestSingleValuePredicateReplace` to verify the fix produces valid JSON with no duplicate keys
- Ran the full query package test suite: `go test ./query/` — all tests pass
- Built dgraph binary successfully: `make dgraph`

**What I Couldn't Test**

I don't have a running Dgraph instance to test the exact reproduction case from the issue (delete + set mutations with queries in the same transaction). The fix addresses the JSON encoding layer, which is where the duplicate keys were being created, but I can't verify the full end-to-end behavior with live mutations.

**Checklist**

- [x] The PR title follows the Conventional Commits syntax
- [x] Code compiles correctly
- [x] Tests added for new functionality

---

I'm trying to get more involved with this project — happy to iterate on this if anything looks off or if you'd like me to add more test coverage.

## How to Create the PR

Since this is a fork, you'll need to manually create the PR on GitHub:

1. Go to https://github.com/PRTLCTRL/dgraph/pull/new/cursor/fix-duplicate-json-fields-9ad9
2. Change the base repository to `dgraph-io/dgraph` and base branch to `main`
3. Copy the title and description above
4. Mark it as a draft PR initially
5. Submit for review
