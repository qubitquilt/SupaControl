# API Documentation

Complete API reference for SupaControl REST API.

## Table of Contents

- [Overview](#overview)
- [Base URL](#base-url)
- [Authentication](#authentication)
- [Endpoints](#endpoints)
  - [Health Check](#health-check)
  - [Authentication](#authentication-endpoints)
  - [API Keys](#api-keys)
  - [Instances](#instances)
- [Error Responses](#error-responses)

## Overview

SupaControl provides a RESTful API for managing Supabase instances programmatically. All endpoints (except `/healthz` and login) require authentication via Bearer token.

## Base URL

```
https://supacontrol.yourdomain.com/api/v1
```

## Authentication

All protected endpoints require Bearer token authentication:

```http
Authorization: Bearer <jwt-token-or-api-key>
```

**Two authentication methods:**
1. **JWT Token** - Short-lived (24 hours), obtained via login
2. **API Key** - Long-lived, revocable, generated via dashboard/API

## Endpoints

### Health Check

Check if the SupaControl server is running.

```http
GET /healthz
```

**Response:**
```json
{
  "status": "ok"
}
```

**Status Codes:**
- `200 OK` - Server is healthy

---

### Authentication Endpoints

#### Login

Authenticate with username and password to receive a JWT token.

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

**Status Codes:**
- `200 OK` - Login successful
- `400 Bad Request` - Missing credentials
- `401 Unauthorized` - Invalid credentials

**Example:**
```bash
curl -X POST https://supacontrol.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin"
  }'
```

#### Get Current User

Get information about the currently authenticated user.

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

**Status Codes:**
- `200 OK` - Success
- `401 Unauthorized` - Invalid or missing token

**Example:**
```bash
curl -X GET https://supacontrol.example.com/api/v1/auth/me \
  -H "Authorization: Bearer $TOKEN"
```

---

### API Keys

Manage API keys for programmatic access.

#### Create API Key

Generate a new API key for CLI or automation use.

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
  "created_at": "2025-01-15T10:30:00Z"
}
```

**Status Codes:**
- `201 Created` - API key created successfully
- `400 Bad Request` - Invalid request body
- `401 Unauthorized` - Invalid or missing token

⚠️ **Important:** The API key is shown only once. Save it securely!

**Example:**
```bash
curl -X POST https://supacontrol.example.com/api/v1/auth/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI/CD Pipeline Key"
  }'
```

#### List API Keys

Get all API keys for the current user.

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
    "created_at": "2025-01-15T10:30:00Z",
    "revoked_at": null
  },
  {
    "id": 2,
    "name": "Development Key",
    "created_at": "2025-01-14T08:00:00Z",
    "revoked_at": "2025-01-15T12:00:00Z"
  }
]
```

**Status Codes:**
- `200 OK` - Success
- `401 Unauthorized` - Invalid or missing token

**Example:**
```bash
curl -X GET https://supacontrol.example.com/api/v1/auth/api-keys \
  -H "Authorization: Bearer $TOKEN"
```

#### Revoke API Key

Revoke an existing API key to prevent further use.

```http
DELETE /api/v1/auth/api-keys/:id
Authorization: Bearer <token>
```

**Response:**
```json
{
  "message": "API key revoked successfully"
}
```

**Status Codes:**
- `200 OK` - API key revoked
- `401 Unauthorized` - Invalid or missing token
- `404 Not Found` - API key not found

**Example:**
```bash
curl -X DELETE https://supacontrol.example.com/api/v1/auth/api-keys/1 \
  -H "Authorization: Bearer $TOKEN"
```

---

### Instances

Manage Supabase instances.

#### List Instances

Get all Supabase instances.

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
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T10:05:00Z"
  },
  {
    "id": 2,
    "name": "staging-app",
    "namespace": "supa-staging-app",
    "status": "Pending",
    "created_at": "2025-01-15T11:00:00Z",
    "updated_at": "2025-01-15T11:00:00Z"
  }
]
```

**Status Codes:**
- `200 OK` - Success
- `401 Unauthorized` - Invalid or missing token

**Example:**
```bash
curl -X GET https://supacontrol.example.com/api/v1/instances \
  -H "Authorization: Bearer $TOKEN"
```

#### Create Instance

Deploy a new Supabase instance.

```http
POST /api/v1/instances
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "my-app"
}
```

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Instance name (lowercase, alphanumeric, hyphens only, max 63 chars) |

**Response:**
```json
{
  "id": 1,
  "name": "my-app",
  "namespace": "supa-my-app",
  "status": "Pending",
  "created_at": "2025-01-15T10:00:00Z"
}
```

**Status Codes:**
- `201 Created` - Instance creation initiated
- `400 Bad Request` - Invalid instance name
- `401 Unauthorized` - Invalid or missing token
- `409 Conflict` - Instance with this name already exists
- `500 Internal Server Error` - Kubernetes/Helm error

**Validation Rules:**
- Name must be lowercase
- Only alphanumeric characters and hyphens allowed
- Cannot start or end with a hyphen
- Maximum 63 characters (Kubernetes limit)
- Must be unique

**Example:**
```bash
curl -X POST https://supacontrol.example.com/api/v1/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-production-app"
  }'
```

**What Happens:**
1. API validates the instance name
2. Creates Kubernetes namespace `supa-my-app`
3. Installs Supabase Helm chart in the namespace
4. Configures ingress with domain `my-app.supabase.yourdomain.com`
5. Saves instance record to database
6. Returns instance details

**Note:** Instance creation is asynchronous. Status will be `Pending` initially, then change to `Running` once all pods are ready (typically 2-5 minutes).

#### Get Instance

Get details about a specific instance.

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
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T10:05:00Z"
}
```

**Status Values:**
- `Pending` - Instance is being created
- `Running` - Instance is operational
- `Failed` - Instance deployment failed
- `Deleting` - Instance is being deleted

**Status Codes:**
- `200 OK` - Success
- `401 Unauthorized` - Invalid or missing token
- `404 Not Found` - Instance not found

**Example:**
```bash
curl -X GET https://supacontrol.example.com/api/v1/instances/my-app \
  -H "Authorization: Bearer $TOKEN"
```

#### Delete Instance

Delete a Supabase instance and all its resources.

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

**Status Codes:**
- `200 OK` - Instance deletion initiated
- `401 Unauthorized` - Invalid or missing token
- `404 Not Found` - Instance not found
- `500 Internal Server Error` - Deletion failed

**What Happens:**
1. Uninstalls Helm release from namespace
2. Deletes Kubernetes namespace and all resources
3. Soft deletes instance record from database (sets `deleted_at`)

**Warning:** This operation is destructive and cannot be undone. All data in the instance will be permanently lost.

**Example:**
```bash
curl -X DELETE https://supacontrol.example.com/api/v1/instances/my-app \
  -H "Authorization: Bearer $TOKEN"
```

---

## Error Responses

All errors follow a consistent format:

```json
{
  "message": "Error description"
}
```

### Common HTTP Status Codes

| Code | Meaning | Description |
|------|---------|-------------|
| `200` | OK | Request succeeded |
| `201` | Created | Resource created successfully |
| `400` | Bad Request | Invalid request body or parameters |
| `401` | Unauthorized | Missing or invalid authentication token |
| `404` | Not Found | Resource not found |
| `409` | Conflict | Resource already exists |
| `500` | Internal Server Error | Server error (check logs) |

### Error Examples

**Missing Authentication:**
```json
{
  "message": "missing or malformed jwt"
}
```

**Invalid Instance Name:**
```json
{
  "message": "instance name must be lowercase alphanumeric with hyphens"
}
```

**Instance Already Exists:**
```json
{
  "message": "instance with name 'my-app' already exists"
}
```

**Instance Not Found:**
```json
{
  "message": "instance not found"
}
```

---

## Rate Limiting

Currently, there are no enforced rate limits. However, we recommend:
- Maximum 10 requests per second per API key
- Maximum 1000 requests per hour per API key

Rate limiting may be enforced in future versions.

---

## Pagination

Currently, list endpoints return all results. Pagination will be added in a future version.

**Future format:**
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

---

## Versioning

The API is versioned via the URL path: `/api/v1/`

**Current version:** v1

**Breaking changes:**
- Will be introduced in new versions (`/api/v2/`)
- v1 will be maintained for backward compatibility
- Deprecated endpoints will be documented

---

## SDKs and Client Libraries

**Official CLI:**
- [supactl](https://github.com/qubitquilt/supactl) - Go-based CLI tool

**Community Libraries:**
- Coming soon!

Want to create a client library? We'd love to feature it here. Open an issue to let us know!

---

## Webhooks (Future)

Webhook support is planned for future versions to notify you of:
- Instance creation completion
- Instance status changes
- Deployment failures
- API key usage

---

## Need Help?

- **API Issues**: [Open an issue](https://github.com/qubitquilt/SupaControl/issues)
- **Questions**: [GitHub Discussions](https://github.com/qubitquilt/SupaControl/discussions)
- **Security**: See [SECURITY.md](SECURITY.md)

---

**Last Updated**: November 2025
