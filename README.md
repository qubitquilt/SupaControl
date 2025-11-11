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
- [CLI Tool](#cli-tool)
- [API Documentation](#api-documentation)
- [Configuration](#configuration)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)
- [Troubleshooting](#troubleshooting)
- [Security](#security)
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

### Component Architecture

```
┌────────────────────────────────────────────────────────┐
│                  SupaControl Server                     │
├────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐│
│  │ API Layer    │  │ Auth Service │  │  Web Assets  ││
│  │              │  │              │  │              ││
│  │ • Handlers   │  │ • JWT        │  │ • React SPA  ││
│  │ • Routes     │  │ • Passwords  │  │ • Dashboard  ││
│  │ • Middleware │  │ • API Keys   │  │ • Settings   ││
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘│
│         │                 │                            │
│         └─────────┬───────┘                            │
│                   │                                    │
│         ┌─────────▼───────────┐                       │
│         │  Business Logic     │                       │
│         │                     │                       │
│         │  • Validation       │                       │
│         │  • State Management │                       │
│         │  • Error Handling   │                       │
│         └─────────┬───────────┘                       │
│                   │                                    │
│         ┌─────────┼───────────┐                       │
│         │         │           │                       │
│  ┌──────▼──────┐ │ ┌─────────▼────────┐              │
│  │ DB Layer    │ │ │ K8s Orchestrator │              │
│  │             │ │ │                  │              │
│  │ • Instances │ │ │ • Helm Ops       │              │
│  │ • API Keys  │ │ │ • Namespaces     │              │
│  │ • Users     │ │ │ • Status Checks  │              │
│  └─────────────┘ │ └──────────────────┘              │
│                   │                                    │
└───────────────────┼────────────────────────────────────┘
                    │
         ┌──────────┴──────────┐
         │                     │
    PostgreSQL          Kubernetes API
```

### Data Flow

**Instance Creation Flow:**
```
1. User/CLI → POST /api/v1/instances {"name": "myapp"}
2. API validates JWT/API key
3. API validates instance name (DNS-compliant)
4. Check if instance already exists in DB
5. Create namespace: supa-myapp
6. Install Helm chart in namespace
7. Save instance record to PostgreSQL
8. Return instance details to user
```

**Instance Status Check Flow:**
```
1. User/CLI → GET /api/v1/instances/myapp
2. API validates authentication
3. Query database for instance record
4. Query Kubernetes for pod status
5. Aggregate health information
6. Return comprehensive status
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed system design documentation.

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

#### Step 1: Clone Repository

```bash
git clone https://github.com/qubitquilt/SupaControl.git
cd SupaControl
```

#### Step 2: Create values.yaml

Create a `values.yaml` file with your configuration:

```yaml
# values.yaml
config:
  # REQUIRED: Set a secure JWT secret (64+ characters recommended)
  jwtSecret: "your-very-long-and-secure-jwt-secret-here-change-this"

  database:
    host: "supacontrol-postgresql"
    port: "5432"
    user: "supacontrol"
    # REQUIRED: Set a secure database password
    password: "your-secure-db-password-here-change-this"
    name: "supacontrol"

  kubernetes:
    ingressClass: "nginx"
    # Base domain for Supabase instances
    ingressDomain: "supabase.yourdomain.com"

ingress:
  enabled: true
  className: "nginx"
  hosts:
    - host: supacontrol.yourdomain.com
      paths:
        - path: /
          pathType: Prefix

  # Recommended: Enable TLS
  tls:
    - secretName: supacontrol-tls
      hosts:
        - supacontrol.yourdomain.com

postgresql:
  enabled: true
  auth:
    # REQUIRED: Set a secure PostgreSQL password
    # Leave empty for auto-generation
    password: "your-secure-postgres-password-here"
    database: "supacontrol"
    username: "supacontrol"

# Optional: Resource limits
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi

# Optional: Replica count for HA
replicaCount: 1
```

#### Step 3: Install with Helm

```bash
# Create namespace
kubectl create namespace supacontrol

# Install the chart
helm install supacontrol ./charts/supacontrol \
  -f values.yaml \
  -n supacontrol

# Watch deployment progress
kubectl get pods -n supacontrol --watch
```

#### Step 4: Verify Installation

```bash
# Check all resources
kubectl get all -n supacontrol

# Check ingress
kubectl get ingress -n supacontrol

# View logs
kubectl logs -n supacontrol -l app.kubernetes.io/name=supacontrol -f
```

### Option 3: Development Mode

For local development without Kubernetes:

```bash
# 1. Start PostgreSQL with Docker
docker run --name supacontrol-postgres \
  -e POSTGRES_USER=supacontrol \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=supacontrol \
  -p 5432:5432 \
  -d postgres:14

# 2. Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=supacontrol
export DB_PASSWORD=password
export DB_NAME=supacontrol
export JWT_SECRET=your-dev-jwt-secret-at-least-32-chars
export KUBECONFIG=~/.kube/config

# 3. Run backend
cd server
go run main.go

# 4. Run frontend (in another terminal)
cd ui
npm install
npm run dev
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

**Common Tasks:**

1. **Create an Instance:**
   - Click "Create Instance"
   - Enter instance name (e.g., "my-app")
   - Click "Create"
   - Wait for deployment (status will show "Running")

2. **View Instance Details:**
   - Click on instance name
   - View endpoint URLs, status, and creation date

3. **Delete an Instance:**
   - Click "Delete" next to instance
   - Confirm deletion
   - Instance and all resources will be removed

4. **Generate API Key:**
   - Go to Settings
   - Click "Create API Key"
   - Give it a name (e.g., "CI/CD Pipeline")
   - Copy and save the key securely (shown only once!)

### API Usage

All API endpoints require authentication except `/healthz` and `/api/v1/auth/login`.

**Authentication:**
```bash
# Include Bearer token in all requests
Authorization: Bearer <your-jwt-token-or-api-key>
```

**Common Operations:**

```bash
# 1. Login to get JWT token
curl -X POST https://supacontrol.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin"
  }'

# Response: {"token": "eyJhbGc..."}

# 2. Create instance
curl -X POST https://supacontrol.example.com/api/v1/instances \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-app"
  }'

# 3. List all instances
curl -X GET https://supacontrol.example.com/api/v1/instances \
  -H "Authorization: Bearer YOUR_TOKEN"

# 4. Get specific instance
curl -X GET https://supacontrol.example.com/api/v1/instances/my-app \
  -H "Authorization: Bearer YOUR_TOKEN"

# 5. Delete instance
curl -X DELETE https://supacontrol.example.com/api/v1/instances/my-app \
  -H "Authorization: Bearer YOUR_TOKEN"

# 6. Create API key
curl -X POST https://supacontrol.example.com/api/v1/auth/api-keys \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production API Key"
  }'
```

See [API Documentation](#api-documentation) for complete endpoint reference.

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

## API Documentation

### Base URL

```
https://supacontrol.yourdomain.com/api/v1
```

### Authentication

All endpoints except `/healthz` and `/api/v1/auth/login` require Bearer token authentication:

```http
Authorization: Bearer <jwt-token-or-api-key>
```

### Endpoints

#### Health Check

```http
GET /healthz
```

**Response:**
```json
{
  "status": "ok"
}
```

#### Authentication

##### Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

##### Get Current User

```http
GET /api/v1/auth/me
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": 1,
  "username": "admin"
}
```

#### API Keys

##### Create API Key

```http
POST /api/v1/auth/api-keys
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Production Key"
}
```

**Response:**
```json
{
  "id": 1,
  "name": "Production Key",
  "key": "sk_live_abc123...",
  "created_at": "2024-01-15T10:30:00Z"
}
```

⚠️ **Important:** The key is shown only once. Save it securely!

##### List API Keys

```http
GET /api/v1/auth/api-keys
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "id": 1,
    "name": "Production Key",
    "created_at": "2024-01-15T10:30:00Z",
    "revoked_at": null
  }
]
```

##### Revoke API Key

```http
DELETE /api/v1/auth/api-keys/:id
Authorization: Bearer <token>
```

#### Instances

##### List Instances

```http
GET /api/v1/instances
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "id": 1,
    "name": "my-app",
    "namespace": "supa-my-app",
    "status": "Running",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:05:00Z"
  }
]
```

##### Create Instance

```http
POST /api/v1/instances
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "my-app"
}
```

**Requirements:**
- Name must be lowercase alphanumeric with hyphens
- Maximum 63 characters
- Must be unique

**Response:**
```json
{
  "id": 1,
  "name": "my-app",
  "namespace": "supa-my-app",
  "status": "Pending",
  "created_at": "2024-01-15T10:00:00Z"
}
```

##### Get Instance

```http
GET /api/v1/instances/:name
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": 1,
  "name": "my-app",
  "namespace": "supa-my-app",
  "status": "Running",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:05:00Z"
}
```

##### Delete Instance

```http
DELETE /api/v1/instances/:name
Authorization: Bearer <token>
```

**Response:**
```json
{
  "message": "Instance deleted successfully"
}
```

This will:
1. Uninstall the Helm release
2. Delete the Kubernetes namespace
3. Soft delete the database record

### Error Responses

All errors follow this format:

```json
{
  "message": "Error description"
}
```

**Common HTTP Status Codes:**
- `200` - Success
- `201` - Created
- `400` - Bad Request (validation error)
- `401` - Unauthorized (invalid/missing token)
- `404` - Not Found
- `500` - Internal Server Error

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SERVER_PORT` | HTTP server port | `8091` | No |
| `SERVER_HOST` | HTTP server host | `0.0.0.0` | No |
| `DB_HOST` | PostgreSQL host | `localhost` | Yes |
| `DB_PORT` | PostgreSQL port | `5432` | Yes |
| `DB_USER` | Database username | `supacontrol` | Yes |
| `DB_PASSWORD` | Database password | - | **Yes** |
| `DB_NAME` | Database name | `supacontrol` | Yes |
| `JWT_SECRET` | JWT signing secret | - | **Yes** |
| `KUBECONFIG` | Path to kubeconfig file | Empty (in-cluster) | No |
| `DEFAULT_INGRESS_CLASS` | Ingress class for instances | `nginx` | No |
| `DEFAULT_INGRESS_DOMAIN` | Base domain for instances | `supabase.example.com` | No |
| `SUPABASE_CHART_REPO` | Supabase Helm chart repo | `https://supabase-community.github.io/supabase-kubernetes` | No |
| `SUPABASE_CHART_NAME` | Supabase chart name | `supabase` | No |
| `SUPABASE_CHART_VERSION` | Chart version | Latest | No |

### Helm Chart Values

Key configuration options in `values.yaml`:

```yaml
# Number of replicas (for HA)
replicaCount: 1

# Docker image configuration
image:
  repository: supacontrol/server
  pullPolicy: IfNotPresent
  tag: "latest"

# Resource limits
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi

# Application configuration
config:
  jwtSecret: ""          # REQUIRED
  database:
    host: ""
    port: "5432"
    user: ""
    password: ""         # REQUIRED
    name: ""
  kubernetes:
    ingressClass: "nginx"
    ingressDomain: ""

# Ingress configuration
ingress:
  enabled: true
  className: "nginx"
  hosts: []
  tls: []

# PostgreSQL subchart
postgresql:
  enabled: true
  auth:
    database: "supacontrol"
    username: "supacontrol"
    password: ""         # REQUIRED

# Service Account (RBAC)
serviceAccount:
  create: true
  annotations: {}
  name: ""
```

Full chart values: `charts/supacontrol/values.yaml`

## Development

### Prerequisites for Development

- **Go** 1.21+
- **Node.js** 18+
- **PostgreSQL** 14+
- **Docker** (optional, for containerized PostgreSQL)
- **Kubernetes cluster** (for integration testing)
- **kubectl** and **helm** (for Kubernetes testing)

### Project Structure

```
/SupaControl
├── /server/              # Go backend
│   ├── /api             # REST API (handlers, routes, middleware)
│   ├── /internal        # Core business logic
│   │   ├── /auth        # JWT & password hashing
│   │   ├── /config      # Configuration management
│   │   ├── /db          # Database layer (sqlx + migrations)
│   │   └── /k8s         # Kubernetes orchestration (Helm + client-go)
│   ├── main.go          # Application entry point
│   └── go.mod           # Go dependencies
│
├── /ui/                 # React frontend
│   ├── /src
│   │   ├── /pages       # Page components (Dashboard, Login, Settings)
│   │   ├── api.js       # API client
│   │   ├── App.jsx      # Main app component
│   │   └── main.jsx     # Entry point
│   ├── package.json     # NPM dependencies
│   └── vite.config.js   # Vite build configuration
│
├── /cli/                # Interactive installer
│   ├── /src
│   │   ├── /components  # Ink React components
│   │   └── /utils       # Helper functions
│   └── package.json
│
├── /pkg/                # Shared packages
│   └── /api-types       # API request/response types
│
├── /charts/             # Helm charts
│   └── /supacontrol     # SupaControl deployment chart
│
├── /.github/            # GitHub Actions CI/CD
│   └── /workflows
│       └── ci.yml       # CI pipeline
│
├── README.md            # This file
├── CONTRIBUTING.md      # Contribution guidelines
├── TESTING.md           # Testing documentation
├── CLAUDE.md            # AI assistant guide
└── LICENSE.md           # MIT License
```

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
export JWT_SECRET=your-dev-jwt-secret-change-this

# Option 1: Use Docker for PostgreSQL
docker run --name supacontrol-postgres \
  -e POSTGRES_USER=supacontrol \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=supacontrol \
  -p 5432:5432 \
  -d postgres:14

# Option 2: Use local PostgreSQL
createdb supacontrol

# Run the server (migrations auto-apply on startup)
go run main.go

# Server runs on http://localhost:8091
```

**Backend Code Guidelines:**
- Follow [Effective Go](https://golang.org/doc/effective_go.html) conventions
- Use `gofmt` for formatting
- Run `go vet` before committing
- Add tests for new features
- Handle errors explicitly
- Add comments for exported functions

### Frontend Development

```bash
cd ui

# Install dependencies
npm install

# Start development server with hot reload
npm run dev

# Development server runs on http://localhost:5173
# API calls are proxied to backend (see vite.config.js)
```

**Frontend Code Guidelines:**
- Use functional components with hooks
- Follow React best practices
- Use meaningful component/variable names
- Keep components focused and reusable
- Add PropTypes or TypeScript types
- Test user interactions

### Building

```bash
# Build backend binary
cd server
go build -o supacontrol

# Build frontend for production
cd ui
npm run build
# Output: ui/dist/

# Build Docker image (with UI embedded)
cd ui && npm run build && cd ..
docker build -t supacontrol/server:latest .

# Build Helm chart
helm package ./charts/supacontrol
# Output: supacontrol-<version>.tgz
```

### Code Style

**Go:**
```bash
# Format code
gofmt -w .

# Lint code
go vet ./...

# Run with golangci-lint (recommended)
golangci-lint run
```

**JavaScript/React:**
```bash
cd ui

# Lint code
npm run lint

# Fix auto-fixable issues
npm run lint -- --fix
```

## Testing

SupaControl uses comprehensive testing across all components.

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run CI checks (tests + lints + build)
make ci
```

### Backend Tests

```bash
cd server

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Run with race detection
go test -race ./...

# Run specific package
go test ./internal/auth

# Run specific test
go test -run TestHashPassword ./internal/auth
```

### Frontend Tests

```bash
cd ui

# Run tests
npm test

# Run with coverage
npm run test:coverage

# Run in watch mode
npm test -- --watch

# Run with UI
npm run test:ui

# View coverage report
open coverage/index.html
```

### Test Coverage

**Current Status (November 2024):**
- Backend: 6.3% coverage (goal: 70%+)
- Frontend: 5.87% coverage (goal: 70%+)
- Critical paths need improvement

**Priority areas for testing:**
1. API handlers
2. Database operations
3. Kubernetes orchestration
4. React components
5. Authentication flows

See [TESTING.md](TESTING.md) for comprehensive testing documentation.

### Writing Tests

**Backend (Table-driven tests):**
```go
func TestValidateInstanceName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid name", "my-app", false},
        {"empty name", "", true},
        {"too long", strings.Repeat("a", 100), true},
        {"uppercase", "MyApp", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateInstanceName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Frontend (Component tests):**
```javascript
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Dashboard from './Dashboard';

test('creates instance on form submit', async () => {
    const user = userEvent.setup();
    render(<Dashboard />);

    await user.type(screen.getByLabelText('Instance Name'), 'test-app');
    await user.click(screen.getByText('Create'));

    await waitFor(() => {
        expect(screen.getByText('test-app')).toBeInTheDocument();
    });
});
```

## Deployment

### Production Deployment Checklist

Before deploying to production:

- [ ] **Security:**
  - [ ] Change default admin password
  - [ ] Generate strong JWT secret (64+ chars)
  - [ ] Use strong database passwords
  - [ ] Enable TLS/HTTPS on all endpoints
  - [ ] Review and restrict RBAC permissions
  - [ ] Enable network policies for namespace isolation

- [ ] **Reliability:**
  - [ ] Configure resource limits and requests
  - [ ] Set up pod autoscaling (HPA)
  - [ ] Configure multiple replicas for HA
  - [ ] Set up liveness and readiness probes
  - [ ] Configure persistent volumes for database

- [ ] **Observability:**
  - [ ] Configure logging aggregation
  - [ ] Set up metrics collection (Prometheus)
  - [ ] Configure alerting
  - [ ] Enable audit logging

- [ ] **Backup:**
  - [ ] Schedule database backups
  - [ ] Test backup restoration
  - [ ] Document disaster recovery procedures

- [ ] **Monitoring:**
  - [ ] Monitor API response times
  - [ ] Track instance creation/deletion rates
  - [ ] Monitor Kubernetes resource usage
  - [ ] Set up health check alerts

### High Availability Setup

```yaml
# values.yaml for HA deployment
replicaCount: 3

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80

postgresql:
  enabled: true
  architecture: replication
  replication:
    enabled: true
    numSynchronousReplicas: 1
  persistence:
    enabled: true
    size: 20Gi
```

### Kubernetes RBAC

SupaControl requires cluster-wide permissions:

**Required Permissions:**
- `namespaces`: create, delete, get, list
- `secrets`: create, delete, get, list, update
- `configmaps`: create, delete, get, list, update
- `services`: create, delete, get, list, update
- `deployments`: create, delete, get, list, update
- `statefulsets`: create, delete, get, list, update
- `ingresses`: create, delete, get, list, update
- `persistentvolumeclaims`: create, delete, get, list
- `pods`: get, list (for status checks)

The Helm chart includes appropriate `ClusterRole` and `ClusterRoleBinding`.

### Monitoring with Prometheus

```yaml
# values.yaml - Enable Prometheus metrics
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
```

**Metrics exposed:**
- `supacontrol_instances_total` - Total instances managed
- `supacontrol_api_requests_total` - API request count
- `supacontrol_api_request_duration_seconds` - API latency
- `supacontrol_instance_creation_duration_seconds` - Instance creation time

## Troubleshooting

### Common Issues

#### 1. Installation Fails

**Symptom:** Helm install fails or pods crash

**Diagnosis:**
```bash
# Check Helm release status
helm list -n supacontrol

# Check pod status
kubectl get pods -n supacontrol

# View pod logs
kubectl logs -n supacontrol -l app.kubernetes.io/name=supacontrol

# Describe pod for events
kubectl describe pod -n supacontrol <pod-name>
```

**Common Causes:**
- Missing required values (JWT_SECRET, DB_PASSWORD)
- Insufficient RBAC permissions
- Image pull failures
- Resource constraints

**Solutions:**
```bash
# Delete and reinstall with correct values
helm uninstall supacontrol -n supacontrol
helm install supacontrol ./charts/supacontrol -f values.yaml -n supacontrol

# Check resource availability
kubectl top nodes
kubectl describe node <node-name>
```

#### 2. Database Connection Failures

**Symptom:** Server logs show "connection refused" or "authentication failed"

**Diagnosis:**
```bash
# Check PostgreSQL pod
kubectl get pods -n supacontrol -l app.kubernetes.io/name=postgresql

# View PostgreSQL logs
kubectl logs -n supacontrol -l app.kubernetes.io/name=postgresql

# Test connection from SupaControl pod
kubectl exec -it -n supacontrol deployment/supacontrol -- /bin/sh
# Inside pod:
psql -h $DB_HOST -U $DB_USER -d $DB_NAME
```

**Solutions:**
- Verify database credentials in values.yaml
- Check PostgreSQL pod is running
- Verify service name matches DB_HOST
- Check network policies aren't blocking traffic

#### 3. Instance Creation Fails

**Symptom:** Instance stuck in "Pending" or creation errors

**Diagnosis:**
```bash
# Check SupaControl logs
kubectl logs -n supacontrol -l app.kubernetes.io/name=supacontrol -f

# Check if namespace was created
kubectl get namespace supa-<instance-name>

# Check Helm release status
helm list -n supa-<instance-name>

# Check pod status in instance namespace
kubectl get pods -n supa-<instance-name>
```

**Common Causes:**
- Insufficient cluster resources
- Helm chart repository unreachable
- RBAC permission issues
- Ingress misconfiguration

**Solutions:**
```bash
# Manually check Helm repository
helm repo list
helm repo update

# Test Helm chart download
helm pull supabase/supabase --version <version>

# Check RBAC permissions
kubectl auth can-i create namespaces --as=system:serviceaccount:supacontrol:supacontrol

# View detailed error from logs
kubectl logs -n supacontrol -l app.kubernetes.io/name=supacontrol --tail=100
```

#### 4. Dashboard Not Accessible

**Symptom:** Cannot access SupaControl dashboard URL

**Diagnosis:**
```bash
# Check ingress configuration
kubectl get ingress -n supacontrol
kubectl describe ingress -n supacontrol supacontrol

# Check ingress controller
kubectl get pods -n ingress-nginx

# Check service
kubectl get svc -n supacontrol
```

**Solutions:**

**Option 1: Port Forward (Temporary)**
```bash
kubectl port-forward -n supacontrol svc/supacontrol 8091:8091
# Access at http://localhost:8091
```

**Option 2: Fix Ingress**
```bash
# Verify DNS points to ingress controller
nslookup supacontrol.yourdomain.com

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx

# Verify ingress class
kubectl get ingressclass
```

**Option 3: Fix TLS Certificate**
```bash
# Check certificate status
kubectl get certificate -n supacontrol

# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager

# Describe certificate for events
kubectl describe certificate -n supacontrol supacontrol-tls
```

#### 5. Authentication Issues

**Symptom:** "Unauthorized" errors or login fails

**Diagnosis:**
```bash
# Check if JWT_SECRET is set correctly
kubectl get secret -n supacontrol supacontrol -o jsonpath='{.data.JWT_SECRET}' | base64 -d

# Test login endpoint
curl -X POST https://supacontrol.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' \
  -v
```

**Solutions:**
- Verify JWT_SECRET is set and consistent
- Check password hasn't been changed from default
- Clear browser cache/cookies
- Verify API key hasn't been revoked

#### 6. Helm Release Conflicts

**Symptom:** "release already exists" errors

**Solutions:**
```bash
# List existing releases
helm list -A

# Delete existing release
helm uninstall supacontrol -n supacontrol

# Clean up resources
kubectl delete namespace supacontrol

# Reinstall
helm install supacontrol ./charts/supacontrol -f values.yaml -n supacontrol
```

### Debug Mode

Enable debug logging:

```yaml
# values.yaml
env:
  - name: LOG_LEVEL
    value: "debug"
```

### Getting Help

If you're still stuck:

1. **Check logs** with maximum verbosity:
   ```bash
   kubectl logs -n supacontrol -l app.kubernetes.io/name=supacontrol --all-containers=true --tail=500
   ```

2. **Gather diagnostic info**:
   ```bash
   kubectl get all -n supacontrol -o yaml > diagnostics.yaml
   helm get values supacontrol -n supacontrol > current-values.yaml
   ```

3. **Open an issue** with:
   - SupaControl version
   - Kubernetes version (`kubectl version`)
   - Error messages and logs
   - Steps to reproduce
   - Diagnostic files (redact secrets!)

4. **Check existing issues**:
   [github.com/qubitquilt/SupaControl/issues](https://github.com/qubitquilt/SupaControl/issues)

## Security

### Security Best Practices

#### Secrets Management

**DO:**
- ✅ Use strong, randomly generated secrets
- ✅ Store secrets in Kubernetes Secrets
- ✅ Rotate secrets regularly
- ✅ Use separate secrets for dev/staging/prod
- ✅ Limit access to secrets using RBAC

**DON'T:**
- ❌ Commit secrets to git
- ❌ Use default passwords in production
- ❌ Share secrets via insecure channels
- ❌ Reuse secrets across environments

#### Network Security

```yaml
# Enable network policies for instance isolation
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
```

#### TLS/HTTPS

**Always use TLS in production:**
```yaml
ingress:
  enabled: true
  tls:
    - secretName: supacontrol-tls
      hosts:
        - supacontrol.yourdomain.com
```

**Use cert-manager for automatic certificates:**
```yaml
ingress:
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
```

#### API Security

- All endpoints require authentication (except health check and login)
- JWT tokens expire after 24 hours
- API keys can be revoked at any time
- Rate limiting recommended (use ingress annotations)

#### RBAC

Review and minimize ServiceAccount permissions:

```bash
# View current permissions
kubectl describe clusterrole supacontrol

# Audit access
kubectl auth can-i --list --as=system:serviceaccount:supacontrol:supacontrol
```

### Security Updates

- Monitor [GitHub Security Advisories](https://github.com/qubitquilt/SupaControl/security/advisories)
- Keep dependencies updated: `go get -u ./...` and `npm update`
- Subscribe to Kubernetes security announcements
- Regularly review audit logs

### Reporting Security Issues

**DO NOT** open public issues for security vulnerabilities.

Instead, email: security@qubitquilt.io (if available) or open a [private security advisory](https://github.com/qubitquilt/SupaControl/security/advisories/new).

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

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

### Quick Contribution Guide

1. **Fork** the repository
2. **Clone** your fork
3. **Create** a feature branch: `git checkout -b feature/amazing-feature`
4. **Make** your changes
5. **Test** your changes: `make test`
6. **Commit** with clear messages: `git commit -m 'feat: add amazing feature'`
7. **Push** to your fork: `git push origin feature/amazing-feature`
8. **Open** a Pull Request

### Contribution Areas

We especially welcome contributions in:

- 🧪 **Testing** - Improve test coverage (currently ~6%)
- 📝 **Documentation** - Tutorials, guides, examples
- 🐛 **Bug Fixes** - Fix reported issues
- ✨ **Features** - Implement roadmap items
- 🎨 **UI/UX** - Improve dashboard design
- 🔒 **Security** - Security audits and improvements

### Code of Conduct

- Be respectful and inclusive
- Accept constructive criticism gracefully
- Focus on what's best for the community
- Show empathy towards others

## License

MIT License - See [LICENSE.md](LICENSE.md) for details.

Copyright (c) 2024 SupaControl Contributors

## Support

### Documentation

- **README** (this file) - Overview and getting started
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- **[TESTING.md](TESTING.md)** - Testing documentation
- **[CLAUDE.md](CLAUDE.md)** - AI assistant development guide
- **[cli/README.md](cli/README.md)** - CLI installer documentation

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

[Get Started](#quick-start) · [Documentation](#table-of-contents) · [Report Bug](https://github.com/qubitquilt/SupaControl/issues) · [Request Feature](https://github.com/qubitquilt/SupaControl/issues/new)

Made with ❤️ by the SupaControl community

</div>
