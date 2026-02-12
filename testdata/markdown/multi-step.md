# Multi-Step E2E Workflow

This document describes a complex deployment workflow with multiple test groups.

## Prerequisites

Make sure you have `kubectl` and `helm` configured.

## Stage 1: Infrastructure Setup

<!-- test-start: Infrastructure provisioning -->

<!-- test-step-start: Setup Database -->

### Deploy Database

```go-e2e-step step-name="Install PostgreSQL via Helm"
helm install postgres bitnami/postgresql --set auth.postgresPassword=testpass
```

<!-- test-step-end -->

<!-- test-step-start: Wait for Ready -->

### Wait for Database

```go-e2e-step timeout=120s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql --timeout=180s
```

<!-- test-step-end -->

<!-- test-end -->

## Stage 2: Application Deployment

<!-- test-start: Application deployment -->

### Build and Push Image

```go-e2e-step step-name="Build Docker image"
docker build -t myapp:e2e-test .
```

### Deploy Application

```go-e2e-step
kubectl apply -f ./k8s/
```

### Verify Application Health

```go-e2e-step step-name="Check application health endpoint" timeout=30s
curl -f http://localhost:8080/healthz
```

<!-- test-end -->

## Cleanup

This section has no test tags and should be ignored by the generator.

```bash
kubectl delete -f ./k8s/
helm uninstall postgres
```
