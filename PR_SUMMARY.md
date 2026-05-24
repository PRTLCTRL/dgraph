# Fix for Issue #9422: Duplicate JSON Fields in Multiple Mutations

## Branch
`fix/issue-9422-e7dd` (pushed to PRTLCTRL/dgraph)

## Summary
Fixed a bug where multiple mutations (delete + set) on the same non-list predicate in a single transaction produced invalid JSON with duplicate keys.

## The Problem
When executing multiple mutations in a single transaction where:
1. First mutation deletes an edge (`uid(person) <like> * .`)
2. Second mutation creates a new edge (`uid(person) <like> uid(banana) .`)

The query result contained invalid JSON:
```json
{"like":{"uid":"0x2","fruit":"apple","uid":"0x3","fruit":"banana"}}
```

This violates JSON spec (duplicate keys) and the schema definition (`like: uid` not `[uid]`).

## Root Cause
The `AddMapChild` function in `query/outputnode.go` was merging children when finding an existing child with the same attribute, instead of replacing it. For non-list predicates, the latest value should replace the old one.

## The Fix
Modified `AddMapChild` to replace the existing child node entirely rather than merging children. The function now:
1. Searches for an existing child with the same attribute
2. If found, replaces it in-place by updating pointers
3. If not found, adds as a new child (original behavior)

## Changes Made
1. **query/outputnode.go**: Updated `AddMapChild` function (lines 499-529)
2. **query/outputnode_duplicate_uid_test.go**: Added unit test reproducing and validating the fix

## Testing Done
✅ Unit test `TestAddMapChildDuplicateUID` passes
✅ Existing tests pass: `TestFastJsonNode`, `TestChildrenOrder`, `TestStringJsonMarshal`
✅ Project builds successfully with `make install`

## Testing NOT Done (requires infrastructure)
- Full integration test with live Dgraph cluster
- End-to-end mutation scenarios with Docker
- Broader integration test suite

## Creating the Pull Request
The branch has been pushed to `PRTLCTRL/dgraph`. To create a PR to the upstream repository:

1. Visit: https://github.com/PRTLCTRL/dgraph/compare/main...fix/issue-9422-e7dd
2. Click "Create Pull Request"
3. Change base repository to `dgraph-io/dgraph`
4. Use the title: `fix: prevent duplicate JSON keys in non-list predicates with multiple mutations`
5. Use the body from the template below

## PR Description Template

```markdown
**Description**

Fixes an issue where multiple mutations (delete + set) on the same non-list predicate in a single transaction produced invalid JSON with duplicate keys. The query endpoint was returning things like:

\`\`\`json
{"like":{"uid":"0x2","fruit":"apple","uid":"0x3","fruit":"banana"}}
\`\`\`

This is invalid JSON—objects can't have duplicate keys—and it violates the schema since \`like: uid\` is defined as a single value, not \`[uid]\`.

**Root cause:** The \`AddMapChild\` function in \`query/outputnode.go\` was merging children when it found an existing child with the same attribute, instead of replacing it. For non-list predicates, the latest value should replace the old one, not get tacked on like a list.

**The fix:** Updated \`AddMapChild\` to replace the existing child node entirely rather than merging their children. Now when you delete an edge and add a new one in the same transaction, only the new edge appears in the response—exactly as it should for a non-list predicate.

**What I tested:**

- Added a unit test (\`TestAddMapChildDuplicateUID\`) that reproduces the exact scenario: two nodes with the same attribute being added via \`AddMapChild\`. Confirmed the output is now valid JSON with only the most recent value.
- Ran existing outputnode tests (\`TestFastJsonNode\`, \`TestChildrenOrder\`, \`TestStringJsonMarshal\`) to ensure nothing broke.
- Built the project with \`make install\`—compiled successfully.

**What I couldn't test:**

- The full integration scenario from the issue (running actual mutations against a live Dgraph instance) would require a running Dgraph cluster with Docker, which I don't have access to in this environment. The unit test covers the core encoding logic where the bug lives, but end-to-end validation would be good.
- Broader integration tests that exercise more complex mutation scenarios across the codebase.

**Checklist**

- [x] The PR title follows the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#summary) syntax
- [x] Code compiles correctly
- [x] Tests added for the bug fix

Fixes #9422

---

I'm trying to get more involved with this project—happy to iterate on this if anything looks off or if you'd like me to add more test coverage.
\`\`\`
```

## Commit Message
```
Fix duplicate JSON fields in non-list predicates

When multiple mutations (delete + set) operate on the same non-list
predicate in a single transaction, the JSON encoder was producing
invalid output with duplicate keys. For example, a 'like: uid'
predicate would generate:

{"like":{"uid":"0x2","fruit":"apple","uid":"0x3","fruit":"banana"}}

This is invalid JSON (duplicate keys) and violates the schema since
'like' is defined as uid, not [uid].

Root cause: AddMapChild was merging children when finding an existing
child with the same attribute, instead of replacing it. For non-list
predicates, the latest value should replace the old one.

The fix updates AddMapChild to replace the existing child node
entirely rather than merging their children, ensuring only the most
recent value appears in the output.

Fixes dgraph-io/dgraph#9422
```
