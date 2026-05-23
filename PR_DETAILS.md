# Pull Request Details for Issue #9687

## Summary

Fixed validation error that was rejecting valid conditional upsert syntax with whitespace.

## Changes Made

### Modified Files:
1. **edgraph/server.go** - Updated `validateCondValue()` function to allow optional whitespace between `@if`/`@filter` and opening parenthesis
2. **edgraph/server_test.go** - Added 5 new test cases covering whitespace scenarios

### Branch
- `fix/issue-9687-00dd` (already pushed to PRTLCTRL/dgraph)

### Commit
```
fix(edgraph): allow whitespace between @if/@filter and opening parenthesis

The validateCondValue function introduced in v25.3.3 was too strict in
validating conditional upsert syntax. It required @if( or @filter( with
no space between the directive and opening parenthesis.

This broke existing mutations like:
  Cond: @if ( NOT eq(len(RoutesId), 0) )

The fix allows optional whitespace between the directive name and the
opening parenthesis while maintaining protection against DQL injection.

Fixes dgraph-io/dgraph#9687
```

## Testing Results

✅ **Passed**:
- `go test -v ./edgraph/ -run TestValidate` - All validation tests pass
- `make dgraph` - Builds successfully
- Added test cases for the exact syntax from the issue

❌ **Not Tested** (no infrastructure access):
- End-to-end mutation with live Dgraph cluster
- Integration tests requiring Docker

## Create the PR

**Option 1: Use this URL** (will pre-fill the branch comparison):
https://github.com/dgraph-io/dgraph/compare/main...PRTLCTRL:dgraph:fix/issue-9687-00dd

**Option 2: Manual steps**:
1. Go to https://github.com/dgraph-io/dgraph
2. Click "New Pull Request"
3. Click "compare across forks"
4. Set head repository to: PRTLCTRL/dgraph
5. Set compare branch to: fix/issue-9687-00dd
6. Use the PR body below

---

## PR Title
```
fix(edgraph): allow whitespace between @if/@filter and opening parenthesis
```

## PR Body
```markdown
**Description**

Starting with v25.3.3, Dgraph got picky about whitespace in conditional upserts — which is a bit like complaining about the font in a perfectly valid legal document. The validation added to prevent DQL injection was rejecting mutations like `@if ( NOT eq(len(RoutesId), 0) )` because of the space between the directive and its opening paren.

**Root cause**: The `validateCondValue` function introduced in v25.3.3 checks for exact prefix matches of `@if(` or `@filter(`, treating any space as suspicious. Meanwhile, the actual query builder (`buildUpsertQuery`) has zero opinion about whitespace — it just does a simple string replace and carries on. Classic case of the doorman being stricter than the actual venue.

**The fix**: Modified the validation to check for `@if` or `@filter` followed by optional whitespace, then a `(`. Security is preserved (still validates balanced parens and no trailing injection attempts), but legitimate mutations with breathing room are now allowed.

**What I tested**:
- `go test -v ./edgraph/ -run TestValidate` — all validation tests pass
- Added 5 new test cases covering various whitespace scenarios including the exact case from the issue
- `make dgraph` — builds successfully
- Did NOT test with a live Dgraph instance (no cluster access), so this is validated at the unit level only

**What I couldn't test**:
- End-to-end mutation with a running Dgraph cluster
- Integration tests that require Docker/infrastructure

Fixes dgraph-io/dgraph#9687

**Checklist**

- [x] The PR title follows the Conventional Commits syntax
- [x] Code compiles correctly
- [x] Tests added for new functionality

---

I'm trying to get more involved with this project — happy to iterate on this if anything looks off or if maintainers want me to run additional tests.
```

---

## Technical Details

### The Bug
In v25.3.3, a new security validation `validateCondValue()` was added to prevent DQL injection attacks via the `Cond` field in conditional upserts. However, it was checking for exact prefix `@if(` or `@filter(` which rejected valid syntax like `@if ( expression )`.

### The Fix
Changed the validation logic from:
```go
if !strings.HasPrefix(lower, "@if(") && !strings.HasPrefix(lower, "@filter(") {
    return errors.Errorf("invalid cond value: must start with @if( or @filter(")
}
```

To:
```go
if strings.HasPrefix(lower, "@if") {
    rest := strings.TrimSpace(lower[3:])
    if strings.HasPrefix(rest, "(") {
        hasValidPrefix = true
        openIdx = strings.Index(cond, "(")
    }
} else if strings.HasPrefix(lower, "@filter") {
    rest := strings.TrimSpace(lower[7:])
    if strings.HasPrefix(rest, "(") {
        hasValidPrefix = true
        openIdx = strings.Index(cond, "(")
    }
}
```

This allows optional whitespace while maintaining all security checks (balanced parentheses, no trailing injection).
