# Pod Label Injector Example

This example demonstrates a mutating webhook that automatically injects labels into pods.

## Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured
- Docker (for building images)

## Quick Start

### 1. Build and Push Image

```bash
# Build docker image
make docker-build IMAGE=your-registry/pod-label-injector:latest

# Push to registry
make docker-push IMAGE=your-registry/pod-label-injector:latest

# Or do both
make docker-build-push IMAGE=your-registry/pod-label-injector:latest
```

### 2. Update Image in Deployment

Edit `deploy/03-deployment.yaml` and update the image:

```yaml
image: your-registry/pod-label-injector:latest
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

Create a test pod in the default namespace:

```bash
kubectl run test-pod --image=nginx --restart=Never

# Check the labels
kubectl get pod test-pod --show-labels
```

You should see labels `injected-by=pod-label-injector` and `injection-time=admission`.

### 6. Cleanup

```bash
make undeploy

# Or manually
kubectl delete -f deploy/
```

## Configuration

The webhook can be customized by modifying `main.go`:

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

## Project Structure

```
examples/pod-mutating/
├── main.go           # Webhook implementation
├── Dockerfile        # Container image build
├── Makefile          # Build automation
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
