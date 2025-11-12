package api

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
	"github.com/qubitquilt/supacontrol/server/internal/db"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// mockDBClient is a mock implementation of DBClient for testing
type mockDBClient struct {
	getUserByUsernameFunc    func(username string) (*db.User, error)
	getUserByIDFunc          func(id int64) (*db.User, error)
	createAPIKeyFunc         func(userID int64, name, keyHash string, expiresAt *time.Time) (*apitypes.APIKey, error)
	listAPIKeysByUserFunc    func(userID int64) ([]*apitypes.APIKey, error)
	listAllAPIKeysFunc       func() ([]*apitypes.APIKey, error)
	getAPIKeyByIDFunc        func(id int64) (*apitypes.APIKey, error)
	deleteAPIKeyFunc         func(id int64) error
	getAPIKeyByHashFunc      func(keyHash string) (*apitypes.APIKey, error)
	updateAPIKeyLastUsedFunc func(id int64) error
}

func (m *mockDBClient) GetUserByUsername(username string) (*db.User, error) {
	if m.getUserByUsernameFunc != nil {
		return m.getUserByUsernameFunc(username)
	}
	return nil, fmt.Errorf("GetUserByUsername not implemented")
}

func (m *mockDBClient) GetUserByID(id int64) (*db.User, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(id)
	}
	return nil, fmt.Errorf("GetUserByID not implemented")
}

func (m *mockDBClient) CreateAPIKey(userID int64, name, keyHash string, expiresAt *time.Time) (*apitypes.APIKey, error) {
	if m.createAPIKeyFunc != nil {
		return m.createAPIKeyFunc(userID, name, keyHash, expiresAt)
	}
	return nil, fmt.Errorf("CreateAPIKey not implemented")
}

func (m *mockDBClient) ListAPIKeysByUser(userID int64) ([]*apitypes.APIKey, error) {
	if m.listAPIKeysByUserFunc != nil {
		return m.listAPIKeysByUserFunc(userID)
	}
	return nil, fmt.Errorf("ListAPIKeysByUser not implemented")
}

func (m *mockDBClient) ListAllAPIKeys() ([]*apitypes.APIKey, error) {
	if m.listAllAPIKeysFunc != nil {
		return m.listAllAPIKeysFunc()
	}
	return nil, fmt.Errorf("ListAllAPIKeys not implemented")
}

func (m *mockDBClient) GetAPIKeyByID(id int64) (*apitypes.APIKey, error) {
	if m.getAPIKeyByIDFunc != nil {
		return m.getAPIKeyByIDFunc(id)
	}
	return nil, fmt.Errorf("GetAPIKeyByID not implemented")
}

func (m *mockDBClient) DeleteAPIKey(id int64) error {
	if m.deleteAPIKeyFunc != nil {
		return m.deleteAPIKeyFunc(id)
	}
	return fmt.Errorf("DeleteAPIKey not implemented")
}

func (m *mockDBClient) GetAPIKeyByHash(keyHash string) (*apitypes.APIKey, error) {
	if m.getAPIKeyByHashFunc != nil {
		return m.getAPIKeyByHashFunc(keyHash)
	}
	return nil, fmt.Errorf("GetAPIKeyByHash not implemented")
}

func (m *mockDBClient) UpdateAPIKeyLastUsed(id int64) error {
	if m.updateAPIKeyLastUsedFunc != nil {
		return m.updateAPIKeyLastUsedFunc(id)
	}
	return fmt.Errorf("UpdateAPIKeyLastUsed not implemented")
}

// mockCRClient is a mock implementation of CRClient for testing
type mockCRClient struct {
	createSupabaseInstanceFunc func(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error
	getSupabaseInstanceFunc    func(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error)
	listSupabaseInstancesFunc  func(ctx context.Context) (*supacontrolv1alpha1.SupabaseInstanceList, error)
	updateSupabaseInstanceFunc func(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error
	deleteSupabaseInstanceFunc func(ctx context.Context, name string) error
}

func (m *mockCRClient) CreateSupabaseInstance(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	if m.createSupabaseInstanceFunc != nil {
		return m.createSupabaseInstanceFunc(ctx, instance)
	}
	return fmt.Errorf("CreateSupabaseInstance not implemented")
}

func (m *mockCRClient) GetSupabaseInstance(ctx context.Context, name string) (*supacontrolv1alpha1.SupabaseInstance, error) {
	if m.getSupabaseInstanceFunc != nil {
		return m.getSupabaseInstanceFunc(ctx, name)
	}
	return nil, fmt.Errorf("GetSupabaseInstance not implemented")
}

func (m *mockCRClient) ListSupabaseInstances(ctx context.Context) (*supacontrolv1alpha1.SupabaseInstanceList, error) {
	if m.listSupabaseInstancesFunc != nil {
		return m.listSupabaseInstancesFunc(ctx)
	}
	return nil, fmt.Errorf("ListSupabaseInstances not implemented")
}

func (m *mockCRClient) UpdateSupabaseInstance(ctx context.Context, instance *supacontrolv1alpha1.SupabaseInstance) error {
	if m.updateSupabaseInstanceFunc != nil {
		return m.updateSupabaseInstanceFunc(ctx, instance)
	}
	return fmt.Errorf("UpdateSupabaseInstance not implemented")
}

func (m *mockCRClient) DeleteSupabaseInstance(ctx context.Context, name string) error {
	if m.deleteSupabaseInstanceFunc != nil {
		return m.deleteSupabaseInstanceFunc(ctx, name)
	}
	return fmt.Errorf("DeleteSupabaseInstance not implemented")
}

// mockK8sClient is a mock implementation of the K8sClient interface for testing
type mockK8sClient struct {
	clientset kubernetes.Interface
}

func (m *mockK8sClient) GetClientset() kubernetes.Interface {
	if m.clientset != nil {
		return m.clientset
	}
	return &fake.Clientset{}
}

// newTestContext creates a test echo context with the given method, path, and body
func newTestContext(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// setAuthContext sets the authentication context for a test request
func setAuthContext(c echo.Context, userID int64, username, role string) {
	c.Set("auth", &AuthContext{
		UserID:   userID,
		Username: username,
		Role:     role,
		IsAPIKey: false,
	})
}
