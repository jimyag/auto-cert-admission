# Pod Validator Example

This example demonstrates a validating webhook that enforces pod security policies.

## Policies Enforced

1. **Labels Required**: Pods must have at least one label
2. **No Latest Tag**: Container images cannot use the `latest` tag
3. **Resource Limits**: All containers must have CPU and memory limits defined

## Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured
- Docker (for building images)

## Quick Start

### 1. Build and Push Image

```bash
cd examples/pod-validating

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
image: registry.i.jimyag.com/test/pod-validator:20251227-20-00-00
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
kubectl -n webhook-system get pods -l app.kubernetes.io/name=pod-validator

# Check logs
kubectl -n webhook-system logs -l app.kubernetes.io/name=pod-validator -f

# Check webhook configuration (caBundle should be auto-populated)
kubectl get validatingwebhookconfiguration pod-validator -o yaml
```

### 5. Test the Webhook

Run the automated test script:

```bash
make test
```

The test script validates all policies:
- Rejects pods without labels
- Rejects pods with `latest` image tag
- Rejects pods without image tag (defaults to latest)
- Rejects pods without resource limits
- Rejects pods missing CPU limit
- Accepts pods that pass all validations

Or test manually:

#### Test 1: Missing Labels (Should be DENIED)

```bash
kubectl run test-no-labels --image=nginx:1.25 --restart=Never
# Expected: Error - pod must have at least one label
```

#### Test 2: Latest Tag (Should be DENIED)

```bash
kubectl run test-latest --image=nginx:latest --labels="app=test" --restart=Never
# Expected: Error - containers using 'latest' tag
```

#### Test 3: No Image Tag (Should be DENIED)

```bash
kubectl run test-no-tag --image=nginx --labels="app=test" --restart=Never
# Expected: Error - containers using 'latest' tag (no tag defaults to latest)
```

#### Test 4: Missing Resource Limits (Should be DENIED)

```bash
kubectl run test-no-limits --image=nginx:1.25 --labels="app=test" --restart=Never
# Expected: Error - containers missing resource limits
```

#### Test 5: Valid Pod (Should be ALLOWED)

```bash
kubectl run test-valid --image=nginx:1.25 --labels="app=test" --restart=Never \
  --overrides='{"spec":{"containers":[{"name":"test-valid","image":"nginx:1.25","resources":{"limits":{"cpu":"100m","memory":"128Mi"}}}]}}'
# Expected: pod/test-valid created
```

### 6. Cleanup

```bash
# Delete test pods
kubectl delete pod test-no-labels test-latest test-no-tag test-no-limits test-valid --ignore-not-found

# Undeploy webhook
make undeploy
```

## Configuration

### Webhook Configuration

Edit `deploy/05-webhook.yaml` to customize:

```yaml
# failurePolicy options:
# - Fail: reject pod if webhook unavailable (strict, recommended for production)
# - Ignore: allow pod if webhook unavailable (lenient)
failurePolicy: Fail

# namespaceSelector: exclude namespaces from validation
namespaceSelector:
  matchExpressions:
  - key: kubernetes.io/metadata.name
    operator: NotIn
    values:
    - kube-system
    - webhook-system
```

### Environment Variables

All configuration can be overridden via environment variables in `deploy/03-deployment.yaml`:

```yaml
env:
- name: ACW_PORT
  value: "8443"
- name: ACW_METRICS_PORT
  value: "8080"
- name: ACW_METRICS_ENABLED
  value: "true"
- name: ACW_LEADER_ELECTION
  value: "true"
```

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
examples/pod-validating/
├── main.go                    # Webhook implementation
├── Dockerfile                 # Container image build
├── Makefile                   # Build automation
├── test.sh                    # Automated test script
├── README.md                  # This file
└── deploy/
    ├── 00-namespace.yaml      # Namespace
    ├── 01-serviceaccount.yaml # ServiceAccount
    ├── 02-rbac.yaml           # RBAC permissions
    ├── 03-deployment.yaml     # Deployment
    ├── 04-service.yaml        # Service
    └── 05-webhook.yaml        # ValidatingWebhookConfiguration
```

## Customizing Policies

Edit `main.go` to add or modify validation rules:

```go
func (p *podValidator) validatePod(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
    // Add your custom validation logic here

    if someCondition {
        return webhook.Denied("Custom validation failed: reason")
    }

    return webhook.Allowed()
}
```

## Troubleshooting

### Webhook not receiving requests

1. Check if pods are running and ready:
   ```bash
   kubectl -n webhook-system get pods -l app.kubernetes.io/name=pod-validator
   ```

2. Check if caBundle is populated:
   ```bash
   kubectl get validatingwebhookconfiguration pod-validator \
     -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | base64 -d | openssl x509 -text -noout
   ```

3. Check webhook logs:
   ```bash
   kubectl -n webhook-system logs -l app.kubernetes.io/name=pod-validator --tail=100
   ```

### Pods being rejected unexpectedly

1. Check which policy is failing:
   ```bash
   kubectl run test --image=nginx:1.25 --dry-run=server -o yaml
   ```

2. The error message will indicate which validation rule failed.

### Certificate issues

The framework automatically manages certificates. Check:

```bash
kubectl -n webhook-system get secrets
kubectl -n webhook-system get configmap
```

## Metrics

Prometheus metrics are available at port 8080:

```bash
kubectl -n webhook-system port-forward svc/pod-validator 8080:8080
curl http://localhost:8080/metrics
```

## Comparison with Mutating Webhook

| Feature              | Validating Webhook | Mutating Webhook      |
| -------------------- | ------------------ | --------------------- |
| Can modify resources | No                 | Yes                   |
| Execution order      | After mutating     | Before validating     |
| Use case             | Policy enforcement | Resource modification |
| Response             | Allow/Deny only    | Allow/Deny + Patch    |
