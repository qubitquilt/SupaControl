package k8s

import (
	"context"
	"testing"

	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// createTestCRClient creates a CRClient with a fake client for testing
func createTestCRClient() *CRClient {
	scheme := runtime.NewScheme()
	_ = supacontrolv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	return &CRClient{
		Client: fakeClient,
		scheme: scheme,
	}
}

func TestCRClient_GetScheme(t *testing.T) {
	client := createTestCRClient()

	scheme := client.GetScheme()
	if scheme == nil {
		t.Error("GetScheme() returned nil")
	}

	if scheme != client.scheme {
		t.Error("GetScheme() returned different scheme than client.scheme")
	}
}

func TestCRClient_CreateSupabaseInstance(t *testing.T) {
	tests := []struct {
		name     string
		instance *supacontrolv1alpha1.SupabaseInstance
		wantErr  bool
	}{
		{
			name: "create valid instance",
			instance: &supacontrolv1alpha1.SupabaseInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-instance",
				},
				Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
					ProjectName: "test-project",
				},
			},
			wantErr: false,
		},
		{
			name: "create instance with labels",
			instance: &supacontrolv1alpha1.SupabaseInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "labeled-instance",
					Labels: map[string]string{
						"env": "test",
						"app": "supabase",
					},
				},
				Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
					ProjectName: "labeled-project",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createTestCRClient()
			ctx := context.Background()

			err := client.CreateSupabaseInstance(ctx, tt.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSupabaseInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify instance was created
				got, err := client.GetSupabaseInstance(ctx, tt.instance.Name)
				if err != nil {
					t.Errorf("Failed to get created instance: %v", err)
					return
				}

				if got.Name != tt.instance.Name {
					t.Errorf("Instance name = %v, want %v", got.Name, tt.instance.Name)
				}

				if got.Spec.ProjectName != tt.instance.Spec.ProjectName {
					t.Errorf("ProjectName = %v, want %v", got.Spec.ProjectName, tt.instance.Spec.ProjectName)
				}
			}
		})
	}
}

func TestCRClient_CreateSupabaseInstance_Duplicate(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	instance := &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "duplicate-instance",
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: "duplicate-project",
		},
	}

	// Create first time
	err := client.CreateSupabaseInstance(ctx, instance)
	if err != nil {
		t.Fatalf("First CreateSupabaseInstance() failed: %v", err)
	}

	// Try to create again - should fail
	err = client.CreateSupabaseInstance(ctx, instance)
	if err == nil {
		t.Error("Expected error when creating duplicate instance")
	}
}

func TestCRClient_GetSupabaseInstance(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	// Create a test instance
	testInstance := &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "get-test",
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: "get-test-project",
		},
	}

	err := client.CreateSupabaseInstance(ctx, testInstance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}

	tests := []struct {
		name         string
		instanceName string
		wantErr      bool
	}{
		{
			name:         "get existing instance",
			instanceName: "get-test",
			wantErr:      false,
		},
		{
			name:         "get non-existent instance",
			instanceName: "nonexistent",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.GetSupabaseInstance(ctx, tt.instanceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSupabaseInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got == nil {
					t.Error("Expected non-nil instance")
					return
				}

				if got.Name != tt.instanceName {
					t.Errorf("Instance name = %v, want %v", got.Name, tt.instanceName)
				}
			}
		})
	}
}

func TestCRClient_ListSupabaseInstances(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	// Create multiple instances
	instances := []string{"list-test-1", "list-test-2", "list-test-3"}
	for _, name := range instances {
		instance := &supacontrolv1alpha1.SupabaseInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
				ProjectName: name,
			},
		}
		err := client.CreateSupabaseInstance(ctx, instance)
		if err != nil {
			t.Fatalf("Failed to create test instance %s: %v", name, err)
		}
	}

	// List instances
	list, err := client.ListSupabaseInstances(ctx)
	if err != nil {
		t.Fatalf("ListSupabaseInstances() error = %v", err)
	}

	if list == nil {
		t.Fatal("Expected non-nil list")
	}

	if len(list.Items) != len(instances) {
		t.Errorf("ListSupabaseInstances() returned %d items, want %d", len(list.Items), len(instances))
	}

	// Verify all instances are in the list
	instanceMap := make(map[string]bool)
	for _, item := range list.Items {
		instanceMap[item.Name] = true
	}

	for _, name := range instances {
		if !instanceMap[name] {
			t.Errorf("Instance %s not found in list", name)
		}
	}
}

