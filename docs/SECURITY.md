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

SupaControl is designed to follow the principle of least privilege.

**CRITICAL**: A security advisory **[ADVISORY-001-provisioner-rbac.md](./security/ADVISORY-001-provisioner-rbac.md)** was issued regarding overly permissive RBAC roles in early versions. The architecture has been updated to a more secure, two-tiered model. Ensure your deployment uses this model.

#### Secure RBAC Model

1.  **Controller `ClusterRole`**: The main SupaControl controller runs with a `ClusterRole` that is tightly scoped. It only has permissions to manage the `SupabaseInstance` CRDs and the RBAC resources for its child instances. It **cannot** access secrets or other resources inside the instance namespaces.

2.  **Provisioner `Role` (Namespace-Scoped)**: For each Supabase instance, the controller creates a `Role` and `RoleBinding` that are scoped **only to that instance's namespace**. The provisioning `Job` that creates the Supabase stack uses a `ServiceAccount` bound to this limited role.

This design ensures that the blast radius of a compromised provisioning job is contained to a single tenant's namespace, which is a critical security feature for a multi-tenant platform.

#### Auditing RBAC

Regularly audit the permissions of the SupaControl components.

```bash
# 1. Audit the main controller's ClusterRole (should be limited)
kubectl describe clusterrole supacontrol-controller-manager

# 2. Audit the ServiceAccount used by the controller pods
kubectl describe serviceaccount supacontrol-controller-manager -n supacontrol-system

# 3. Audit the per-instance Role for a sample instance (should be namespace-scoped)
kubectl describe role supacontrol-provisioner -n supa-my-app

# 4. Verify the provisioner ServiceAccount has no cluster-wide permissions
# Note: The 'supacontrol-provisioner' ServiceAccount is created in each instance namespace,
# so you must specify a namespace.
kubectl auth can-i --list --as=system:serviceaccount:<instance-namespace>:supacontrol-provisioner
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
- [Deployment Guide](./DEPLOYMENT.md)
- [Configuration Guide](../README.md#configuration)
- [Troubleshooting Guide](./TROUBLESHOOTING.md)
- [CONTRIBUTING.md](../CONTRIBUTING.md)

**Last Updated: November 2025**
