# Security Guide

This document outlines security best practices and considerations for running SupaControl in production.

## Table of Contents

- [Security Best Practices](#security-best-practices)
  - [Secrets Management](#secrets-management)
  - [Network Security](#network-security)
  - [TLS/HTTPS](#tlshttps)
  - [API Security](#api-security)
  - [RBAC](#rbac)
- [Security Updates](#security-updates)
- [Reporting Security Issues](#reporting-security-issues)

## Security Best Practices

### Secrets Management

**DO:**
- ✅ Use strong, randomly generated secrets
- ✅ Store secrets in Kubernetes Secrets
- ✅ Rotate secrets regularly
- ✅ Use separate secrets for dev/staging/prod
- ✅ Limit access to secrets using RBAC

**DON'T:**
- ❌ Commit secrets to git
- ❌ Use default passwords in production
- ❌ Share secrets via insecure channels
- ❌ Reuse secrets across environments

### Network Security

```yaml
# Enable network policies for instance isolation
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
```

### TLS/HTTPS

**Always use TLS in production:**
```yaml
ingress:
  enabled: true
  tls:
    - secretName: supacontrol-tls
      hosts:
        - supacontrol.yourdomain.com
```

**Use cert-manager for automatic certificates:**
```yaml
ingress:
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
```

### API Security

- All endpoints require authentication (except health check and login)
- JWT tokens expire after 24 hours
- API keys can be revoked at any time
- Rate limiting recommended (use ingress annotations)

### RBAC

Review and minimize ServiceAccount permissions:

```bash
# View current permissions
kubectl describe clusterrole supacontrol

# Audit access
kubectl auth can-i --list --as=system:serviceaccount:supacontrol:supacontrol
```

## Security Updates

- Monitor [GitHub Security Advisories](https://github.com/qubitquilt/SupaControl/security/advisories)
- Keep dependencies updated: `go get -u ./...` and `npm update`
- Subscribe to Kubernetes security announcements
- Regularly review audit logs

## Reporting Security Issues

**DO NOT** open public issues for security vulnerabilities.

Instead, email: security@qubitquilt.io (if available) or open a [private security advisory](https://github.com/qubitquilt/SupaControl/security/advisories/new).

---

**Related Documentation:**
- [Deployment Guide](../README.md#deployment)
- [Configuration Guide](../README.md#configuration)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [CONTRIBUTING.md](../CONTRIBUTING.md)

**Last Updated: November 2025**
