# SupaControl

**SupaControl** is a self-hosted management platform for orchestrating multi-tenant Supabase instances on Kubernetes. It provides a robust API and web dashboard for automated provisioning, monitoring, and lifecycle management of Supabase deployments.

## Overview

SupaControl acts as a control plane that:
- **Automates** Supabase instance deployment using Helm charts
- **Isolates** each instance in its own Kubernetes namespace
- **Manages** instance lifecycle (create, delete, monitor)
- **Provides** secure API access for CLI and programmatic control
- **Offers** a user-friendly web dashboard for visual management

## Features

### Core Features
- 🚀 **Automated Provisioning**: Deploy complete Supabase stacks with a single API call
- 🔒 **Complete Isolation**: Each instance in its own dedicated Kubernetes namespace
- 🔐 **Security First**: API key and JWT authentication, encrypted secrets
- 📊 **Persistent Inventory**: PostgreSQL-backed state management
- 🌐 **Web Dashboard**: Modern React-based UI for instance management
- 🔑 **API Key Management**: Generate and revoke keys for CLI access

### Technical Highlights
- **API-First Design**: REST API with complete functionality
- **Kubernetes-Native**: Built on client-go and Helm v3 SDK
- **Stateless Application**: Horizontally scalable for high availability
- **Declarative Orchestration**: K8s resources managed declaratively
- **Multi-Tenant Ready**: Designed for managing dozens of instances

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      SupaControl                         │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐    ┌──────────────┐                  │
│  │  Web UI      │    │  REST API    │                  │
│  │  (React)     │◄───┤  (Echo/Go)   │                  │
│  └──────────────┘    └──────┬───────┘                  │
│                              │                           │
│                    ┌─────────▼──────────┐               │
│                    │   Orchestrator     │               │
│                    │   (K8s + Helm)     │               │
│                    └─────────┬──────────┘               │
│                              │                           │
│                    ┌─────────▼──────────┐               │
│                    │  Inventory DB      │               │
│                    │  (PostgreSQL)      │               │
│                    └────────────────────┘               │
└─────────────────────────────────────────────────────────┘
                              │
                    ┌─────────▼──────────┐
                    │  Kubernetes API    │
                    └─────────┬──────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   ┌────▼────┐           ┌────▼────┐           ┌────▼────┐
   │Instance1│           │Instance2│           │Instance3│
   │(supa-app1)│         │(supa-app2)│         │(supa-app3)│
   └─────────┘           └─────────┘           └─────────┘
```

## Project Structure

```
/SupaControl
├── /server/              # Go backend
│   ├── /api             # REST API routes and handlers
│   ├── /internal        # Core business logic
│   │   ├── /auth        # Authentication service
│   │   ├── /config      # Configuration management
│   │   ├── /db          # Database layer (sqlx)
│   │   └── /k8s         # K8s orchestration
│   ├── main.go          # Application entry point
│   └── go.mod           # Go dependencies
├── /ui/                 # React frontend
│   ├── /src             # React components
│   ├── package.json     # NPM dependencies
│   └── vite.config.js   # Vite configuration
├── /pkg/                # Shared packages
│   └── /api-types       # API request/response types
├── /charts/             # Helm chart
│   └── /supacontrol     # SupaControl deployment chart
└── README.md            # This file
```

## Prerequisites

- **Kubernetes Cluster** (v1.24+)
- **Helm** (v3.13+)
- **PostgreSQL** (for SupaControl inventory)
- **Go** (v1.21+) for development
- **Node.js** (v18+) for UI development
- **Ingress Controller** (e.g., nginx-ingress)
- **Cert Manager** (optional, for TLS)

## Quick Start

### Option 1: Interactive Installer (Recommended)

The easiest way to install SupaControl is using our interactive CLI installer:

```bash
git clone https://github.com/qubitquilt/SupaControl.git
cd SupaControl/cli
npm install
npm start
```

The installer will:
- ✅ Check prerequisites (kubectl, helm, k8s connection)
- 🔐 Generate secure secrets automatically
- 📝 Guide you through configuration
- 🚀 Deploy to your Kubernetes cluster
- 📋 Provide access information and next steps

**See [cli/README.md](cli/README.md) for detailed installer documentation.**

### Option 2: Manual Installation

If you prefer manual control:

#### 1. Clone the Repository

```bash
git clone https://github.com/qubitquilt/SupaControl.git
cd SupaControl
```

#### 2. Deploy SupaControl

Create a `values.yaml` file:

```yaml
config:
  jwtSecret: "your-super-secret-jwt-key-here"

  database:
    password: "secure-db-password"

  kubernetes:
    ingressClass: "nginx"
    ingressDomain: "supabase.yourdomain.com"

ingress:
  enabled: true
  hosts:
    - host: supacontrol.yourdomain.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: supacontrol-tls
      hosts:
        - supacontrol.yourdomain.com

postgresql:
  auth:
    password: "secure-db-password"
```

Install the Helm chart:

```bash
helm install supacontrol ./charts/supacontrol -f values.yaml
```

### 3. Access the Dashboard

Once deployed, access the web dashboard at `https://supacontrol.yourdomain.com`

