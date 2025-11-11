# Security Advisory: Overly Permissive Provisioner RBAC

## Issue

**Severity:** High
**Component:** `charts/supacontrol/templates/provisioner-rbac.yaml`
**Issue Type:** Security - Overly Permissive Permissions

## Description

The current ClusterRole for provisioner Jobs grants cluster-wide access to sensitive resources including secrets, configmaps, and all workload types across ALL namespaces. This violates the principle of least privilege and creates a significant security risk.

### Current Architecture (Insecure)

```
┌──────────────────────────────────────────────┐
│ Provisioner ServiceAccount                   │
│ + ClusterRole (cluster-wide permissions)     │
├──────────────────────────────────────────────┤
│ Can access:                                  │
│ - Secrets in ANY namespace ⚠️                │
│ - ConfigMaps in ANY namespace ⚠️             │
│ - Services, Pods, Deployments everywhere ⚠️  │
│ - Create/Delete namespaces                   │
└──────────────────────────────────────────────┘
```

**Risk:**  A compromised provisioner Job could:
- Read secrets from ANY namespace (including other tenants' credentials)
- Modify or delete resources in ANY namespace
- Create malicious workloads anywhere in the cluster
- Escalate privileges via namespace-scoped RBAC manipulation

## Recommended Solution

### Architecture Refactoring

**Phase 1: Controller Creates Namespaces**

Modify `server/controllers/job_helpers.go`:

1. **Controller creates namespace** before creating provisioning Job:
```go
func (r *SupabaseInstanceReconciler) reconcilePending(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) (ctrl.Result, error) {
    // Create namespace first
    namespace := &corev1.Namespace{
        ObjectMeta: metav1.ObjectMeta{
            Name: fmt.Sprintf("supa-%s", instance.Spec.ProjectName),
            Labels: map[string]string{
                "app.kubernetes.io/managed-by": "supacontrol",
                "supacontrol.io/instance":      instance.Spec.ProjectName,
            },
        },
    }
    if err := r.Create(ctx, namespace); err != nil && !apierrors.IsAlreadyExists(err) {
        return r.transitionToFailed(ctx, instance, fmt.Sprintf("Failed to create namespace: %v", err))
    }

    // Create namespace-scoped RBAC
    if err := r.createProvisionerRBAC(ctx, instance); err != nil {
        return r.transitionToFailed(ctx, instance, fmt.Sprintf("Failed to create RBAC: %v", err))
    }

    // Create Job (now operates in pre-created namespace)
    job, err := r.createProvisioningJob(ctx, instance)
    // ... rest of logic
}
```

2. **Controller creates Role + RoleBinding** per instance:
```go
func (r *SupabaseInstanceReconciler) createProvisionerRBAC(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
    namespace := fmt.Sprintf("supa-%s", instance.Spec.ProjectName)

    // Create namespace-scoped Role
    role := &rbacv1.Role{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "supacontrol-provisioner",
            Namespace: namespace,
        },
        Rules: []rbacv1.PolicyRule{
            {
                APIGroups: [""],
                Resources: ["secrets", "services", "configmaps", "pods", "serviceaccounts", "persistentvolumeclaims"],
                Verbs:     ["create", "delete", "get", "list", "patch", "update", "watch"],
            },
            {
                APIGroups: ["apps"],
                Resources: ["deployments", "statefulsets", "replicasets"],
                Verbs:     ["create", "delete", "get", "list", "patch", "update", "watch"],
            },
            {
                APIGroups: ["networking.k8s.io"],
                Resources: ["ingresses"],
                Verbs:     ["create", "delete", "get", "list", "patch", "update", "watch"],
            },
            {
                APIGroups: ["batch"],
                Resources: ["jobs"],
                Verbs:     ["create", "delete", "get", "list", "watch"],
            },
            {
                APIGroups: ["rbac.authorization.k8s.io"],
                Resources: ["roles", "rolebindings"],
                Verbs:     ["create", "delete", "get", "list", "patch", "update", "watch"],
            },
        },
    }

    if err := r.Create(ctx, role); err != nil && !apierrors.IsAlreadyExists(err) {
        return err
    }

    // Create RoleBinding
    roleBinding := &rbacv1.RoleBinding{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "supacontrol-provisioner",
            Namespace: namespace,
        },
        RoleRef: rbacv1.RoleRef{
            APIGroup: "rbac.authorization.k8s.io",
            Kind:     "Role",
            Name:     "supacontrol-provisioner",
        },
        Subjects: []rbacv1.Subject{
            {
                Kind:      "ServiceAccount",
                Name:      ServiceAccountName,
                Namespace: ControllerNamespace,
            },
        },
    }

    if err := r.Create(ctx, roleBinding); err != nil && !apierrors.IsAlreadyExists(err) {
        return err
    }

    return nil
}
```

3. **Update Job script** to remove namespace creation:
```bash
# Remove this section from job script:
# kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Namespace now pre-exists, Job just operates within it
```

**Phase 2: Minimal ClusterRole (Optional)**

If namespace deletion needs to remain in Jobs, keep a minimal ClusterRole:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: supacontrol-provisioner-minimal
rules:
# ONLY namespace delete (for cleanup Jobs)
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["delete", "get"]  # Minimal permissions
```

Or better: Have controller delete namespace after cleanup Job succeeds.

### Security Benefits

**After Refactoring:**

```
┌──────────────────────────────────────────────┐
│ Provisioner ServiceAccount                   │
│ + Namespace-scoped Role (per instance)       │
├──────────────────────────────────────────────┤
│ Can access:                                  │
│ - Resources ONLY in supa-<instance> namespace│
│ - No access to other namespaces ✅           │
│ - No cluster-wide permissions ✅             │
│ - Blast radius limited to single tenant ✅   │
└──────────────────────────────────────────────┘
```

**Benefits:**
- ✅ **Least Privilege:** Jobs can only affect their own namespace
- ✅ **Blast Radius Containment:** Compromised Job limited to one tenant
- ✅ **Multi-Tenancy:** Other instances' resources are protected
- ✅ **Audit Trail:** Clear per-namespace RBAC for compliance
- ✅ **Defense in Depth:** Even if Job is compromised, damage is contained

## Implementation Checklist

- [ ] Refactor controller to create namespace before Job
- [ ] Implement `createProvisionerRBAC()` function
- [ ] Update `createProvisioningJob()` to remove namespace creation
- [ ] Update `createCleanupJob()` to work with pre-existing namespace
- [ ] Update Helm chart to remove/minimize ClusterRole
- [ ] Update tests to verify namespace-scoped RBAC
- [ ] Security audit of new RBAC implementation
- [ ] Documentation updates
- [ ] Consider OPA/Kyverno policies to prevent privilege escalation

## Temporary Mitigation

Until the refactoring is complete:

1. **Network Policies:** Restrict provisioner Job network access
2. **Pod Security Standards:** Enforce restricted PSS on provisioner namespace
3. **RBAC Audit:** Monitor provisioner SA usage with audit logs
4. **Resource Quotas:** Limit resources provisioner can consume
5. **Documentation:** Warn users about current security model

## Timeline

- **Priority:** High
- **Target:** Next major release (v0.2.0)
- **Effort Estimate:** 2-3 days
  - Controller refactoring: 1 day
  - Testing: 0.5 days
  - Security review: 0.5 days
  - Documentation: 0.5 days

## References

- [Kubernetes RBAC Best Practices](https://kubernetes.io/docs/concepts/security/rbac-good-practices/)
- [Principle of Least Privilege](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [ADR-002: Job-based Provisioning Pattern](../../docs/adr/002-job-based-provisioning-pattern.md)
