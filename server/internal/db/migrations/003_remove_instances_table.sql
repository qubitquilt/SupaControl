-- Migration: Remove instances table (ADR-001)
--
-- Context: ADR-001 establishes that SupabaseInstance CRDs are the Single Source of Truth
-- for instance state, not PostgreSQL. This migration removes the orphaned instances table
-- and its associated database objects.
--
-- Date: 2025-11-11
-- Reference: docs/adr/001-crd-as-single-source-of-truth.md

-- Drop the trigger for updating updated_at on instances table
DROP TRIGGER IF EXISTS update_instances_updated_at ON instances;

-- Drop indexes
DROP INDEX IF EXISTS idx_instances_status;
DROP INDEX IF EXISTS idx_instances_project_name;

-- Drop the instances table
-- Note: This table was never used by the application (handlers use CRDs via crClient)
DROP TABLE IF EXISTS instances;

-- Note: We keep the update_updated_at_column() function as it's still used by the users table
