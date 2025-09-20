# GitHub CI/CD Setup Guide

This document explains how to set up the GitHub Actions workflows for the bsonic project.

## Workflows Created

### 1. `test.yml` - Basic Test Workflow
- **Triggers**: Push to main/develop, Pull Requests
- **Purpose**: Quick test feedback
- **Features**:
  - Tests on Go 1.22, 1.23, 1.24, 1.25
  - Caches dependencies
  - Runs tests with coverage
  - Uploads coverage to Codecov

### 2. `ci.yml` - Comprehensive CI Pipeline
- **Triggers**: Push to main/develop, Pull Requests
- **Purpose**: Full CI pipeline
- **Features**:
  - **Test Job**: Multi-version testing (Go 1.22-1.25)
  - **Lint Job**: Code quality checks with golangci-lint
  - **Build Job**: Cross-platform builds (Linux, macOS, Windows)
  - **Security Job**: Security scanning with Gosec

### 3. `pr.yml` - Pull Request Checks
- **Triggers**: Pull Requests only
- **Purpose**: PR-specific validation
- **Features**:
  - Format checking
  - Go vet
  - Test execution
  - Coverage requirements (80% minimum)
  - Dependency verification

### 4. `release.yml` - Automated Releases
- **Triggers**: Git tags (v*)
- **Purpose**: Automated release creation
- **Features**:
  - Runs tests before release
  - Generates changelog
  - Creates GitHub release

## Setup Instructions

### 1. Enable GitHub Actions
1. Go to your repository on GitHub
2. Click on "Actions" tab
3. Enable GitHub Actions if prompted

### 2. Set Up Branch Protection
1. Go to Settings → Branches
2. Add rule for `main` branch
3. Configure required status checks:
   - `Test` (from test.yml)
   - `Lint` (from ci.yml)
   - `Build` (from ci.yml)
   - `Security Scan` (from ci.yml)

### 3. Optional: Set Up Codecov
1. Go to [codecov.io](https://codecov.io)
2. Sign in with GitHub
3. Add your repository
4. The coverage will be automatically uploaded

### 4. Optional: Set Up Security Alerts
1. Go to Settings → Security & analysis
2. Enable "Dependency graph"
3. Enable "Dependabot alerts"
4. Enable "Dependabot security updates"

## Local Development

### Pre-commit Checks
Run the same checks locally that CI will run:

```bash
# Run all pre-commit checks
make pre-commit

# Or run the script directly
./scripts/test-local.sh
```

### Available Make Targets
```bash
make test          # Run tests
make test-coverage # Run tests with coverage
make fmt           # Format code
make lint          # Run linter
make check         # Run fmt, test, lint
make pre-commit    # Run all CI checks locally
make security      # Run security scan
make check-all     # Run all checks including security
```

## Workflow Status Badges

The README includes status badges that will show:
- CI status
- Test status
- Go Report Card
- License

These will automatically update once the workflows are running.

## Coverage Requirements

- **Minimum coverage**: 80%
- **Current coverage**: 85.5%
- **Coverage reports**: Available in PR checks and Codecov

## Security Features

- **Dependency verification**: All dependencies are verified
- **Security scanning**: Gosec scans for security issues
- **SARIF upload**: Security results uploaded to GitHub Security tab
- **Dependabot**: Automated dependency updates

## Troubleshooting

### Common Issues

1. **Workflow not running**: Check if GitHub Actions is enabled
2. **Tests failing**: Run `make pre-commit` locally first
3. **Coverage too low**: Add more tests or adjust coverage threshold
4. **Linting errors**: Run `make lint` to see issues

### Getting Help

- Check the Actions tab for detailed logs
- Run local checks with `make pre-commit`
- Review the workflow files in `.github/workflows/`
