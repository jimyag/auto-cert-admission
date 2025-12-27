# Pod Label Injector Example

This example demonstrates a mutating webhook that automatically injects labels into pods.

## What It Does

When a pod is created in any namespace (except `kube-system` and `webhook-system`), the webhook automatically adds:

- `injected-by: pod-label-injector` - Indicates the webhook modified this pod
- `injection-time: admission` - Indicates when the modification occurred

## Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured
- Docker (for building images)

## Quick Start

### 1. Build and Push Image

```bash
cd examples/pod-mutating

# Build and push with auto-generated timestamp tag
make docker-build-push

# Or specify a custom tag
make docker-build-push IMAGE_TAG=v1.0.0

# Or use a different registry
make docker-build-push IMAGE_REGISTRY=docker.io IMAGE_REPO=myrepo
```

### 2. Update Image in Deployment

Edit `deploy/03-deployment.yaml` and update the image with the tag from step 1:

```yaml
image: registry.i.jimyag.com/test/pod-label-injector:20251227-20-00-00
```

### 3. Deploy to Kubernetes

```bash
make deploy
```

Or manually:

```bash
kubectl apply -f deploy/
```

### 4. Verify Deployment

```bash
# Check pods are running
kubectl -n webhook-system get pods

# Check logs
kubectl -n webhook-system logs -l app.kubernetes.io/name=pod-label-injector -f

# Check webhook configuration
kubectl get mutatingwebhookconfiguration pod-label-injector -o yaml
```

The `caBundle` field should be automatically populated after the webhook starts.

### 5. Test the Webhook

Run the automated test script:

```bash
make test
```

Or test manually:

```bash
# Create a test pod
kubectl run test-pod --image=nginx:1.25 --restart=Never \
  --labels="app=test" \
  --overrides='{"spec":{"containers":[{"name":"test-pod","image":"nginx:1.25","resources":{"limits":{"cpu":"100m","memory":"128Mi"}}}]}}'

# Check the labels (should include injected-by and injection-time)
kubectl get pod test-pod --show-labels

# Cleanup
kubectl delete pod test-pod
```

### 6. Cleanup

```bash
make undeploy

# Or manually
kubectl delete -f deploy/
```

## Configuration

The webhook can be configured via environment variables or in code. Environment variables use the `ACW_` prefix and override code defaults.

### Code Configuration

```go
func (p *podLabelInjector) Configure() webhook.Config {
    return webhook.Config{
        Name:           "pod-label-injector",
        // Namespace:   "custom-namespace",  // default: auto-detected
        // Port:        8443,                // default: 8443
        // MetricsPort: 8080,                // default: 8080
    }
}
```

### Environment Variables

See the [main README](../../README.md#environment-variables) for the complete list of environment variables.

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build binary locally |
| `make docker-build` | Build docker image |
| `make docker-push` | Push docker image |
| `make docker-build-push` | Build and push docker image (ensures same tag) |
| `make deploy` | Deploy to Kubernetes |
| `make undeploy` | Remove from Kubernetes |
| `make test` | Run automated tests against deployed webhook |
| `make clean` | Clean build artifacts |
| `make help` | Show available targets |

## Project Structure

```
examples/pod-mutating/
├── main.go           # Webhook implementation
├── Dockerfile        # Container image build
├── Makefile          # Build automation
├── test.sh           # Automated test script
├── README.md         # This file
└── deploy/
    ├── 00-namespace.yaml      # Namespace
    ├── 01-serviceaccount.yaml # ServiceAccount
    ├── 02-rbac.yaml           # RBAC permissions
    ├── 03-deployment.yaml     # Deployment
    ├── 04-service.yaml        # Service
    └── 05-webhook.yaml        # MutatingWebhookConfiguration
```

## Troubleshooting

### Webhook not receiving requests

1. Check if the webhook pods are running and ready:
   ```bash
   kubectl -n webhook-system get pods
   ```

2. Check if the caBundle is populated:
   ```bash
   kubectl get mutatingwebhookconfiguration pod-label-injector -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | base64 -d | openssl x509 -text -noout
   ```

3. Check webhook logs:
   ```bash
   kubectl -n webhook-system logs -l app.kubernetes.io/name=pod-label-injector
   ```

### Certificate issues

The framework automatically manages certificates. Check the secrets:

```bash
kubectl -n webhook-system get secrets
kubectl -n webhook-system get secret pod-label-injector-cert -o yaml
```

### Leader election

With multiple replicas, only the leader manages certificates. Check leader:

```bash
kubectl -n webhook-system get lease pod-label-injector-leader -o yaml
```

## Metrics

Prometheus metrics are available at port 8080:

```bash
kubectl -n webhook-system port-forward svc/pod-label-injector 8080:8080
curl http://localhost:8080/metrics
```
