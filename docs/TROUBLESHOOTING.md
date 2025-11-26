# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with SupaControl.

## Table of Contents

- [Common Issues](#common-issues)
  - [1. Installation Fails](#1-installation-fails)
  - [2. Database Connection Failures](#2-database-connection-failures)
  - [3. Instance Creation Fails](#3-instance-creation-fails)
  - [4. Dashboard Not Accessible](#4-dashboard-not-accessible)
  - [5. Authentication Issues](#5-authentication-issues)
  - [6. Helm Release Conflicts](#6-helm-release-conflicts)
- [Debug Mode](#debug-mode)
- [Getting Help](#getting-help)

## Common Issues

### 1. Installation Fails

**Symptom:** Helm install fails or pods crash

**Diagnosis:**
```bash
# Check Helm release status
helm list -n supacontrol

# Check pod status
kubectl get pods -n supacontrol

# View pod logs
kubectl logs -n supacontrol -l app.kubernetes.io/name=supacontrol

# Describe pod for events
kubectl describe pod -n supacontrol <pod-name>
```

**Common Causes:**
- Missing required values (JWT_SECRET, DB_PASSWORD)
- Insufficient RBAC permissions
- Image pull failures
- Resource constraints

**Solutions:**
```bash
# Delete and reinstall with correct values
helm uninstall supacontrol -n supacontrol
helm install supacontrol ./charts/supacontrol -f values.yaml -n supacontrol

# Check resource availability
kubectl top nodes
kubectl describe node <node-name>
```

### 2. Database Connection Failures

**Symptom:** Server logs show "connection refused" or "authentication failed"

**Diagnosis:**
```bash
# Check PostgreSQL pod
kubectl get pods -n supacontrol -l app.kubernetes.io/name=postgresql

# View PostgreSQL logs
kubectl logs -n supacontrol -l app.kubernetes.io/name=postgresql

# Test connection from SupaControl pod
kubectl exec -it -n supacontrol deployment/supacontrol -- /bin/sh
# Inside pod:
psql -h $DB_HOST -U $DB_USER -d $DB_NAME
```

**Solutions:**
- Verify database credentials in values.yaml
- Check PostgreSQL pod is running
- Verify service name matches DB_HOST
- Check network policies aren't blocking traffic

### 3. Instance Creation Fails

**Symptom:** Instance stuck in "Provisioning" or "Failed" phase.

**Diagnosis:**
```bash
# 1. Check the SupabaseInstance CR status
kubectl get supabaseinstance <instance-name> -o yaml

# 2. Check the logs of the SupaControl controller
kubectl logs -n supacontrol-system -l app.kubernetes.io/name=supacontrol -f

# 3. Check if the instance namespace was created
kubectl get namespace supa-<instance-name>

# 4. Check the status of the provisioning Job
kubectl get job -n supa-<instance-name> -l supacontrol.qubitquilt.com/job-type=provision

# 5. Describe the Job to see events and potential errors
kubectl describe job -n supa-<instance-name> <job-name>

# 6. Check the logs of the provisioning Job's pod
kubectl logs -n supa-<instance-name> -l job-name=<job-name>
```

**Common Causes:**
- Insufficient cluster resources (CPU, memory, or storage).
- Helm chart repository is unreachable from the cluster.
- Incorrect Helm chart version specified.
- Network policies blocking the provisioning job.
- RBAC permission issues for the provisioner ServiceAccount within the namespace.

**Solutions:**
```bash
# Check resource availability in the cluster
kubectl top nodes

# Verify Helm chart accessibility from a pod inside the cluster
kubectl run -it --rm --restart=Never --image=alpine/helm:latest helm-test -- /bin/sh
# Inside pod:
helm repo add supabase https://supabase.github.io/helm-charts
helm repo update
helm pull supabase/supabase --version <version>

# Review the RBAC roles for the provisioner in the instance namespace
kubectl get role,rolebinding -n supa-<instance-name>
```

### 4. Dashboard Not Accessible

**Symptom:** Cannot access SupaControl dashboard URL

**Diagnosis:**
```bash
# Check ingress configuration
kubectl get ingress -n supacontrol
kubectl describe ingress -n supacontrol supacontrol

# Check ingress controller
kubectl get pods -n ingress-nginx

# Check service
kubectl get svc -n supacontrol
```

**Solutions:**

**Option 1: Port Forward (Temporary)**
```bash
kubectl port-forward -n supacontrol svc/supacontrol 8091:8091
# Access at http://localhost:8091
```

**Option 2: Fix Ingress**
```bash
# Verify DNS points to ingress controller
nslookup supacontrol.yourdomain.com

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx

# Verify ingress class
kubectl get ingressclass
```

**Option 3: Fix TLS Certificate**
```bash
# Check certificate status
kubectl get certificate -n supacontrol

# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager

# Describe certificate for events
kubectl describe certificate -n supacontrol supacontrol-tls
```

### 5. Authentication Issues

**Symptom:** "Unauthorized" errors or login fails

**Diagnosis:**
```bash
# Check if JWT_SECRET is set correctly
kubectl get secret -n supacontrol supacontrol -o jsonpath='{.data.JWT_SECRET}' | base64 -d

# Test login endpoint
curl -X POST https://supacontrol.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' \
  -v
```

**Solutions:**
- Verify JWT_SECRET is set and consistent
- Check password hasn't been changed from default
- Clear browser cache/cookies
- Verify API key hasn't been revoked

### 6. Helm Release Conflicts

**Symptom:** "release already exists" errors

**Solutions:**
```bash
# List existing releases
helm list -A

# Delete existing release
helm uninstall supacontrol -n supacontrol

# Clean up resources
kubectl delete namespace supacontrol

# Reinstall
helm install supacontrol ./charts/supacontrol -f values.yaml -n supacontrol
```

## Debug Mode

Enable debug logging:

```yaml
# values.yaml
env:
  - name: LOG_LEVEL
    value: "debug"
```

## Getting Help

If you're still stuck:

1. **Check logs** with maximum verbosity:
   ```bash
   kubectl logs -n supacontrol -l app.kubernetes.io/name=supacontrol --all-containers=true --tail=500
   ```

2. **Gather diagnostic info**:
   ```bash
   kubectl get all -n supacontrol -o yaml > diagnostics.yaml
   helm get values supacontrol -n supacontrol > current-values.yaml
   ```

3. **Open an issue** with:
   - SupaControl version
   - Kubernetes version (`kubectl version`)
   - Error messages and logs
   - Steps to reproduce
   - Diagnostic files (redact secrets!)

4. **Check existing issues**:
   [github.com/qubitquilt/SupaControl/issues](https://github.com/qubitquilt/SupaControl/issues)

---

**Related Documentation:**
- [Installation Guide](../README.md#installation)
- [Configuration Guide](../README.md#configuration)
- [Security Guide](./SECURITY.md)
- [Development Guide](./DEVELOPMENT.md)

**Last Updated: November 2025**
