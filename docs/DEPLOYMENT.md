# Deployment Guide

Complete guide for deploying SupaControl to production.

## Table of Contents

- [Production Deployment Checklist](#production-deployment-checklist)
- [High Availability Setup](#high-availability-setup)
- [Kubernetes RBAC](#kubernetes-rbac)
- [Monitoring with Prometheus](#monitoring-with-prometheus)
- [Backup and Disaster Recovery](#backup-and-disaster-recovery)
- [Scaling](#scaling)
- [Upgrades](#upgrades)

## Production Deployment Checklist

Before deploying to production, ensure you've completed all items:

### Security

- [ ] **Change default admin password** immediately after first login
- [ ] **Generate strong JWT secret** (64+ characters, cryptographically random)
- [ ] **Use strong database passwords** (32+ characters, mix of characters)
- [ ] **Enable TLS/HTTPS** on all endpoints (use cert-manager)
- [ ] **Review RBAC permissions** (minimize permissions to least privilege)
- [ ] **Enable network policies** for namespace isolation
- [ ] **Never commit secrets** to version control

### Reliability

- [ ] **Configure resource limits and requests** for all pods
- [ ] **Set up pod autoscaling** (HPA) if expected traffic is variable
- [ ] **Configure multiple replicas** (minimum 3 for HA)
- [ ] **Set up liveness and readiness probes** (included in Helm chart)
- [ ] **Configure persistent volumes** for PostgreSQL
- [ ] **Enable PostgreSQL replication** for database HA

### Observability

- [ ] **Configure logging aggregation** (e.g., ELK, Loki)
- [ ] **Set up metrics collection** (Prometheus)
- [ ] **Configure alerting** (Alertmanager)
- [ ] **Enable audit logging** for compliance
- [ ] **Set up distributed tracing** (optional: Jaeger)

### Backup

- [ ] **Schedule database backups** (daily minimum)
- [ ] **Test backup restoration** regularly
- [ ] **Document disaster recovery procedures**
- [ ] **Store backups off-cluster** (S3, GCS, Azure Blob)
- [ ] **Implement backup retention policy** (e.g., 30 days)

### Monitoring

- [ ] **Monitor API response times** (target: <100ms)
- [ ] **Track instance creation/deletion rates**
- [ ] **Monitor Kubernetes resource usage** (CPU, memory, disk)
- [ ] **Set up health check alerts** for critical services
- [ ] **Monitor database connections** and query performance

## High Availability Setup

For production deployments, configure SupaControl for high availability.

### HA Configuration

Create an `ha-values.yaml` file:

```yaml
# High Availability Configuration
replicaCount: 3  # Minimum 3 replicas for HA

# Pod anti-affinity to spread across nodes
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                  - supacontrol
          topologyKey: kubernetes.io/hostname

# Resource limits (prevent resource starvation)
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

# Health checks (already included, but can be tuned)
livenessProbe:
  httpGet:
    path: /healthz
    port: 8091
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /healthz
    port: 8091
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3

# Horizontal Pod Autoscaling
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80

# PostgreSQL High Availability
postgresql:
  enabled: true
  architecture: replication  # Master-slave replication
  replication:
    enabled: true
    numSynchronousReplicas: 1
  persistence:
    enabled: true
    size: 20Gi
    storageClass: ""  # Use default or specify
  resources:
    limits:
      cpu: 2000m
      memory: 2Gi
    requests:
      cpu: 1000m
      memory: 1Gi

# Pod Disruption Budget (prevent too many pods down at once)
podDisruptionBudget:
  enabled: true
  minAvailable: 2  # At least 2 pods must be available

# Update strategy
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0  # Zero downtime updates
```

### Deploy with HA

```bash
helm upgrade --install supacontrol ./charts/supacontrol \
  -f ha-values.yaml \
  -n supacontrol \
  --create-namespace \
  --wait
```

### Verify HA Deployment

```bash
# Check all pods are running
kubectl get pods -n supacontrol

# Should see 3+ SupaControl pods
NAME                          READY   STATUS    RESTARTS   AGE
supacontrol-5d4f8c9b7-abc12   1/1     Running   0          5m
supacontrol-5d4f8c9b7-def34   1/1     Running   0          5m
supacontrol-5d4f8c9b7-ghi56   1/1     Running   0          5m

# Check HPA status
kubectl get hpa -n supacontrol

# Check PDB status
kubectl get pdb -n supacontrol
```

## Kubernetes RBAC

SupaControl follows the principle of least privilege by using a two-tiered RBAC model. This approach avoids granting broad, cluster-wide permissions to the main controller and provisioning jobs, significantly reducing the security risk.

### 1. Controller RBAC (`ClusterRole`)

The main SupaControl controller runs with a `ClusterRole` that grants it only the permissions needed to manage `SupabaseInstance` CRDs, create namespaces, and manage RBAC resources within those namespaces.

```yaml
# Simplified main controller ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: supacontrol-controller-manager
rules:
  - apiGroups: ["supacontrol.qubitquilt.com"]
    resources: ["supabaseinstances", "supabaseinstances/finalizers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["create", "get", "list", "watch"] # Note: delete is handled by finalizer logic
  - apiGroups: ["rbac.authorization.k8s.io"]
    resources: ["roles", "rolebindings"]
    verbs: ["create", "get", "list", "watch"]
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["create", "get", "list", "watch", "delete"]
```

### 2. Provisioning Job RBAC (`Role` - Per-Instance)

For each Supabase instance, the controller creates a dedicated, namespace-scoped `Role` and `RoleBinding`. The provisioning `Job` for that instance is then bound to this role, limiting its permissions to **only its own namespace**.

This ensures that a compromised provisioning Job for one instance **cannot** access resources in any other namespace.

```yaml
# Example Role created by the controller for a single instance
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: supacontrol-provisioner
  namespace: supa-my-app # Scoped to the instance's namespace
rules:
  # Permissions to manage resources ONLY within this namespace
  - apiGroups: ["", "apps", "networking.k8s.io", "batch"]
    resources: ["secrets", "services", "deployments", "ingresses", "jobs", ...]
    verbs: ["create", "delete", "get", "list", "update", "watch"]
```

### Security Best Practices

**Principle of Least Privilege:**
- The main controller has limited cluster-wide permissions.
- Provisioning Jobs have **zero** cluster-wide permissions.
- The blast radius of a compromised provisioning Job is contained to a single instance's namespace.

**Audit RBAC:**

```bash
# View the main controller's ClusterRole
kubectl describe clusterrole supacontrol-controller-manager

# To check the permissions of a provisioning job, first find the Role and RoleBinding in the instance's namespace
kubectl get role -n <instance-namespace>
kubectl get rolebinding -n <instance-namespace>
```

## Monitoring with Prometheus

Enable Prometheus metrics for SupaControl.

### Enable Metrics

```yaml
# values.yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    scrapeTimeout: 10s
```

### Metrics Exposed

SupaControl exposes these metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `supacontrol_instances_total` | Gauge | Total number of instances managed |
| `supacontrol_api_requests_total` | Counter | Total API requests by endpoint and status |
| `supacontrol_api_request_duration_seconds` | Histogram | API request latency |
| `supacontrol_instance_creation_duration_seconds` | Histogram | Time to create instances |
| `supacontrol_database_connections` | Gauge | Active database connections |
| `supacontrol_instance_status` | Gauge | Instance status (0=pending, 1=running, 2=failed) |

### Example Prometheus Queries

```promql
# Average API response time
rate(supacontrol_api_request_duration_seconds_sum[5m]) /
rate(supacontrol_api_request_duration_seconds_count[5m])

# Total instances by status
sum(supacontrol_instance_status) by (status)

# API error rate
rate(supacontrol_api_requests_total{status=~"5.."}[5m])

# P95 instance creation time
histogram_quantile(0.95,
  rate(supacontrol_instance_creation_duration_seconds_bucket[5m]))
```

### Grafana Dashboard

Import the SupaControl dashboard:

```bash
# Dashboard ID: Coming soon
# Or create custom dashboard with above queries
```

## Backup and Disaster Recovery

### Database Backups

**Automated Backups with pg_dump:**

```yaml
# backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: supacontrol-backup
  namespace: supacontrol
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:14
            env:
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: supacontrol-postgresql
                  key: password
            command:
            - /bin/sh
            - -c
            - |
              pg_dump -h supacontrol-postgresql \
                -U supacontrol \
                -d supacontrol \
                -F c \
                -f /backup/supacontrol-$(date +%Y%m%d).dump

              # Upload to S3 (example)
              aws s3 cp /backup/supacontrol-$(date +%Y%m%d).dump \
                s3://my-backups/supacontrol/

              # Delete backups older than 30 days
              aws s3 ls s3://my-backups/supacontrol/ | \
                awk '{print $4}' | \
                while read file; do
                  age=$(( ($(date +%s) - $(date -d "${file:12:8}" +%s)) / 86400 ))
                  if [ $age -gt 30 ]; then
                    aws s3 rm "s3://my-backups/supacontrol/$file"
                  fi
                done
            volumeMounts:
            - name: backup
              mountPath: /backup
          restartPolicy: OnFailure
          volumes:
          - name: backup
            emptyDir: {}
```

**Using Velero (Recommended):**

```bash
# Install Velero
velero install \
  --provider aws \
  --bucket my-velero-backups \
  --backup-location-config region=us-west-2 \
  --snapshot-location-config region=us-west-2 \
  --secret-file ./credentials-velero

# Create backup schedule
velero schedule create supacontrol-daily \
  --schedule="0 2 * * *" \
  --include-namespaces supacontrol

# Manual backup
velero backup create supacontrol-manual \
  --include-namespaces supacontrol
```

### Disaster Recovery Procedure

**1. Restore from Velero:**

```bash
# List backups
velero backup get

# Restore latest backup
velero restore create --from-backup supacontrol-daily-20250115

# Check restore status
velero restore describe supacontrol-daily-20250115
```

**2. Restore from pg_dump:**

```bash
# Download backup from S3
aws s3 cp s3://my-backups/supacontrol/supacontrol-20250115.dump .

# Restore to database
pg_restore -h supacontrol-postgresql \
  -U supacontrol \
  -d supacontrol \
  -c \
  supacontrol-20250115.dump
```

**3. Verify Restoration:**

```bash
# Check pods are running
kubectl get pods -n supacontrol

# Check for SupabaseInstance CRs
kubectl get supabaseinstances --all-namespaces

# Test API
curl https://supacontrol.example.com/healthz
```

## Scaling

### Horizontal Scaling

**Manual Scaling:**

```bash
# Scale to 5 replicas
kubectl scale deployment supacontrol \
  -n supacontrol \
  --replicas=5

# Verify
kubectl get pods -n supacontrol
```

**Auto-Scaling (HPA):**

```yaml
# Already configured in HA values
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
```

**Monitor Scaling:**

```bash
# Watch HPA decisions
kubectl get hpa -n supacontrol --watch

# View HPA details
kubectl describe hpa supacontrol -n supacontrol
```

### Database Scaling

**Vertical Scaling (increase resources):**

```yaml
postgresql:
  resources:
    limits:
      cpu: 4000m
      memory: 8Gi
    requests:
      cpu: 2000m
      memory: 4Gi
```

**Read Replicas:**

```yaml
postgresql:
  architecture: replication
  replication:
    enabled: true
    numSynchronousReplicas: 2  # Increase replicas
```

## Upgrades

### Upgrade Procedure

**1. Check Current Version:**

```bash
helm list -n supacontrol
```

**2. Backup Before Upgrade:**

```bash
velero backup create supacontrol-pre-upgrade \
  --include-namespaces supacontrol
```

**3. Update Helm Chart:**

```bash
# Pull latest chart
git pull origin main

# Review changes
git diff v0.1.0..v0.2.0

# Check for breaking changes
cat CHANGELOG.md
```

**4. Perform Upgrade:**

```bash
helm upgrade supacontrol ./charts/supacontrol \
  -f values.yaml \
  -n supacontrol \
  --wait \
  --timeout 10m

# Watch rollout
kubectl rollout status deployment/supacontrol -n supacontrol
```

**5. Verify Upgrade:**

```bash
# Check pod versions
kubectl get pods -n supacontrol -o wide

# Test API
curl https://supacontrol.example.com/healthz

# Check instances still work
kubectl get namespaces | grep supa-
```

**6. Rollback if Needed:**

```bash
# View history
helm history supacontrol -n supacontrol

# Rollback to previous version
helm rollback supacontrol -n supacontrol

# Or specific revision
helm rollback supacontrol 2 -n supacontrol
```

### Zero-Downtime Upgrades

Ensure these settings for zero-downtime:

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0  # Never have all pods down

readinessProbe:
  # Don't route traffic until ready
  httpGet:
    path: /healthz
    port: 8091
```

---

## Multi-Region Deployment

For global deployments, consider:

**Architecture:**
```
Region 1 (us-west)          Region 2 (eu-west)
├─ SupaControl              ├─ SupaControl
├─ PostgreSQL (primary)     ├─ PostgreSQL (replica)
└─ Instances                └─ Instances
```

**Implementation:**
1. Deploy SupaControl in each region
2. Configure PostgreSQL cross-region replication
3. Use global load balancer (e.g., AWS Global Accelerator)
4. Implement active-passive or active-active setup

---

## Need Help?

- **Deployment Issues**: [Open an issue](https://github.com/qubitquilt/SupaControl/issues)
- **Kubernetes Questions**: [GitHub Discussions](https://github.com/qubitquilt/SupaControl/discussions)
- **Security Concerns**: See [SECURITY.md](SECURITY.md)

---

**Last Updated**: November 2025
