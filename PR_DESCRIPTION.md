# PR Title
fix(edgraph): allow whitespace between @if/@filter and opening parenthesis

# PR Description

**Description**

After upgrading from v25.3.1 to v25.3.3, users started getting "invalid cond value: must start with @if( or @filter(" errors on conditional mutations that previously worked. The issue is that the new `validateCondValue` function (introduced for security hardening) was rejecting conditionals with whitespace between the directive and the opening parenthesis, like `@if ( NOT eq(len(RoutesId), 0) )`.

Root cause: The validation was checking for `@if(` or `@filter(` as an exact string prefix, requiring no space. The security goal (preventing DQL injection) only requires that nothing *other than whitespace* appears between the directive and the opening paren. The strictness was accidental collateral damage.

**What changed:**
- Modified `validateCondValue` to check for `@if` or `@filter` prefix
- Added explicit check that content between directive and opening paren is whitespace-only
- Non-whitespace characters between them are still rejected (preserving injection prevention)

**What I tested:**
- Ran unit tests: `go test -v -run TestValidateCondValue ./edgraph/` — **passed**
- Ran all edgraph package tests: `go test ./edgraph/` — **passed**
- Added test cases for single space, multiple spaces, tab character, and the exact example from the issue
- Added negative test case to ensure `@ifx(...)` (non-whitespace) is still rejected

**What I couldn't test:**
I didn't run the full integration test suite or systest — those require a running Dgraph cluster with specific infrastructure I don't have access to. The unit tests cover the validation logic thoroughly, but if there's a specific integration test you'd like me to run, let me know and I can try to set it up.

**Checklist**
- [x] The PR title follows Conventional Commits syntax
- [x] Code compiles correctly and tests pass locally
- [x] Tests added for new functionality (whitespace cases + negative case)
- Docs: Not applicable — this restores previously-working behavior

Fixes dgraph-io/dgraph#9687

---

I'm trying to get more involved with this project — happy to iterate on this if anything looks off or if you'd like me to test something else.

# Instructions for Creating the PR

1. Go to: https://github.com/PRTLCTRL/dgraph/compare/dgraph-io:main...PRTLCTRL:fix/issue-9687-0855
2. Click "Create pull request"
3. Copy the title and description from above
4. Set as draft PR
5. Submit
