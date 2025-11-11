---
name: Bug Report
about: Create a report to help us improve SupaControl
title: '[BUG] '
labels: bug
assignees: ''
---

## Bug Description

A clear and concise description of what the bug is.

## Steps To Reproduce

Steps to reproduce the behavior:

1. Go to '...'
2. Click on '...'
3. Execute command '...'
4. See error

## Expected Behavior

A clear and concise description of what you expected to happen.

## Actual Behavior

A clear and concise description of what actually happened.

## Environment

**SupaControl Version:**
- Version: [e.g., v0.1.0, latest, commit SHA]

**Kubernetes:**
- Kubernetes Version: [e.g., v1.28.0]
- Installation Method: [e.g., CLI installer, manual Helm]
- Helm Version: [e.g., v3.13.0]

**Operating System:**
- OS: [e.g., Ubuntu 22.04, macOS 14.0]
- Architecture: [e.g., amd64, arm64]

**Additional Context:**
- Ingress Controller: [e.g., nginx-ingress, traefik]
- Certificate Manager: [e.g., cert-manager, none]
- Cloud Provider: [e.g., AWS, GCP, Azure, bare-metal]

## Logs

<details>
<summary>SupaControl Logs</summary>

```
kubectl logs -n supacontrol -l app.kubernetes.io/name=supacontrol --tail=100

# Paste logs here
```

</details>

<details>
<summary>Additional Logs (if applicable)</summary>

```
# Paste relevant logs here
# e.g., PostgreSQL logs, ingress controller logs, etc.
```

</details>

## Screenshots

If applicable, add screenshots to help explain your problem.

## Additional Context

Add any other context about the problem here. For example:
- Does the issue occur consistently or intermittently?
- Have you made any custom modifications to the deployment?
- Are there any workarounds you've discovered?

## Possible Solution

If you have an idea of what might be causing this or how to fix it, please share.

## Checklist

- [ ] I have searched existing issues to ensure this is not a duplicate
- [ ] I have provided all requested information
- [ ] I have included relevant logs
- [ ] I am running a supported version of Kubernetes (v1.24+)
- [ ] I have checked the [troubleshooting guide](../README.md#troubleshooting)
