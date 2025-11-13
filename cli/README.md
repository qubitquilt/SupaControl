# SupaControl Interactive Installer

This README covers the interactive installer in the `cli/` directory, which deploys the SupaControl server to your Kubernetes cluster. It provides a step-by-step wizard for initial setup.

For command-line management of Supabase instances (e.g., login, create, list, status, delete), use the external full supactl CLI tool available at [https://github.com/qubitquilt/supactl](https://github.com/qubitquilt/supactl).

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

## Full supactl CLI

The interactive installer deploys the SupaControl server. For ongoing management of Supabase instances, install the full supactl CLI tool from the external repository: [https://github.com/qubitquilt/supactl](https://github.com/qubitquilt/supactl).

### Installation

```bash
# Linux/macOS
curl -sSL https://raw.githubusercontent.com/qubitquilt/supactl/main/scripts/install.sh | bash

# Or download from releases
# https://github.com/qubitquilt/supactl/releases
```

### Key Commands

- `supactl login <url>` - Authenticate with your SupaControl server
- `supactl create <name>` - Create a new Supabase instance
- `supactl list` - List all instances
- `supactl status <name>` - Check instance status
- `supactl delete <name>` - Delete an instance

### Features

- 🚀 **Single binary** - No dependencies, works everywhere
- 🔐 **Secure auth** - Credential management built-in
- 📂 **Directory linking** - Associate local dirs with instances
- 🎨 **Interactive UI** - Beautiful prompts and progress indicators
- 🐳 **Local mode** - Manage Docker-based instances without a server

Full documentation is available in the external repository.

## Post-Installation Usage

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

4. **Install and Use Full CLI**:
   ```bash
   # Install supactl
   curl -sSL https://raw.githubusercontent.com/qubitquilt/supactl/main/scripts/install.sh | bash

   # Login to SupaControl
   supactl login https://supacontrol.yourdomain.com

   # Create your first Supabase instance
   supactl create my-first-app

   # Check instance status
   supactl status my-first-app

   # List all instances
   supactl list
   ```

The installer deploys the server, enabling use of the full CLI for instance management.

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

### Helm Repo Addition Required

**Problem**: Supabase Helm chart not found during installation

**Solution**:
```bash
helm repo add supabase-community https://supabase-community.github.io/supabase-kubernetes
helm repo update
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

### Secret Generation Errors

**Problem**: Issues generating JWT secrets or passwords

**Solution**:
- Ensure Node.js version is compatible (v18+)
- Check for write permissions in `~/.supacontrol/`
- Manually generate secrets if needed:
  ```bash
  # Generate JWT secret (64 bytes)
  openssl rand -base64 48

  # Generate DB password (32 chars)
  openssl rand -base64 24 | tr -d "=+/" | cut -c1-32
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
   kubectl port-forward -n supacontrol svc/supacontrol 8091:8091
   # Access at http://localhost:8091
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

### Local Development

```bash
# Development mode (with hot reload)
npm run dev
```

This runs the installer in development mode for testing changes interactively.

### Building

```bash
# Build for distribution
npm run build

# Output: dist/cli.js
```

The build creates a single executable bundle for the installer.

### Testing

```bash
# Run all tests
npm test
```

Tests cover:
- Components (e.g., Welcome.test.tsx, PrerequisitesCheck.test.tsx)
- Utilities (e.g., helm.test.ts, prerequisites.test.ts, secrets.test.ts)
- End-to-end installation flow

Use Vitest for fast, reliable testing with TypeScript support.

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