**Default credentials:**
- Username: `admin`
- Password: `admin`

⚠️ **IMPORTANT**: Change the default password immediately after first login!

## API Documentation

### Authentication

All API endpoints (except `/healthz` and `/api/v1/auth/login`) require authentication via Bearer token:

```bash
Authorization: Bearer <token_or_api_key>
```

### Endpoints

#### Health Check
```bash
GET /healthz
```

#### Login
```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin"
}
```

#### Get Current User
```bash
GET /api/v1/auth/me
Authorization: Bearer <token>
```

#### Create API Key
```bash
POST /api/v1/auth/api-keys
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Development Key"
}
```

#### List Instances
```bash
GET /api/v1/instances
Authorization: Bearer <api_key>
```

#### Create Instance
```bash
POST /api/v1/instances
Authorization: Bearer <api_key>
Content-Type: application/json

{
  "name": "my-app"
}
```

#### Get Instance
```bash
GET /api/v1/instances/{name}
Authorization: Bearer <api_key>
```

#### Delete Instance
```bash
DELETE /api/v1/instances/{name}
Authorization: Bearer <api_key>
```

## Development

### Backend Development

```bash
cd server

# Install dependencies
go mod download

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=supacontrol
export DB_PASSWORD=password
export DB_NAME=supacontrol
export JWT_SECRET=your-jwt-secret

# Run migrations (if PostgreSQL is running)
# The migrations are auto-applied on startup

# Run the server
go run main.go
```

### Frontend Development

```bash
cd ui

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

### Build Docker Image

```bash
# Build backend
docker build -t supacontrol/server:latest -f server/Dockerfile .

# Build with UI included
cd ui && npm run build && cd ..
docker build -t supacontrol/server:latest .
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `SERVER_HOST` | HTTP server host | `0.0.0.0` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database username | `supacontrol` |
| `DB_PASSWORD` | Database password | **Required** |
| `DB_NAME` | Database name | `supacontrol` |
| `JWT_SECRET` | JWT signing secret | **Required** |
| `KUBECONFIG` | Path to kubeconfig | Empty (in-cluster) |
| `DEFAULT_INGRESS_CLASS` | Ingress class | `nginx` |
| `DEFAULT_INGRESS_DOMAIN` | Base domain for instances | `supabase.example.com` |
| `SUPABASE_CHART_REPO` | Supabase Helm repo | `https://...` |
| `SUPABASE_CHART_NAME` | Chart name | `supabase` |
| `SUPABASE_CHART_VERSION` | Chart version | Latest |

## Security Considerations

### Production Deployment Checklist

- [ ] Change default admin password
- [ ] Use strong, random `JWT_SECRET`
- [ ] Use strong database passwords
- [ ] Enable TLS/HTTPS on all endpoints
- [ ] Configure proper RBAC for ServiceAccount
- [ ] Enable network policies for namespace isolation
- [ ] Regular backup of inventory database
- [ ] Monitor and audit API key usage
- [ ] Implement rate limiting on API endpoints
- [ ] Review and restrict ingress access

### RBAC Permissions

SupaControl requires cluster-wide permissions to:
- Create and delete namespaces
- Manage secrets, configmaps, services
- Deploy workloads (deployments, statefulsets)
- Configure ingresses
- Install Helm releases

The provided Helm chart includes appropriate ClusterRole and ClusterRoleBinding.

## Troubleshooting

### Instance Provisioning Fails

Check the SupaControl logs:
```bash
kubectl logs -n supacontrol deployment/supacontrol -f
```

Check instance status:
```bash
kubectl get all -n supa-<instance-name>
```

### Database Connection Issues

Verify PostgreSQL is running:
```bash
kubectl get pods -n supacontrol -l app.kubernetes.io/name=postgresql
```

Test database connectivity:
```bash
kubectl exec -it deployment/supacontrol -n supacontrol -- /bin/sh
# Inside the pod
psql -h $DB_HOST -U $DB_USER -d $DB_NAME
```

### Authentication Issues

- Verify JWT_SECRET is set correctly
- Check API key is not expired
- Ensure Bearer token format is correct

## Roadmap

- [ ] Instance update/upgrade support
- [ ] Custom resource limits per instance
- [ ] Metrics and monitoring integration (Prometheus)
- [ ] Backup and restore functionality
- [ ] Multi-cluster support
- [ ] Instance templates/presets
- [ ] Webhook notifications
- [ ] Cost tracking per instance

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## License

MIT License - See LICENSE file for details

## Support

- **Issues**: [GitHub Issues](https://github.com/qubitquilt/SupaControl/issues)
- **Documentation**: This README
- **Community**: TBD

## Acknowledgments

- Built on the excellent [supabase-kubernetes](https://github.com/supabase-community/supabase-kubernetes) Helm chart
- Powered by [Echo](https://echo.labstack.com/) web framework
- UI built with [React](https://react.dev/) and [Vite](https://vitejs.dev/)

---

**SupaControl** - Self-hosted Supabase management, simplified.
