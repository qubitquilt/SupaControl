# SupaControl Installer

An interactive CLI installer for SupaControl, built with [Ink](https://github.com/vadimdemedes/ink).

## Features

- 🎯 **Interactive Wizard** - Step-by-step installation guide
- ✅ **Prerequisites Check** - Automatically verifies system requirements
- 🔐 **Secure Secret Generation** - Auto-generates JWT secrets and passwords
- 📝 **Configuration Management** - Guides through all configuration options
- 🚀 **One-Command Install** - Deploys to Kubernetes automatically
- 📊 **Real-time Progress** - Shows installation progress with spinners
- 🎨 **Beautiful UI** - Colorful, modern terminal interface

## Prerequisites

The installer will check for these requirements:

### Required
- **kubectl** - Kubernetes command-line tool
- **helm** (v3+) - Kubernetes package manager
- **Kubernetes cluster** - Accessible and configured

### Optional
- **docker** - For building custom images
- **git** - For cloning the repository

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/qubitquilt/SupaControl.git
cd SupaControl/cli

# Install dependencies
npm install

# Run the installer
npm start
```

### As Global Package

```bash
# Install globally
npm install -g supacontrol-installer

# Run the installer
supacontrol-install
```

### Using npx

```bash
npx supacontrol-installer
```

## Usage

Simply run the installer and follow the prompts:

```bash
npm start
```

### Installation Flow

1. **Welcome Screen** - Introduction and overview
2. **Prerequisites Check** - Verifies system requirements
3. **Configuration Wizard** - Interactive configuration:
   - Kubernetes namespace
   - Helm release name
   - Dashboard hostname
   - Supabase instance domain
   - Ingress configuration
   - TLS/HTTPS settings
   - Database options
   - Secret generation
4. **Installation** - Deploys to Kubernetes
5. **Completion** - Shows access information and next steps

## Configuration Options

### Kubernetes Settings

- **Namespace**: Kubernetes namespace for SupaControl (default: `supacontrol`)
- **Release Name**: Helm release name (default: `supacontrol`)

### Network Settings

- **Dashboard Hostname**: URL for SupaControl dashboard (e.g., `supacontrol.example.com`)
- **Supabase Domain**: Base domain for Supabase instances (e.g., `supabase.example.com`)
- **Ingress Class**: Ingress controller class (default: `nginx`)
- **TLS Enabled**: Enable HTTPS with cert-manager (recommended)

### Database Settings

- **Install Database**: Install PostgreSQL with SupaControl (recommended)
- **External Database**: Use existing PostgreSQL instance

### Security

The installer automatically generates:
- **JWT Secret**: 64-byte secure random string
- **Database Password**: 32-character strong password

## Output

The installer creates:

1. **Helm Values File**: `~/.supacontrol/supacontrol-values.yaml`
   - Contains all configuration
   - Used for future upgrades
   - Keep this file secure!

2. **Kubernetes Resources**:
   - SupaControl deployment
   - PostgreSQL database (if selected)
   - Services and ingresses
   - RBAC configuration

## Post-Installation

After successful installation:

1. **Wait for Pods**:
   ```bash
   kubectl get pods -n supacontrol --watch
   ```

2. **Access Dashboard**:
   - Navigate to your configured hostname
   - Login with default credentials: `admin` / `admin`
   - **Change password immediately!**

3. **Generate API Key**:
   - Go to Settings in the dashboard
   - Create an API key for CLI access

4. **Create First Instance**:
   - Use the dashboard or API
   - Deploy your first Supabase instance

## Troubleshooting

### Prerequisites Check Fails

**Problem**: kubectl or helm not found

**Solution**:
```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

### Kubernetes Connection Failed

**Problem**: Cannot connect to Kubernetes cluster

**Solution**:
```bash
# Check kubectl configuration
kubectl cluster-info

# Set correct context
kubectl config use-context <your-context>

# Verify access
kubectl get nodes
```

### Installation Fails

**Problem**: Helm installation errors

**Solution**:
1. Check Helm logs: `helm list -n supacontrol`
2. View pod logs: `kubectl logs -n supacontrol -l app.kubernetes.io/name=supacontrol`
3. Verify resources: `kubectl get all -n supacontrol`
4. Delete and retry:
   ```bash
   helm uninstall supacontrol -n supacontrol
   kubectl delete namespace supacontrol
   # Run installer again
   ```

### Dashboard Not Accessible

**Problem**: Cannot access dashboard URL

**Solutions**:

1. **Check Ingress**:
   ```bash
   kubectl get ingress -n supacontrol
   ```

2. **Port Forward** (temporary):
   ```bash
   kubectl port-forward -n supacontrol svc/supacontrol 8080:8080
   # Access at http://localhost:8080
   ```

3. **Check TLS Certificate**:
   ```bash
   kubectl get certificate -n supacontrol
   ```

## Development

### Project Structure

```
/cli
├── src/
│   ├── components/      # Ink React components
│   │   ├── Welcome.tsx
│   │   ├── PrerequisitesCheck.tsx
│   │   ├── ConfigurationWizard.tsx
│   │   ├── Installation.tsx
│   │   └── Complete.tsx
│   ├── utils/           # Utility functions
│   │   ├── prerequisites.ts
│   │   ├── secrets.ts
│   │   └── helm.ts
│   └── cli.tsx          # Main entry point
├── package.json
├── tsconfig.json
└── README.md
```

### Building

```bash
# Build for distribution
npm run build

# Output: dist/cli.js
```

### Testing Locally

```bash
# Development mode (with hot reload)
npm run dev

# Test build
npm run build
node dist/cli.js
```

## Technologies

- **[Ink](https://github.com/vadimdemedes/ink)** - React for CLIs
- **[ink-text-input](https://github.com/vadimdemedes/ink-text-input)** - Text input component
- **[ink-select-input](https://github.com/vadimdemedes/ink-select-input)** - Select input component
- **[ink-spinner](https://github.com/vadimdemedes/ink-spinner)** - Loading spinners
- **[execa](https://github.com/sindresorhus/execa)** - Process execution
- **[yaml](https://github.com/eemeli/yaml)** - YAML generation

## Contributing

Contributions welcome! See [CONTRIBUTING.md](../CONTRIBUTING.md).

## License

MIT - See [LICENSE](../LICENSE) file.

---

**SupaControl Installer** - Interactive installation made easy.
