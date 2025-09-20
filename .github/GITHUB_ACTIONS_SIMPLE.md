# GitHub Actions - Simple Guide

## What Are GitHub Actions?

GitHub Actions are **automated workflows** that run when certain events happen in your repository (like pushing code or creating a PR).

## Your Workflow

You'll work like this:
1. Create a feature branch (any name): `git checkout -b my-feature`
2. Make changes and commit: `git commit -m "Add new feature"`
3. Push and create PR: `git push origin my-feature`
4. GitHub automatically runs tests
5. If tests pass, merge to main

## What Happens Automatically

### When you push to main:
- ✅ Run all tests
- ✅ Check code quality
- ✅ Build the project
- ✅ Security scan

### When you create a PR:
- ✅ Run tests on your branch
- ✅ Check code formatting
- ✅ Verify dependencies
- ✅ Check test coverage (80% minimum)

## Workflow Files Explained

### `test.yml` - Quick Tests
```yaml
on:
  push:
    branches: [ main ]      # Run when pushing to main
  pull_request:
    branches: [ main ]      # Run when creating PRs to main
```
**What it does**: Runs tests quickly on multiple Go versions

### `ci.yml` - Full Pipeline
```yaml
on:
  push:
    branches: [ main ]      # Run when pushing to main
  pull_request:
    branches: [ main ]      # Run when creating PRs to main
```
**What it does**: 
- Tests on Go 1.22, 1.23, 1.24, 1.25
- Lints your code
- Builds for different platforms
- Scans for security issues

### `pr.yml` - PR Checks
```yaml
on:
  pull_request:
    types: [opened, synchronize, reopened]  # Only on PRs
```
**What it does**: 
- Checks code formatting
- Runs tests
- Ensures 80% test coverage
- Validates dependencies

### `release.yml` - Auto Releases
```yaml
on:
  push:
    tags:
      - 'v*'              # Run when you create tags like v1.0.0
```
**What it does**: 
- Creates GitHub releases automatically
- Runs tests before release
- Generates changelog

## Key Concepts

### `branches: [ main ]`
- Only runs on the `main` branch
- Your feature branches can have any name
- PRs from any branch → main will trigger tests

### `runs-on: ubuntu-latest`
- Runs on GitHub's Linux servers
- You don't need to set up anything

### `strategy.matrix`
- Tests multiple Go versions simultaneously
- If any version fails, the whole job fails

### `steps`
- Each step does one thing
- Steps run in order
- If a step fails, the job stops

## What You Need to Do

### Nothing! 🎉
The workflows are already set up. Just:

1. **Push your code** - Tests run automatically
2. **Create PRs** - Tests run automatically  
3. **Merge to main** - Tests run automatically
4. **Create tags** - Releases happen automatically

### Optional: Check Status
- Go to the "Actions" tab in GitHub
- See which workflows are running
- Click on a workflow to see details

## Common Scenarios

### ✅ Everything Works
- Push code → Green checkmarks → Merge PR

### ❌ Tests Fail
- Push code → Red X → Fix code → Push again

### 🔄 Tests Running
- Push code → Yellow circle → Wait for completion

## Branch Protection (Optional)

You can set up rules so PRs can't be merged unless tests pass:

1. Go to Settings → Branches
2. Add rule for `main` branch
3. Check "Require status checks to pass before merging"
4. Select the test workflows

This prevents broken code from being merged!

## Need Help?

- Check the Actions tab for detailed logs
- Run `make pre-commit` locally to test before pushing
- All workflows are in `.github/workflows/` folder
