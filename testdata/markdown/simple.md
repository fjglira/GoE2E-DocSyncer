# Simple Deployment Guide

This is a basic deployment guide for testing the Markdown parser.

## Deploy the Application

<!-- test-start: Simple deployment test -->

### Apply Kubernetes Manifests

Deploy the application to the cluster:

```go-e2e-step step-name="Apply deployment manifests"
kubectl apply -f ./manifests/deployment.yaml
```

### Verify Deployment

Wait for the deployment to become ready:

```go-e2e-step timeout=60s
kubectl wait --for=condition=ready pod -l app=myapp --timeout=120s
```

### Check Service

Verify the service is accessible:

```go-e2e-step
kubectl get svc myapp-service -o jsonpath='{.spec.clusterIP}'
```

<!-- test-end -->
