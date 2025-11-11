# SupaControl

[![GitHub Actions](https://img.shields.io/github/actions/workflow/status/qubitquilt/SupaControl/ci.yml?style=for-the-badge&logo=githubactions&logoColor=white)](https://github.com/qubitquilt/SupaControl/actions)
[![Codecov](https://img.shields.io/codecov/c/github/qubitquilt/SupaControl?style=for-the-badge&logo=codecov&logoColor=white)](https://codecov.io/gh/qubitquilt/SupaControl)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen?style=for-the-badge&logo=go&logoColor=white)](https://goreportcard.com/report/github.com/qubitquilt/supacontrol/server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)

## Built With

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![React](https://img.shields.io/badge/react-%2320232a.svg?style=for-the-badge&logo=react&logoColor=%2361DAFB)
![Vite](https://img.shields.io/badge/vite-%23646CFF.svg?style=for-the-badge&logo=vite&logoColor=white)
![NodeJS](https://img.shields.io/badge/node.js-6DA55F?style=for-the-badge&logo=node.js&logoColor=white)
![NPM](https://img.shields.io/badge/NPM-%23CB3837.svg?style=for-the-badge&logo=npm&logoColor=white)
![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Helm](https://img.shields.io/badge/Helm-0F1689?style=for-the-badge&logo=Helm&labelColor=0F1689)
![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![GitHub Actions](https://img.shields.io/badge/github%20actions-%232671E5.svg?style=for-the-badge&logo=githubactions&logoColor=white)

---

**SupaControl** is a self-hosted management platform for orchestrating multi-tenant Supabase instances on Kubernetes. It provides a robust API and web dashboard for automated provisioning, monitoring, and lifecycle management of Supabase deployments.

Perfect for:
- 🏢 **SaaS Providers** offering Supabase as a managed service
- 👨‍💻 **Development Teams** managing multiple environments
- 🎓 **Educational Institutions** providing isolated instances for students
- 🏗️ **Platform Engineers** building internal developer platforms

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Documentation](#documentation)
- [CLI Tool](#cli-tool)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

## Overview

SupaControl acts as a **control plane** that sits between you and your Kubernetes cluster, simplifying the complexity of managing multiple Supabase instances.

### What It Does

- 🚀 **Automates** Supabase instance deployment using Helm charts
- 🔒 **Isolates** each instance in its own Kubernetes namespace
- 📊 **Manages** complete instance lifecycle (create, monitor, delete)
- 🔐 **Secures** API access with JWT authentication and API keys
- 🌐 **Provides** a modern web dashboard for visual management
- 🛠️ **Integrates** with CI/CD pipelines via REST API

### Why SupaControl?

**Without SupaControl:**
```bash
# Manual Supabase deployment
helm repo add supabase https://...
kubectl create namespace supa-myapp
helm install myapp supabase/supabase -n supa-myapp -f custom-values.yaml
kubectl apply -f ingress.yaml -n supa-myapp
# Repeat for each instance... 😓
```

**With SupaControl:**
```bash
# One command deployment
supactl create myapp

# Or via API
curl -X POST https://supacontrol.example.com/api/v1/instances \
  -H "Authorization: Bearer $API_KEY" \
  -d '{"name": "myapp"}'
```

## Features

### Core Features

- 🚀 **Automated Provisioning**: Deploy complete Supabase stacks with a single API call
- 🔒 **Complete Isolation**: Each instance in its own dedicated Kubernetes namespace
- 🔐 **Security First**: API key and JWT authentication, encrypted secrets management
- 📊 **Persistent Inventory**: PostgreSQL-backed state tracking and audit logs
- 🌐 **Web Dashboard**: Modern React-based UI for instance management
- 🔑 **API Key Management**: Generate, list, and revoke keys for CLI/programmatic access
- 🎯 **Status Monitoring**: Real-time instance health and deployment status
- 🗑️ **Clean Deletion**: Automated cleanup of namespaces and resources

### Technical Highlights

- **API-First Design**: Complete functionality exposed via REST API
- **Kubernetes-Native**: Built on client-go and Helm v3 SDK
- **Stateless Application**: Horizontally scalable for high availability
- **Declarative Orchestration**: Kubernetes resources managed declaratively
- **Multi-Tenant Ready**: Designed for managing dozens to hundreds of instances
- **Production-Ready**: Includes health checks, logging, and error handling
- **CI/CD Friendly**: Integrate with automated deployment pipelines

### Use Cases

**SaaS Multi-Tenancy**
```
Customer A → Instance: supa-customer-a (dedicated database, isolated)
Customer B → Instance: supa-customer-b (dedicated database, isolated)
Customer C → Instance: supa-customer-c (dedicated database, isolated)
```

**Development Environments**
```
Production  → Instance: supa-prod
Staging     → Instance: supa-staging
Development → Instance: supa-dev
Testing     → Instance: supa-test
```

**Educational Institutions**
```
Student 1 → Instance: supa-student1 (isolated learning environment)
Student 2 → Instance: supa-student2 (isolated learning environment)
...
```

## Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────┐
│                      SupaControl                         │
│                   (Control Plane)                        │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐    ┌──────────────┐                  │
│  │  Web UI      │    │  REST API    │                  │
│  │  (React)     │◄───┤  (Echo/Go)   │◄─── API Clients │
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
   │Namespace│           │Namespace│           │Namespace│
   │supa-app1│           │supa-app2│           │supa-app3│
   │         │           │         │           │         │
   │ Supabase│           │ Supabase│           │ Supabase│
   │  Stack  │           │  Stack  │           │  Stack  │
   └─────────┘           └─────────┘           └─────────┘
```

For detailed architecture documentation, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Quick Start

The fastest way to get started with SupaControl is using our interactive installer.

### Prerequisites Checklist

Before you begin, ensure you have:

- ✅ **Kubernetes Cluster** (v1.24+) running and accessible
- ✅ **kubectl** installed and configured
- ✅ **Helm** (v3.13+) installed
- ✅ **Ingress Controller** (e.g., nginx-ingress) deployed in your cluster
- ⚠️ **DNS/Domain** configured for accessing the dashboard

**Quick Prerequisites Check:**
```bash
# Verify kubectl
kubectl version --client

# Verify Helm
helm version

# Verify cluster access
kubectl cluster-info

# Verify ingress controller
kubectl get ingressclass
```

### 5-Minute Install

```bash
# 1. Clone the repository
git clone https://github.com/qubitquilt/SupaControl.git
cd SupaControl/cli

# 2. Install dependencies
npm install

# 3. Run interactive installer
npm start
```

The installer will:
1. ✅ Check all prerequisites
2. 🔐 Generate secure secrets automatically
3. 📝 Guide you through configuration (domain, ingress, etc.)
4. 🚀 Deploy to your Kubernetes cluster
5. 📋 Provide access information and next steps

**After installation completes:**
```bash
# 1. Wait for pods to be ready (1-2 minutes)
kubectl get pods -n supacontrol --watch

# 2. Access the dashboard
# Navigate to https://supacontrol.yourdomain.com

# 3. Login with default credentials
# Username: admin
# Password: admin
# ⚠️ CHANGE THIS IMMEDIATELY!
```

### First Steps After Install

```bash
# 1. Access dashboard and change default password
# Go to Settings → Change Password

# 2. Generate an API key for CLI access
# Go to Settings → API Keys → Create New Key

# 3. Install the CLI tool
curl -sSL https://raw.githubusercontent.com/qubitquilt/supactl/main/scripts/install.sh | bash

# 4. Login to SupaControl
supactl login https://supacontrol.yourdomain.com

# 5. Create your first Supabase instance
supactl create my-first-app

# 6. Check instance status
supactl status my-first-app

# 7. List all instances
supactl list
```

## Installation

### Option 1: Interactive Installer (Recommended)

The easiest way to install SupaControl. See [Quick Start](#quick-start) above.

**Full CLI installer documentation:** [cli/README.md](cli/README.md)

### Option 2: Manual Helm Installation

For advanced users who want full control over the installation.

```bash
# 1. Clone repository
git clone https://github.com/qubitquilt/SupaControl.git
cd SupaControl

# 2. Create values.yaml
cat > values.yaml <<EOF
config:
  jwtSecret: "your-very-long-and-secure-jwt-secret-here-change-this"
  database:
    host: "supacontrol-postgresql"
    port: "5432"
    user: "supacontrol"
    password: "your-secure-db-password-here-change-this"
    name: "supacontrol"
  kubernetes:
    ingressClass: "nginx"
    ingressDomain: "supabase.yourdomain.com"

ingress:
  enabled: true
  className: "nginx"
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
  enabled: true
  auth:
    password: "your-secure-postgres-password-here"
    database: "supacontrol"
    username: "supacontrol"
EOF

# 3. Install with Helm
kubectl create namespace supacontrol
helm install supacontrol ./charts/supacontrol \
  -f values.yaml \
  -n supacontrol

# 4. Watch deployment
kubectl get pods -n supacontrol --watch
```

### Option 3: Development Mode

For local development without Kubernetes:

```bash
# 1. Start PostgreSQL
docker run --name supacontrol-postgres \
  -e POSTGRES_USER=supacontrol \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=supacontrol \
  -p 5432:5432 \
  -d postgres:14

# 2. Set environment variables
export DB_HOST=localhost DB_PORT=5432
export DB_USER=supacontrol DB_PASSWORD=password DB_NAME=supacontrol
export JWT_SECRET=your-dev-jwt-secret-at-least-32-chars

# 3. Run backend
cd server && go run main.go

# 4. Run frontend (in another terminal)
cd ui && npm install && npm run dev
```

## Usage

### Web Dashboard

Access the dashboard at `https://supacontrol.yourdomain.com`

**Dashboard Features:**
- 📊 **Instance Overview** - List all Supabase instances with status
- ➕ **Create Instances** - Deploy new instances with one click
- 🗑️ **Delete Instances** - Remove instances and clean up resources
- 🔑 **API Key Management** - Generate and revoke API keys
- 👤 **User Settings** - Change password and manage account

### API Usage

All API endpoints require authentication except `/healthz` and `/api/v1/auth/login`.

**Quick Example:**
```bash
# 1. Login
TOKEN=$(curl -X POST https://supacontrol.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

# 2. Create instance
curl -X POST https://supacontrol.example.com/api/v1/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-app"}'

# 3. List instances
curl -X GET https://supacontrol.example.com/api/v1/instances \
  -H "Authorization: Bearer $TOKEN"
```

**Complete API documentation:** [docs/API.md](docs/API.md)

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SERVER_PORT` | HTTP server port | `8091` | No |
| `DB_HOST` | PostgreSQL host | `localhost` | Yes |
| `DB_PORT` | PostgreSQL port | `5432` | Yes |
| `DB_USER` | Database username | `supacontrol` | Yes |
| `DB_PASSWORD` | Database password | - | **Yes** |
| `DB_NAME` | Database name | `supacontrol` | Yes |
| `JWT_SECRET` | JWT signing secret | - | **Yes** |
| `KUBECONFIG` | Path to kubeconfig | Empty (in-cluster) | No |
| `DEFAULT_INGRESS_CLASS` | Ingress class | `nginx` | No |
| `DEFAULT_INGRESS_DOMAIN` | Base domain for instances | `supabase.example.com` | No |

> **Note for Developers**: The `KUBECONFIG` environment variable is crucial for local Kubernetes development. See the [Development Guide](docs/DEVELOPMENT.md#kubernetes-configuration-for-local-development) for detailed setup instructions and troubleshooting.

For complete configuration options, see the [Helm chart values](charts/supacontrol/values.yaml).

## Documentation

Comprehensive documentation is available in the `/docs` directory:

### Getting Started
- 📘 **[README.md](README.md)** (this file) - Overview and quick start
- 🚀 **[cli/README.md](cli/README.md)** - CLI installer documentation

### Technical Documentation
- 🏗️ **[ARCHITECTURE.md](ARCHITECTURE.md)** - System design and architecture
- 📡 **[docs/API.md](docs/API.md)** - Complete API reference
- 🚀 **[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)** - Production deployment guide
- 💻 **[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)** - Development setup and guidelines
- 🔒 **[docs/SECURITY.md](docs/SECURITY.md)** - Security best practices
- 🔧 **[docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)** - Common issues and solutions

### Contributing
- 🤝 **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- 🧪 **[TESTING.md](TESTING.md)** - Testing documentation
- 🤖 **[CLAUDE.md](CLAUDE.md)** - AI assistant development guide

## CLI Tool

For command-line management, use **[supactl](https://github.com/qubitquilt/supactl)** - our official CLI tool.

### Installation

```bash
# Linux/macOS
curl -sSL https://raw.githubusercontent.com/qubitquilt/supactl/main/scripts/install.sh | bash

# Or download from releases
# https://github.com/qubitquilt/supactl/releases
```

### Quick Start

```bash
# Login to SupaControl server
supactl login https://supacontrol.yourdomain.com

# Create instance
supactl create my-project

# List instances
supactl list

# Check instance status
supactl status my-project

# Delete instance
supactl delete my-project
```

### Features

- 🚀 **Single binary** - No dependencies, works everywhere
- 🔐 **Secure auth** - Credential management built-in
- 📂 **Directory linking** - Associate local dirs with instances
- 🎨 **Interactive UI** - Beautiful prompts and progress indicators
- 🐳 **Local mode** - Manage Docker-based instances without a server

**Full CLI documentation:** [github.com/qubitquilt/supactl](https://github.com/qubitquilt/supactl)

## Roadmap

### Planned Features

#### v0.2.0
- [ ] Instance update/upgrade support
- [ ] Custom resource limits per instance
- [ ] Instance status webhooks
- [ ] Backup and restore functionality

#### v0.3.0
- [ ] Prometheus metrics integration
- [ ] Grafana dashboards
- [ ] Instance templates/presets
- [ ] Multi-cluster support

#### v0.4.0
- [ ] Cost tracking per instance
- [ ] Resource quota management
- [ ] Advanced RBAC (user roles)
- [ ] Audit logging

#### Future
- [ ] GitOps integration (ArgoCD/Flux)
- [ ] Instance cloning
- [ ] Automated scaling policies
- [ ] Multi-tenancy improvements
- [ ] Custom domain per instance
- [ ] Instance migration between clusters

### Contributing to Roadmap

Have ideas? We'd love to hear them!

- 💡 [Open a feature request](https://github.com/qubitquilt/SupaControl/issues/new?template=feature_request.md)
- 🗳️ Vote on existing feature requests
- 💬 Join discussions on GitHub

## Contributing

We welcome contributions! Whether you're fixing bugs, improving documentation, or proposing new features, we appreciate your help.

### Quick Start

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Test: `make test`
5. Commit: `git commit -m 'feat: add amazing feature'`
6. Push: `git push origin feature/amazing-feature`
7. Open a Pull Request

### Ways to Contribute

- 🧪 **Testing** - Improve test coverage (currently ~6%)
- 📝 **Documentation** - Tutorials, guides, examples
- 🐛 **Bug Fixes** - Fix reported issues
- ✨ **Features** - Implement roadmap items
- 🎨 **UI/UX** - Improve dashboard design
- 🔒 **Security** - Security audits and improvements

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## License

MIT License - See [LICENSE.md](LICENSE.md) for details.

Copyright (c) 2024 SupaControl Contributors

## Support

### Documentation

- **README** (this file) - Overview and getting started
- **[Complete Documentation](docs/)** - All technical documentation
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- **[TESTING.md](TESTING.md)** - Testing documentation

### Community

- 💬 **Discussions** - [GitHub Discussions](https://github.com/qubitquilt/SupaControl/discussions)
- 🐛 **Bug Reports** - [GitHub Issues](https://github.com/qubitquilt/SupaControl/issues)
- 💡 **Feature Requests** - [GitHub Issues](https://github.com/qubitquilt/SupaControl/issues/new?template=feature_request.md)

### Related Projects

- **[supactl](https://github.com/qubitquilt/supactl)** - Official CLI tool for SupaControl
- **[supabase-kubernetes](https://github.com/supabase-community/supabase-kubernetes)** - Community Helm chart for Supabase

## Acknowledgments

SupaControl is built on excellent open-source projects:

- **[Supabase](https://supabase.com/)** - The open source Firebase alternative
- **[supabase-kubernetes](https://github.com/supabase-community/supabase-kubernetes)** - Community Helm chart
- **[Kubernetes](https://kubernetes.io/)** - Container orchestration platform
- **[Helm](https://helm.sh/)** - The package manager for Kubernetes
- **[Echo](https://echo.labstack.com/)** - High performance Go web framework
- **[React](https://react.dev/)** - JavaScript library for user interfaces
- **[Vite](https://vitejs.dev/)** - Next generation frontend tooling
- **[client-go](https://github.com/kubernetes/client-go)** - Go client for Kubernetes
- **[sqlx](https://github.com/jmoiron/sqlx)** - Extensions to Go's database/sql

Special thanks to all [contributors](https://github.com/qubitquilt/SupaControl/graphs/contributors)!

---

<div align="center">

**SupaControl** - Self-hosted Supabase management, simplified.

[Get Started](#quick-start) · [Documentation](docs/) · [Report Bug](https://github.com/qubitquilt/SupaControl/issues) · [Request Feature](https://github.com/qubitquilt/SupaControl/issues/new)

Made with ❤️ by the SupaControl community

</div>
