// Package k8s provides Kubernetes client functionality for SupaControl.
package k8s

import (
	"context"

	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CRClient wraps controller-runtime client for SupabaseInstance operations
type CRClient struct {
	client.Client
	scheme *runtime.Scheme
}

// NewCRClient creates a new CR client
func NewCRClient(config *rest.Config) (*CRClient, error) {
	scheme := runtime.NewScheme()
	if err := supacontrolv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return &CRClient{
		Client: c,
		scheme: scheme,
	}, nil
}

// GetScheme returns the runtime scheme
func (c *CRClient) GetScheme() *runtime.Scheme {
	return c.scheme
}

// CreateSupabaseInstance creates a new SupabaseInstance CR
func (c *CRClient) CreateSupabaseInstance(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	return c.Create(ctx, instance)
}

// GetSupabaseInstance gets a SupabaseInstance CR by name
func (c *CRClient) GetSupabaseInstance(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
	instance := &supacontrolv1alpha1.SupabaseInstance{}
	if err := c.Get(ctx, client.ObjectKey{Name: name}, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

// ListSupabaseInstances lists all SupabaseInstance CRs
func (c *CRClient) ListSupabaseInstances(ctx context.Context) (*supacontrolv1alpha1.SupabaseInstanceList, error) {
	list := &supacontrolv1alpha1.SupabaseInstanceList{}
	if err := c.List(ctx, list); err != nil {
		return nil, err
	}
	return list, nil
}

// DeleteSupabaseInstance deletes a SupabaseInstance CR
func (c *CRClient) DeleteSupabaseInstance(ctx context.Context, name string) error {
	instance := &supacontrolv1alpha1.SupabaseInstance{}
	instance.Name = name
	return c.Delete(ctx, instance)
}

// UpdateSupabaseInstance updates a SupabaseInstance CR
func (c *CRClient) UpdateSupabaseInstance(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	return c.Update(ctx, instance)
}
