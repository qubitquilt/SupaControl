# ADR 001: SupabaseInstance CRD as Single Source of Truth for Instance State

**Status**: Accepted

**Date**: 2025-11-11

**Decision Makers**: Engineering Team

**Technical Story**: Resolving the "dual source of truth" architectural ambiguity between PostgreSQL `instances` table and Kubernetes Custom Resource Definitions (CRDs).

---

## Context and Problem Statement

SupaControl manages Supabase instances on Kubernetes. During the initial development, there was ambiguity about where instance state should be stored:

1. **PostgreSQL Database** (`instances` table) - Traditional relational database approach
2. **Kubernetes CRDs** (`SupabaseInstance` custom resources) - Cloud-native Kubernetes approach

This ambiguity resulted in:
- Documentation (ARCHITECTURE.md, CLAUDE.md) describing PostgreSQL as the "Inventory DB" for instance state
- A database schema with an `instances` table
- A `server/internal/db/instances.go` repository implementation (174 lines)
- **BUT** - the actual API handlers (`server/api/handlers.go`) using only the Kubernetes CRD client (`crClient`) and ignoring the database

This created technical debt:
- Dead code in the codebase (`instances.go`)
- Orphaned database schema (`instances` table)
- Misleading documentation
- Architectural confusion for new contributors
- Risk of future bugs if developers try to "fix" the code to match the docs

## Decision Drivers

### Technical Considerations

1. **Kubernetes-Native Architecture**: SupaControl is fundamentally a Kubernetes controller/operator pattern
2. **Declarative State Management**: CRDs provide declarative, version-controlled, and auditable state
3. **Built-in K8s Features**: CRDs give us metadata, status conditions, finalizers, and controller-runtime reconciliation for free
4. **Source of Truth Proximity**: Instance state should live where the instances actually run
5. **Observability**: `kubectl get supabaseinstances` provides immediate visibility without querying a separate database

### Operational Considerations

1. **Reduced Complexity**: One less system to synchronize, backup, and monitor
2. **Consistency Guarantees**: K8s API server provides strong consistency, optimistic concurrency control, and watch notifications
3. **Failure Scenarios**: No risk of database-K8s state divergence
4. **GitOps Ready**: CRDs can be managed via ArgoCD/Flux for declarative deployments

### Implementation Reality

The code already implements this pattern correctly:
- `CreateInstance()` creates a `SupabaseInstance` CR (line 239-254 in handlers.go)
- `ListInstances()` queries CRDs via `crClient.ListSupabaseInstances()` (line 269)
- `GetInstance()` retrieves a CR via `crClient.GetSupabaseInstance()` (line 292)
- `DeleteInstance()` deletes a CR (line 322)
- The `instances.go` database file has **zero references** in the API handlers

## Decision Outcome

**Chosen Option**: **SupabaseInstance CRD is the Single Source of Truth for all instance state**

### What This Means

1. **Instance State** (project name, namespace, status, URLs, error messages, timestamps):
   - **STORED IN**: `SupabaseInstance` Custom Resource (`.status` field)
   - **NOT STORED IN**: PostgreSQL `instances` table

2. **PostgreSQL Usage** (remains):
   - **Users** (`users` table): Authentication, user accounts
   - **API Keys** (`api_keys` table): API key management, revocation
   - **(Future) Audit Logs**: Compliance and change tracking

3. **State Flow**:
   ```
   API Request → Create/Update SupabaseInstance CR → Controller Reconciles → Update CR Status
                                                                                      ↓
   API Response ← Convert CR to API Response ← Read CR Status ← K8s API Server
   ```

### Consequences

#### Positive

- **Eliminated Dead Code**: Remove 174 lines of unused database code
- **Simplified Architecture**: One source of truth, not two
- **Better Observability**: `kubectl get supabaseinstances -A` shows all instances
- **Controller-Native**: Aligns with Kubernetes operator best practices
- **GitOps Compatible**: Instances can be declared in YAML and synced via ArgoCD/Flux
- **Automatic Cleanup**: K8s garbage collection handles orphaned resources
- **Versioned State**: CRDs support API versioning (`v1alpha1`, `v1beta1`, `v1`)

#### Negative

- **K8s Dependency**: Cannot query instance state without K8s API access
  - *Mitigation*: This is acceptable - SupaControl IS a K8s control plane
