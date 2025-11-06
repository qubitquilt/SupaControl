package k8s

import (
	"context"
	"fmt"
	"log"

	apitypes "github.com/qubitquilt/supacontrol/pkg/api-types"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
)

// Orchestrator handles the lifecycle of Supabase instances
type Orchestrator struct {
	k8sClient            *Client
	chartRepo            string
	chartName            string
	chartVersion         string
	defaultIngressClass  string
	defaultIngressDomain string
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(k8sClient *Client, chartRepo, chartName, chartVersion, ingressClass, ingressDomain string) *Orchestrator {
	return &Orchestrator{
		k8sClient:            k8sClient,
		chartRepo:            chartRepo,
		chartName:            chartName,
		chartVersion:         chartVersion,
		defaultIngressClass:  ingressClass,
		defaultIngressDomain: ingressDomain,
	}
}

// CreateInstance provisions a new Supabase instance
func (o *Orchestrator) CreateInstance(ctx context.Context, projectName string) (*apitypes.Instance, error) {
	log.Printf("Starting provisioning of instance: %s", projectName)

	// Generate namespace name
	namespace := fmt.Sprintf("supa-%s", projectName)

	// Create namespace
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "supacontrol",
		"supacontrol.io/instance":      projectName,
	}

	if err := o.k8sClient.CreateNamespace(ctx, namespace, labels); err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	log.Printf("Created namespace: %s", namespace)

	// Generate secrets
	postgresPassword, err := GenerateSecurePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate postgres password: %w", err)
	}

	jwtSecret, err := GenerateJWTSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
	}

	anonKey, err := GenerateJWTSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate anon key: %w", err)
	}

	serviceRoleKey, err := GenerateJWTSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate service role key: %w", err)
	}

	// Create secrets in Kubernetes
	secretData := map[string][]byte{
		"postgres-password":  []byte(postgresPassword),
		"jwt-secret":         []byte(jwtSecret),
		"anon-key":           []byte(anonKey),
		"service-role-key":   []byte(serviceRoleKey),
	}

	if err := o.k8sClient.CreateSecret(ctx, namespace, fmt.Sprintf("%s-secrets", projectName), secretData, labels); err != nil {
		return nil, fmt.Errorf("failed to create secrets: %w", err)
	}

	log.Printf("Created secrets for instance: %s", projectName)

	// Install Helm chart
	releaseName := projectName
	if err := o.installHelmChart(namespace, releaseName, postgresPassword, jwtSecret, anonKey, serviceRoleKey); err != nil {
		return nil, fmt.Errorf("failed to install helm chart: %w", err)
	}

	log.Printf("Installed Helm chart for instance: %s", projectName)

	// Create ingress
	studioHost := fmt.Sprintf("%s-studio.%s", projectName, o.defaultIngressDomain)
	apiHost := fmt.Sprintf("%s-api.%s", projectName, o.defaultIngressDomain)

	// Note: These service names are based on the supabase-kubernetes chart conventions
	// You may need to adjust them based on the actual chart
	if err := o.k8sClient.CreateIngress(ctx, namespace, fmt.Sprintf("%s-studio-ingress", projectName), studioHost, fmt.Sprintf("%s-studio", releaseName), 3000, o.defaultIngressClass); err != nil {
		log.Printf("Warning: failed to create studio ingress: %v", err)
	}

	if err := o.k8sClient.CreateIngress(ctx, namespace, fmt.Sprintf("%s-api-ingress", projectName), apiHost, fmt.Sprintf("%s-kong", releaseName), 8000, o.defaultIngressClass); err != nil {
		log.Printf("Warning: failed to create API ingress: %v", err)
	}

	log.Printf("Created ingresses for instance: %s", projectName)

	// Create instance record
	instance := &apitypes.Instance{
		ProjectName: projectName,
		Namespace:   namespace,
		Status:      apitypes.StatusRunning,
		StudioURL:   fmt.Sprintf("https://%s", studioHost),
		APIURL:      fmt.Sprintf("https://%s", apiHost),
	}

	log.Printf("Successfully provisioned instance: %s", projectName)

	return instance, nil
}

// DeleteInstance removes a Supabase instance
func (o *Orchestrator) DeleteInstance(ctx context.Context, projectName, namespace string) error {
	log.Printf("Starting deletion of instance: %s", projectName)

	// Uninstall Helm release
	if err := o.uninstallHelmChart(namespace, projectName); err != nil {
		log.Printf("Warning: failed to uninstall helm chart: %v", err)
	}

	log.Printf("Uninstalled Helm chart for instance: %s", projectName)

	// Delete namespace (this will delete all resources including secrets and ingresses)
	if err := o.k8sClient.DeleteNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	log.Printf("Deleted namespace: %s", namespace)
	log.Printf("Successfully deleted instance: %s", projectName)

	return nil
}

// installHelmChart installs the Supabase Helm chart
func (o *Orchestrator) installHelmChart(namespace, releaseName, postgresPassword, jwtSecret, anonKey, serviceRoleKey string) error {
	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", log.Printf); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	client := action.NewInstall(actionConfig)
	client.Namespace = namespace
	client.ReleaseName = releaseName
	client.CreateNamespace = false // We already created it
	client.Wait = false // Don't wait for resources to be ready (async)
	client.Timeout = 0

	// Set chart version if specified
	if o.chartVersion != "" {
		client.Version = o.chartVersion
	}

	// Values for the chart
	values := map[string]interface{}{
		"postgresql": map[string]interface{}{
			"auth": map[string]interface{}{
				"postgresPassword": postgresPassword,
			},
		},
		"jwt": map[string]interface{}{
			"secret":         jwtSecret,
			"anonKey":        anonKey,
			"serviceRoleKey": serviceRoleKey,
		},
	}

	// Locate the chart
	// Note: This assumes the chart is available locally or in a repo
	// In production, you'd want to add the repo and fetch the chart
	chartPath := o.chartName
	if o.chartRepo != "" {
		// For remote charts, you'd typically use: repo/chartName
		// This requires the repo to be added first using: helm repo add
		chartPath = fmt.Sprintf("%s/%s", o.chartRepo, o.chartName)
	}

	cp, err := client.ChartPathOptions.LocateChart(chartPath, settings)
	if err != nil {
		return fmt.Errorf("failed to locate chart: %w", err)
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	_, err = client.Run(chartRequested, values)
	if err != nil {
		return fmt.Errorf("failed to install chart: %w", err)
	}

	return nil
}

// uninstallHelmChart uninstalls the Helm release
func (o *Orchestrator) uninstallHelmChart(namespace, releaseName string) error {
	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", log.Printf); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	client := action.NewUninstall(actionConfig)
	client.Wait = false
	client.Timeout = 0

	_, err := client.Run(releaseName)
	if err != nil {
		return fmt.Errorf("failed to uninstall chart: %w", err)
	}

	return nil
}

// GetRelease gets information about a Helm release
func (o *Orchestrator) GetRelease(namespace, releaseName string) (*release.Release, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", log.Printf); err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	client := action.NewGet(actionConfig)
	rel, err := client.Run(releaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	return rel, nil
}
