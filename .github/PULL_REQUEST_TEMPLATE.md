# Pull Request

## Description

<!-- Provide a clear and concise description of your changes -->

## Type of Change

<!-- Mark the relevant option with an 'x' -->

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Code refactoring (no functional changes)
- [ ] Performance improvement
- [ ] Test coverage improvement
- [ ] CI/CD improvement
- [ ] Other (please describe):

## Related Issues

<!-- Link to related issues using GitHub keywords -->
<!-- Examples: Fixes #123, Closes #456, Related to #789 -->

- Fixes #
- Related to #

## Changes Made

<!-- List the specific changes made in this PR -->

-
-
-

## Testing

### Test Coverage

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed
- [ ] No tests needed (documentation, minor changes)

### Test Results

<!-- Describe the testing you've done -->

**Backend Tests:**
```bash
cd server
go test ./...

# Output:
# Paste test results here
```

**Frontend Tests:**
```bash
cd ui
npm test

# Output:
# Paste test results here
```

**Manual Testing:**

<!-- Describe manual testing steps and results -->

1. Tested scenario: [description]
   - Result: [pass/fail]
2. Tested scenario: [description]
   - Result: [pass/fail]

## Screenshots (if applicable)

<!-- Add screenshots to demonstrate UI changes or new features -->

## Deployment Notes

<!-- Any special deployment considerations? -->
<!-- Examples: database migrations, configuration changes, breaking changes -->

- [ ] Requires database migration
- [ ] Requires configuration changes
- [ ] Requires Helm chart update
- [ ] Breaking API changes (requires version bump)
- [ ] No special deployment steps needed

**Deployment instructions:**

```bash
# If applicable, provide deployment commands
```

## Documentation

- [ ] Updated README.md
- [ ] Updated CONTRIBUTING.md
- [ ] Updated API documentation
- [ ] Updated ARCHITECTURE.md
- [ ] Updated code comments
- [ ] Created/updated tests documentation
- [ ] No documentation needed

## Performance Impact

<!-- Does this change affect performance? -->

- [ ] No performance impact
- [ ] Performance improved
- [ ] Performance degraded (explained below)

**Performance notes:**

<!-- If performance is affected, provide details -->

## Security Considerations

<!-- Does this change have security implications? -->

- [ ] No security impact
- [ ] Security improved
- [ ] Requires security review

**Security notes:**

<!-- If there are security implications, explain them -->

## Breaking Changes

<!-- If this PR includes breaking changes, describe them -->

**Breaking changes:**

-

**Migration guide:**

```bash
# Provide instructions for users to migrate
```

## Checklist

<!-- Mark completed items with an 'x' -->

### Code Quality

- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] My changes generate no new warnings
- [ ] I have run `gofmt` (for Go code) or `npm run lint` (for JavaScript)

### Testing

- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] I have tested this on a real Kubernetes cluster
- [ ] I have verified that instances can still be created/deleted

### Documentation

- [ ] I have updated the documentation accordingly
- [ ] I have added/updated code comments where necessary
- [ ] I have updated the CHANGELOG (if applicable)

### Commit Standards

- [ ] My commits follow conventional commit format (e.g., `feat:`, `fix:`, `docs:`)
- [ ] My commit messages are clear and descriptive
- [ ] I have squashed/rebased my commits appropriately

### Dependencies

- [ ] I have not introduced unnecessary dependencies
- [ ] All new dependencies are documented with rationale
- [ ] I have updated `go.mod`/`package.json` as needed

### Collaboration

- [ ] I have assigned appropriate reviewers
- [ ] I have added relevant labels
- [ ] I have linked related issues
- [ ] I am available for review feedback and discussions

## Additional Notes

<!-- Any additional information for reviewers -->

## Review Checklist (for maintainers)

- [ ] Code quality meets standards
- [ ] Tests are sufficient and passing
- [ ] Documentation is complete and accurate
- [ ] No security concerns
- [ ] No performance regressions
- [ ] Breaking changes are justified and documented
- [ ] Commit history is clean

---

**Thank you for contributing to SupaControl!** 🎉

We appreciate your time and effort in improving the project. Our maintainers will review your PR as soon as possible.
