# Fix for Issue #9687 - Summary

## What was done

✅ **Branch created**: `fix/issue-9687-0855`
✅ **Changes committed** and **pushed** to PRTLCTRL/dgraph
✅ **All unit tests passing**

## The Fix

Modified the `validateCondValue` function in `edgraph/server.go` to allow optional whitespace between `@if`/`@filter` directives and the opening parenthesis.

**Files changed:**
- `edgraph/server.go` - Updated validation logic
- `edgraph/server_test.go` - Added test cases for whitespace variations

## Testing

**What passed:**
```bash
go test -v -run TestValidateCondValue ./edgraph/  # ✅ PASSED
go test ./edgraph/                                 # ✅ PASSED (all tests)
```

**Test cases added:**
- `@if (eq(len(v), 0))` - single space
- `@if  (eq(len(v), 0))` - multiple spaces
- `@if	(eq(len(v), 0))` - tab character
- `@filter (eq(len(v), 0))` - filter with space
- `@if ( NOT eq(len(RoutesId), 0) )` - exact example from issue
- `@ifx(eq(name, "x"))` - negative test (should fail)

## Next Steps

Create a pull request from the fork to the upstream repository:

**Option 1: Web UI (Recommended)**
Go to: https://github.com/dgraph-io/dgraph/compare/main...PRTLCTRL:fix/issue-9687-0855

**Option 2: Direct link**
Or use the GitHub-suggested URL: https://github.com/PRTLCTRL/dgraph/pull/new/fix/issue-9687-0855
(Note: Make sure to change the base repository to dgraph-io/dgraph if needed)

**PR Details:**
- Title: `fix(edgraph): allow whitespace between @if/@filter and opening parenthesis`
- Base: `dgraph-io/dgraph:main`
- Head: `PRTLCTRL:fix/issue-9687-0855`
- Type: Draft PR
- See `PR_DESCRIPTION.md` for the full PR body

## Commit Details

```
commit 2eedb777a
Author: Matthew McNeely <matthew.mcneely@gmail.com>
Date:   Sat May 23 08:54:17 2026 +0000

    fix(edgraph): allow whitespace between @if/@filter and opening parenthesis
    
    The validateCondValue function was rejecting conditional mutations with
    whitespace between the directive and opening paren (e.g., '@if (...)'),
    which broke existing code after upgrading from v25.3.1 to v25.3.3.
    
    Root cause: The validation was checking for '@if(' or '@filter(' as an exact
    prefix, requiring no space. The security goal (preventing DQL injection) only
    requires that nothing *other than whitespace* appears between the directive
    and the opening paren.
    
    Changes:
    - Modified validation to check for '@if' or '@filter' prefix
    - Added check that content between directive and opening paren is whitespace-only
    - Added test cases for single space, multiple spaces, and tabs
    - Added negative test case to ensure non-whitespace is still rejected
    
    The fix maintains the injection-prevention behavior while accepting the
    whitespace format that users were depending on.
    
    Fixes dgraph-io/dgraph#9687
```
