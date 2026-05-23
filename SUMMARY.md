# Issue #9687 Fix - Complete Summary

## ✅ What Was Done

### 1. Root Cause Identified
The `validateCondValue()` function added in v25.3.3 for DQL injection prevention was rejecting valid conditional upsert syntax that included whitespace between the directive (`@if` or `@filter`) and the opening parenthesis.

**Example of rejected syntax:**
```
Cond: @if ( NOT eq(len(RoutesId), 0) )
```

### 2. Code Fixed
**File:** `edgraph/server.go`
- Modified `validateCondValue()` to allow optional whitespace between directive and `(`
- Maintains all security validations (balanced parentheses, no trailing injection)
- Total change: +20 lines, -2 lines

**File:** `edgraph/server_test.go`  
- Added 5 new test cases covering whitespace scenarios
- Includes the exact syntax from the reported issue

### 3. Testing Completed
✅ **Unit tests pass:** `go test -v ./edgraph/ -run TestValidate`
✅ **Build succeeds:** `make dgraph`
✅ **New test cases validate the fix**

### 4. Git Operations Completed
✅ **Branch created:** `fix/issue-9687-00dd`
✅ **Changes committed** with descriptive message
✅ **Pushed to remote:** `origin/fix/issue-9687-00dd`

---

## 🔗 Create the Pull Request

The code is ready but requires **manual PR creation** due to GitHub permissions.

### Quick Link (Recommended)
👉 **Click here to create the PR:** https://github.com/dgraph-io/dgraph/compare/main...PRTLCTRL:dgraph:fix/issue-9687-00dd

This link will:
- Pre-fill the branch comparison
- Show the diff immediately
- Allow you to add the PR title and description

### PR Title
```
fix(edgraph): allow whitespace between @if/@filter and opening parenthesis
```

### PR Description
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

## 📋 Technical Details

### The Bug
**Introduced in:** v25.3.3
**Location:** `edgraph/server.go` line 724 (before fix)
**Problem:** Exact string match required `@if(` with no space

### The Fix
**Approach:** Check for directive, trim whitespace, then check for `(`
**Security:** All injection protections maintained
**Compatibility:** Backwards compatible with existing syntax

### Code Change Summary
```diff
- if !strings.HasPrefix(lower, "@if(") && !strings.HasPrefix(lower, "@filter(") {
+ if strings.HasPrefix(lower, "@if") {
+     rest := strings.TrimSpace(lower[3:])
+     if strings.HasPrefix(rest, "(") {
+         hasValidPrefix = true
```

---

## 📦 Files Changed
```
edgraph/server.go      | 20 ++++++++++++++++++--
edgraph/server_test.go |  5 +++++
2 files changed, 23 insertions(+), 2 deletions(-)
```

---

## 🎯 Next Steps

1. **Create the PR** using the link above
2. **Mark as draft** (or ready for review if you prefer)
3. **Monitor for maintainer feedback**
4. **Be ready to iterate** based on reviewer comments

The fix is minimal, focused, and includes test coverage. It maintains the security intent of the original validation while allowing legitimate syntax.