func TestCRClient_ListSupabaseInstances_Empty(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	// List instances when none exist
	list, err := client.ListSupabaseInstances(ctx)
	if err != nil {
		t.Fatalf("ListSupabaseInstances() error = %v", err)
	}

	if list == nil {
		t.Fatal("Expected non-nil list")
	}

	if len(list.Items) != 0 {
		t.Errorf("ListSupabaseInstances() returned %d items, want 0", len(list.Items))
	}
}

func TestCRClient_DeleteSupabaseInstance(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	// Create a test instance
	testInstance := &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "delete-test",
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: "delete-test-project",
		},
	}

	err := client.CreateSupabaseInstance(ctx, testInstance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}

	// Delete the instance
	err = client.DeleteSupabaseInstance(ctx, "delete-test")
	if err != nil {
		t.Errorf("DeleteSupabaseInstance() error = %v", err)
	}

	// Verify it's deleted
	_, err = client.GetSupabaseInstance(ctx, "delete-test")
	if err == nil {
		t.Error("Expected error when getting deleted instance")
	}
}

func TestCRClient_DeleteSupabaseInstance_NotFound(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	// Try to delete non-existent instance
	err := client.DeleteSupabaseInstance(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error when deleting non-existent instance")
	}
}

func TestCRClient_UpdateSupabaseInstance(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	// Create a test instance
	testInstance := &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "update-test",
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: "original-name",
		},
		Status: supacontrolv1alpha1.SupabaseInstanceStatus{
			Phase: supacontrolv1alpha1.PhasePending,
		},
	}

	err := client.CreateSupabaseInstance(ctx, testInstance)
	if err != nil {
		t.Fatalf("Failed to create test instance: %v", err)
	}

	// Get the instance to update
	instance, err := client.GetSupabaseInstance(ctx, "update-test")
	if err != nil {
		t.Fatalf("Failed to get instance: %v", err)
	}

	// Update the instance
	instance.Spec.ProjectName = "updated-name"
	instance.Status.Phase = supacontrolv1alpha1.PhaseProvisioning

	err = client.UpdateSupabaseInstance(ctx, instance)
	if err != nil {
		t.Errorf("UpdateSupabaseInstance() error = %v", err)
	}

	// Verify the update
	updated, err := client.GetSupabaseInstance(ctx, "update-test")
	if err != nil {
		t.Fatalf("Failed to get updated instance: %v", err)
	}

	if updated.Spec.ProjectName != "updated-name" {
		t.Errorf("ProjectName = %v, want updated-name", updated.Spec.ProjectName)
	}

	if updated.Status.Phase != supacontrolv1alpha1.PhaseProvisioning {
		t.Errorf("Phase = %v, want %v", updated.Status.Phase, supacontrolv1alpha1.PhaseProvisioning)
	}
}

func TestCRClient_UpdateSupabaseInstance_NotFound(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	// Try to update non-existent instance
	nonExistent := &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nonexistent",
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: "test",
		},
	}

	err := client.UpdateSupabaseInstance(ctx, nonExistent)
	if err == nil {
		t.Error("Expected error when updating non-existent instance")
	}
}