- **Query Limitations**: K8s API is not a database (no complex SQL queries)
  - *Mitigation*: Client-side filtering is sufficient for current scale
  - *Future*: If analytics needed, build read-only projection into PostgreSQL
- **Backup Strategy**: CRDs must be included in K8s cluster backups
  - *Mitigation*: Use Velero or cluster backup solutions

#### Neutral

- **Migration Path**: Existing deployments have no data in `instances` table, so no migration needed
- **Monitoring**: Need to monitor K8s API availability (already required)

## Implementation

### Immediate Actions (Workstream II)

1. ✅ **Create this ADR** - Formalize the decision
2. 🔄 **Delete Dead Code**: Remove `server/internal/db/instances.go`
3. 🔄 **Drop Database Table**: Create migration `003_remove_instances_table.sql`
4. 🔄 **Update Documentation**:
   - ARCHITECTURE.md: Remove "Inventory DB" references
   - CLAUDE.md: Clarify PostgreSQL is for users/API keys only
   - Add this ADR to documentation

### Code Changes Required

```go
// server/internal/db/instances.go
// DELETE THIS ENTIRE FILE (174 lines)
```

```sql
-- server/internal/db/migrations/003_remove_instances_table.sql
-- DROP TABLE instances CASCADE;
-- (New migration to be created)
```

### Documentation Updates

Update ARCHITECTURE.md section "Data Architecture":
- Remove instances table from ERD
- Update "State Management" to clarify CRD as source of truth
- Update data flow diagrams

Update CLAUDE.md:
- Remove references to `server/internal/db/instances.go`
- Clarify PostgreSQL usage (users, API keys, future audit logs)

## Compliance and Standards

This decision aligns with:

- **Kubernetes Operator Best Practices**: Use CRDs for application state
- **12-Factor App**: Stateless application, declarative configuration
- **Cloud Native Principles**: Treat infrastructure as declarative resources

## Alternatives Considered

### Alternative 1: PostgreSQL as Primary Source of Truth

**Approach**: Store instance state in PostgreSQL, sync to K8s

**Pros**:
- Familiar relational database model
- Complex SQL queries possible
- Easier to generate reports

**Cons**:
- **State Synchronization Problem**: Database and K8s can diverge
- **Double Writes**: Every state change requires DB + K8s update
- **Failure Modes**: What if DB write succeeds but K8s write fails?
- **Not Kubernetes-Native**: Fights against the platform
- **Extra Operational Burden**: Backup, monitor, scale PostgreSQL for instance state

**Why Rejected**: Introduces unnecessary complexity and synchronization issues

### Alternative 2: Dual Storage (Both PostgreSQL and CRDs)

**Approach**: Store in both, use one as "primary" and other as cache

**Pros**:
- Flexibility to query either system

**Cons**:
- **Worst of Both Worlds**: All the complexity of synchronization
- **Consistency Nightmares**: Which is the source of truth when they conflict?
- **Operational Overhead**: Monitor and maintain both systems
- **Code Complexity**: Double the CRUD code

**Why Rejected**: Violates "single source of truth" principle, unmaintainable

### Alternative 3: External Database (Not PostgreSQL)

**Approach**: Use a separate database just for instance metadata (e.g., etcd, MongoDB)

**Pros**:
- Could optimize for specific access patterns

**Cons**:
- **Another System**: More operational burden
- **Still Not Kubernetes-Native**: Same sync issues as PostgreSQL
- **Redundant**: K8s already runs etcd for CRD storage

**Why Rejected**: Adds complexity without solving the core problem

## Validation and Testing

This decision is validated by:

1. **Current Implementation**: Code already works this way successfully
2. **Test Coverage**: Existing tests (to be expanded in Workstream I) use CRD client mocks
3. **Kubernetes Best Practices**: Aligns with operator pattern documentation
4. **Community Examples**: Similar operators (e.g., Crossplane, ArgoCD) use CRDs as source of truth

## References

- [Kubernetes Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [Kubebuilder Book - Controller Runtime Client](https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html)
- [Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- Technical Brief: "Evolving SupaControl to a Production-Grade Control Plane" (2025-11-11)

## Review and Approval

- **Proposed By**: Engineering Team
- **Reviewed By**: Technical Lead
- **Approved By**: Engineering Team
- **Date**: 2025-11-11

---

## Changelog

- **2025-11-11**: Initial ADR created, decision accepted
