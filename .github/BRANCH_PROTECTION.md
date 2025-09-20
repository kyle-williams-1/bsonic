# Branch Protection Rules

This document describes the recommended branch protection rules for this repository.

## Main Branch Protection

To set up branch protection for the `main` branch:

1. Go to Settings → Branches
2. Add rule for `main` branch
3. Configure the following settings:

### Required Status Checks
- ✅ Require status checks to pass before merging
- ✅ Require branches to be up to date before merging
- Select the following status checks:
  - `Test` (from test.yml)
  - `Lint` (from ci.yml)
  - `Build` (from ci.yml)
  - `Security Scan` (from ci.yml)

### Additional Settings
- ✅ Require pull request reviews before merging
  - Required number of reviewers: 1
  - Dismiss stale reviews when new commits are pushed
- ✅ Require conversation resolution before merging
- ✅ Require signed commits
- ✅ Require linear history
- ✅ Include administrators
- ✅ Restrict pushes that create files larger than 100MB

## Feature Branch Workflow

Since you use random branch names for features:

1. No branch protection needed for feature branches
2. All protection rules apply to `main` branch only
3. PRs from any branch → main will trigger all checks

## Workflow Files

- `test.yml` - Basic test workflow for quick feedback
- `ci.yml` - Comprehensive CI pipeline with tests, linting, building, and security
- `pr.yml` - Pull request specific checks with coverage requirements
- `release.yml` - Automated release workflow for tags

## Coverage Requirements

The PR workflow requires:
- Minimum 80% test coverage
- All tests must pass
- Code must be properly formatted
- No linting errors

## Security

- Security scans run on every push
- Dependencies are verified
- SARIF results are uploaded to GitHub Security tab