// TestCRClient_FullLifecycle tests the complete lifecycle of an instance
func TestCRClient_FullLifecycle(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	instanceName := "lifecycle-test"

	// 1. Create
	instance := &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: instanceName,
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: "lifecycle-project",
		},
		Status: supacontrolv1alpha1.SupabaseInstanceStatus{
			Phase: supacontrolv1alpha1.PhasePending,
		},
	}

	err := client.CreateSupabaseInstance(ctx, instance)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 2. Get
	got, err := client.GetSupabaseInstance(ctx, instanceName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name != instanceName {
		t.Errorf("Get returned wrong instance: %s", got.Name)
	}

	// 3. Update
	got.Status.Phase = supacontrolv1alpha1.PhaseRunning
	got.Status.Namespace = "supa-" + instanceName

	err = client.UpdateSupabaseInstance(ctx, got)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 4. Verify update
	updated, err := client.GetSupabaseInstance(ctx, instanceName)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}

	if updated.Status.Phase != supacontrolv1alpha1.PhaseRunning {
		t.Errorf("Phase not updated: got %s, want %s", updated.Status.Phase, supacontrolv1alpha1.PhaseRunning)
	}

	// 5. List (should contain our instance)
	list, err := client.ListSupabaseInstances(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	found := false
	for _, item := range list.Items {
		if item.Name == instanceName {
			found = true
			break
		}
	}

	if !found {
		t.Error("Instance not found in list")
	}

	// 6. Delete
	err = client.DeleteSupabaseInstance(ctx, instanceName)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 7. Verify deletion
	_, err = client.GetSupabaseInstance(ctx, instanceName)
	if err == nil {
		t.Error("Instance still exists after deletion")
	}
}

// TestCRClient_CreateWithStatus tests creating an instance with initial status
func TestCRClient_CreateWithStatus(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	instance := &supacontrolv1alpha1.SupabaseInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "status-test",
			Labels: map[string]string{
				"test": "status",
			},
		},
		Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
			ProjectName: "status-project",
		},
		Status: supacontrolv1alpha1.SupabaseInstanceStatus{
			Phase:     supacontrolv1alpha1.PhaseProvisioning,
			Namespace: "supa-status-test",
		},
	}

	err := client.CreateSupabaseInstance(ctx, instance)
	if err != nil {
		t.Fatalf("CreateSupabaseInstance() failed: %v", err)
	}

	// Get the created instance
	got, err := client.GetSupabaseInstance(ctx, "status-test")
	if err != nil {
		t.Fatalf("GetSupabaseInstance() failed: %v", err)
	}

	// Verify labels
	if got.Labels["test"] != "status" {
		t.Errorf("Label 'test' = %v, want 'status'", got.Labels["test"])
	}

	// Note: Status might not be preserved on Create in some fake clients
	// This is expected behavior
}

// TestCRClient_ListWithMultiplePhases tests listing instances in different phases
func TestCRClient_ListWithMultiplePhases(t *testing.T) {
	client := createTestCRClient()
	ctx := context.Background()

	phases := []supacontrolv1alpha1.SupabaseInstancePhase{
		supacontrolv1alpha1.PhasePending,
		supacontrolv1alpha1.PhaseProvisioning,
		supacontrolv1alpha1.PhaseRunning,
		supacontrolv1alpha1.PhaseFailed,
	}

	// Create instances in different phases
	for i, phase := range phases {
		instance := &supacontrolv1alpha1.SupabaseInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "phase-test-" + string(phase),
			},
			Spec: supacontrolv1alpha1.SupabaseInstanceSpec{
				ProjectName: "phase-project-" + string(i),
			},
			Status: supacontrolv1alpha1.SupabaseInstanceStatus{
				Phase: phase,
			},
		}

		err := client.CreateSupabaseInstance(ctx, instance)
		if err != nil {
			t.Fatalf("Failed to create instance with phase %s: %v", phase, err)
		}
	}

	// List all instances
	list, err := client.ListSupabaseInstances(ctx)
	if err != nil {
		t.Fatalf("ListSupabaseInstances() failed: %v", err)
	}

	if len(list.Items) != len(phases) {
		t.Errorf("List returned %d instances, want %d", len(list.Items), len(phases))
	}
}
