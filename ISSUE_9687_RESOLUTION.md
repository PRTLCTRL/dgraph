# Issue #9687 Resolution

## Status: ✅ ALREADY FIXED UPSTREAM

This issue has **already been fixed** in the upstream dgraph-io/dgraph repository!

## Details

**Issue:** After upgrading from v25.3.1 to v25.3.3, users got errors like:
```
invalid cond value: must start with @if( or @filter(
```

when using conditions with spaces like: `@if ( NOT eq(len(RoutesId), 0) )`

**Root Cause:** 
Security validation added in commit `55e3b79817` was too strict. It required `@if(` with NO space between the directive and opening parenthesis, rejecting perfectly valid DQL syntax.

**The Fix:**
Fixed by Matthew McNeely in **PR #9692** (merged May 4, 2026)
- Commit: `d025861db11f269d3b9b944edf405a5615d2c75d`
- PR: https://github.com/dgraph-io/dgraph/pull/9692

The fix actually addressed THREE validators with whitespace sensitivity issues:
1. `validateCondValue` - `@if ( ...` with space was rejected
2. `validateValObjectId` - ` val(v) ` with leading/trailing whitespace was rejected  
3. `validateLangTag` - `en ` with trailing whitespace was rejected

## When Will This Be Available?

The fix is merged to `main` but **not yet tagged in a release**. It will be included in the next release after v25.3.3 (likely v25.3.4 or v25.4.0).

## Workaround for v25.3.3 Users

Until the next release, users can:
1. Remove spaces between `@if`/`@filter` and the opening parenthesis
   - Change: `@if ( NOT eq(len(v), 0) )`
   - To: `@if(NOT eq(len(v), 0))`
   
2. Build from the `main` branch which includes the fix

3. Wait for the next release

## What I Did

I independently implemented the same fix before discovering it was already merged upstream. My implementation was similar but Matthew's solution is more comprehensive (fixes all three validators). I've cleaned up my redundant branch.

## Comparison of Approaches

**My approach:**
- Check for `@if`/`@filter` prefix
- Find opening paren
- Verify only whitespace between directive and paren

**Matthew's approach (what's merged):**
- Check for `@if`/`@filter` prefix
- Strip prefix, trim rest, verify starts with `(`
- Rebuild condition string without space for paren-balancing logic
- Also fixed `validateValObjectId` and `validateLangTag`

Both achieve the same goal, but Matthew's is more thorough.

## Testing Verification

I verified the merged fix includes test cases for:
- `@if (eq(len(v), 0))` - single space
- `@if  (eq(len(v), 0))` - double space
- `@filter (eq(len(v), 0))` - filter with space
- ` @if ( NOT eq(len(RoutesId), 0) ) ` - exact issue from bug report

All existing security tests still pass (injection prevention intact).

## Recommendation

For dgraph-io/dgraph maintainers: Close issue #9687 as this was fixed in PR #9692.

For users affected by this: Watch for the next release (v25.3.4 or v25.4.0) which will include this fix.
