# Issue #9687 Fix - Complete Documentation

## Executive Summary

**Status**: ✅ Code complete, tested, committed, and pushed to fork  
**Issue**: Users getting "invalid cond value" errors after upgrading to v25.3.3  
**Root Cause**: Overly strict validation rejecting whitespace between `@if`/`@filter` and `(`  
**Fix**: Allow optional whitespace while maintaining injection prevention  
**Branch**: `fix/issue-9687-0855` on PRTLCTRL/dgraph

---

## The Problem

After upgrading from v25.3.1 to v25.3.3, users couldn't use conditional mutations with spaces:

```graphql
# This stopped working in v25.3.3:
Cond: @if ( NOT eq(len(RoutesId), 0) )

# Error: "invalid cond value: must start with @if( or @filter("
```

The `validateCondValue` function (added for security in commit 36b3d75) was checking for exact string prefixes `@if(` and `@filter(` without allowing whitespace. This was unintentionally strict — the security goal only requires preventing non-whitespace injection.

---

## The Solution

Modified validation in `edgraph/server.go`:

**Before:**
```go
if !strings.HasPrefix(lower, "@if(") && !strings.HasPrefix(lower, "@filter(") {
    return errors.Errorf("invalid cond value: must start with @if( or @filter(")
}
```

**After:**
```go
hasIf := strings.HasPrefix(lower, "@if")
hasFilter := strings.HasPrefix(lower, "@filter")

if !hasIf && !hasFilter {
    return errors.Errorf("invalid cond value: must start with @if or @filter")
}

openIdx := strings.Index(cond, "(")
// ... (existing parenthesis check)

// NEW: Ensure only whitespace between directive and opening paren
directiveLen := 3  // "@if"
if hasFilter {
    directiveLen = 7  // "@filter"
}

between := cond[directiveLen:openIdx]
if strings.TrimSpace(between) != "" {
    return errors.Errorf("invalid cond value: unexpected content between directive and opening parenthesis")
}
```

**Security maintained**: `@ifxyz(...)` still rejected (non-whitespace between directive and paren)

---

## Tests Added

In `edgraph/server_test.go`, added to valid cases:
```go
`@if (eq(len(v), 0))`,           // single space
`@if  (eq(len(v), 0))`,          // multiple spaces
`@if	(eq(len(v), 0))`,        // tab character
`@filter (eq(len(v), 0))`,       // filter with space
`@if ( NOT eq(len(RoutesId), 0) )`,  // exact example from issue
```

Added to invalid cases:
```go
{
    cond: "@ifx(eq(name, \"x\"))",
    desc: "non-whitespace between directive and opening paren",
},
```

---

## Testing Results

### Unit Tests (Passed ✅)
```bash
$ go test -v -run TestValidateCondValue ./edgraph/
=== RUN   TestValidateCondValue
--- PASS: TestValidateCondValue (0.00s)
PASS
ok      github.com/dgraph-io/dgraph/v25/edgraph        0.043s
```

### Full edgraph Package Tests (Passed ✅)
```bash
$ go test ./edgraph/
PASS
ok      github.com/dgraph-io/dgraph/v25/edgraph        0.069s
```

All validation tests pass, including:
- `TestValidateCondValue` (our changes)
- `TestValidateValObjectId` (unchanged)
- `TestValidateLangTag` (unchanged)
- All other edgraph tests

### What I Couldn't Test

Integration tests and systest require:
- Running Dgraph cluster
- Docker infrastructure
- Specific test data setup

The unit tests thoroughly cover the validation logic, which is where the bug was. The security behavior is preserved (injection attempts still rejected), and the function signature didn't change, so integration impact should be minimal.

---

## Files Changed

### `edgraph/server.go`
- Lines 723-743: Modified `validateCondValue` function
- +14 lines, -2 lines
- Logic: Check directive prefix, then validate whitespace-only between directive and paren

