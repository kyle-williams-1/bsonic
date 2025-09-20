# CI/CD Setup

This document explains the streamlined CI/CD pipeline for the BSON library.

## Workflow Overview

### 1. **CI Workflow** (`ci.yml`)
**Triggers**: Push/PR to `main` or `develop` branches

**Jobs**:
- **Test**: Unit tests + coverage (library code only)
- **Lint**: Code quality checks with golangci-lint
- **Build**: Multi-platform build verification
- **Security**: Security scanning with Gosec

### 2. **Integration Workflow** (`integration.yml`)
**Triggers**: Push/PR to `main` or `develop` branches, manual dispatch

**Jobs**:
- **Integration Tests**: MongoDB integration tests with real database
- **Coverage**: Integration test coverage reporting

### 3. **Release Workflow** (`release.yml`)
**Triggers**: Tag creation (releases)

**Jobs**:
- **Release**: Build and publish releases

## Why This Setup?

### **Before (Problems)**:
- ❌ 5 separate workflow files
- ❌ Redundant test execution
- ❌ Wasted CI minutes
- ❌ Inconsistent configurations
- ❌ Maintenance overhead

### **After (Benefits)**:
- ✅ 3 focused workflows
- ✅ No redundant test execution
- ✅ Efficient resource usage
- ✅ Consistent behavior
- ✅ Easy maintenance

## Workflow Details

### CI Workflow
```yaml
# Runs on every push/PR
- Unit tests (all packages)
- Coverage (library code only)
- Linting
- Multi-platform builds
- Security scanning
```

### Integration Workflow
```yaml
# Runs on every push/PR + manual
- MongoDB integration tests
- Integration coverage
- Real database validation
```

### Release Workflow
```yaml
# Runs on tag creation
- Build verification
- Release preparation
```

## Coverage Strategy

### **Library Coverage** (CI Workflow)
- **Scope**: Main library code only (`.`)
- **Purpose**: Track library code quality
- **Excludes**: Examples, tests, integration tests

### **Integration Coverage** (Integration Workflow)
- **Scope**: Integration test code only
- **Purpose**: Track integration test coverage
- **Includes**: Integration test scenarios

## Best Practices

### **1. Efficient Testing**
- Unit tests run once in CI
- Integration tests run separately
- No duplicate test execution

### **2. Focused Workflows**
- Each workflow has a specific purpose
- Clear separation of concerns
- Easy to understand and maintain

### **3. Resource Optimization**
- Tests run in parallel where possible
- Caching for dependencies
- Minimal redundant execution

### **4. Coverage Accuracy**
- Library coverage excludes test files
- Integration coverage tracks test scenarios
- Clear separation of concerns

## Local Development

### **Run All Tests**
```bash
make test-all
```

### **Run Unit Tests Only**
```bash
make test
```

### **Run Integration Tests Only**
```bash
make test-integration
```

### **Generate Coverage**
```bash
make coverage
```

## Troubleshooting

### **CI Failures**
1. Check the specific workflow that failed
2. Look at the job logs for details
3. Run tests locally to reproduce

### **Coverage Issues**
1. Ensure coverage runs on library code only (`.`)
2. Check that test files are excluded
3. Verify coverage thresholds

### **Integration Test Failures**
1. Check MongoDB container status
2. Verify test data seeding
3. Check network connectivity

## Maintenance

### **Adding New Tests**
- Unit tests: Add to `bsonic_test.go`
- Integration tests: Add to `integration/integration_test.go`

### **Updating Workflows**
- CI changes: Modify `ci.yml`
- Integration changes: Modify `integration.yml`
- Release changes: Modify `release.yml`

### **Coverage Updates**
- Library coverage: Update CI workflow
- Integration coverage: Update integration workflow

## Summary

The streamlined CI setup provides:
- **Efficiency**: No redundant test execution
- **Clarity**: Each workflow has a specific purpose
- **Maintainability**: Easy to understand and modify
- **Accuracy**: Proper coverage tracking
- **Speed**: Faster CI execution

This approach follows industry best practices for Go projects and provides a solid foundation for continuous integration and deployment.