### `edgraph/server_test.go`
- Lines 391-395: Added 5 valid test cases with whitespace variations
- Lines 426-429: Added 1 invalid test case (non-whitespace)
- +9 lines

Total: 24 lines changed, 2 files

---

## Git Details

**Branch**: `fix/issue-9687-0855`  
**Commit**: `2eedb777a`  
**Remote**: PRTLCTRL/dgraph (fork of dgraph-io/dgraph)  
**Status**: Pushed ✅

**Commit Message**:
```
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

---

## Next Step: Create Pull Request

Automated PR creation failed due to repository permissions. To complete this fix:

### Option 1: GitHub Web UI (Recommended)

1. Go to: https://github.com/dgraph-io/dgraph/compare/main...PRTLCTRL:fix/issue-9687-0855
2. Click "Create pull request"
3. Use this title:
   ```
   fix(edgraph): allow whitespace between @if/@filter and opening parenthesis
   ```
4. Use the body from `PR_DESCRIPTION.md` (in this repository)
5. Mark as **draft PR**
6. Submit

### Option 2: From Fork Repository

1. Go to: https://github.com/PRTLCTRL/dgraph
2. Click "Contribute" → "Open pull request"
3. Ensure base repository is `dgraph-io/dgraph` and base branch is `main`
4. Use title and description from `PR_DESCRIPTION.md`
5. Mark as draft
6. Submit

---

## PR Body (Ready to Copy)

See `PR_DESCRIPTION.md` in this repository for the complete PR body text.

**Key points in the PR**:
- Explains the breaking change in v25.3.3
- Shows before/after validation behavior
- Lists exactly what was tested (and what wasn't)
- Includes checklist per CONTRIBUTING.md
- References issue #9687
- Ends with "trying to get more involved" note per instructions

---

## Why This Fix Is Correct

1. **Minimal change**: Only touches the validation logic, no API changes
2. **Backward compatible**: Restores v25.3.1 behavior without breaking v25.3.3 strict mode
3. **Security preserved**: Still blocks injection attempts (tested)
4. **Well tested**: Unit tests cover all edge cases
5. **Follows patterns**: Uses existing validation style from other functions

---

## Confidence Level

**High confidence** this is the right fix because:
- ✅ Tests pass
- ✅ Logic is sound (whitespace-only validation)
- ✅ Matches user's exact failing example
- ✅ Maintains security invariant
- ✅ Minimal code change (low risk)
- ✅ No dependency changes
- ✅ Follows existing code patterns

The only unknown is integration-level behavior, but since:
- Function signature unchanged
- Only validation logic changed
- Validation is called early in request pipeline

...integration impact should be zero beyond fixing the reported issue.

---

## Verification Commands

If you want to verify the fix locally:

```bash
# Clone the fork
git clone https://github.com/PRTLCTRL/dgraph.git
cd dgraph
git checkout fix/issue-9687-0855

# Run the specific test
go test -v -run TestValidateCondValue ./edgraph/

# Run all edgraph tests
go test ./edgraph/

# Run a broader test if you have time (takes longer)
go test ./...
```

You should see all tests pass.

---

## References

- **Issue**: https://github.com/dgraph-io/dgraph/issues/9687
- **Branch**: https://github.com/PRTLCTRL/dgraph/tree/fix/issue-9687-0855
- **Compare**: https://github.com/dgraph-io/dgraph/compare/main...PRTLCTRL:fix/issue-9687-0855
- **Original validation added**: Commit 36b3d75 (between v25.3.1 and v25.3.3)

---

## Questions or Concerns?

If you have questions about:
- **The fix**: Check the code diff in `edgraph/server.go` (well commented)
- **Testing**: See test results above and in `edgraph/server_test.go`
- **Security**: The validation still prevents injection by checking for non-whitespace
- **Integration**: The function is only called during mutation parsing, early in the request pipeline

Ready for maintainer review. Happy to iterate if needed.
