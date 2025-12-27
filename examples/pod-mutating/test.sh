#!/bin/bash
# Test script for pod-label-injector mutating webhook
# This script tests that the webhook correctly injects labels into pods

set -e

NAMESPACE="${TEST_NAMESPACE:-default}"
echo "Testing in namespace: $NAMESPACE"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    exit 1
}

cleanup() {
    echo ""
    echo "=== Cleanup ==="
    kubectl delete pod test-mutating-simple test-mutating-with-labels --namespace="$NAMESPACE" --ignore-not-found 2>/dev/null || true
}

# Cleanup on exit
trap cleanup EXIT

echo "=== Test 1: Pod without labels gets labels injected ==="
# Create a pod - need to satisfy validating webhook if deployed
kubectl run test-mutating-simple \
    --image=nginx:1.25 \
    --namespace="$NAMESPACE" \
    --restart=Never \
    --labels="app=test" \
    --overrides='{
        "spec": {
            "containers": [{
                "name": "test-mutating-simple",
                "image": "nginx:1.25",
                "resources": {
                    "limits": {"cpu": "100m", "memory": "128Mi"},
                    "requests": {"cpu": "50m", "memory": "64Mi"}
                }
            }]
        }
    }' 2>/dev/null

sleep 2

# Check if labels were injected
LABELS=$(kubectl get pod test-mutating-simple --namespace="$NAMESPACE" -o jsonpath='{.metadata.labels}')

if echo "$LABELS" | grep -q "injected-by"; then
    pass "Label 'injected-by' was injected"
else
    fail "Label 'injected-by' was NOT injected. Labels: $LABELS"
fi

if echo "$LABELS" | grep -q "injection-time"; then
    pass "Label 'injection-time' was injected"
else
    fail "Label 'injection-time' was NOT injected. Labels: $LABELS"
fi

echo ""
echo "=== Test 2: Pod with existing labels preserves them ==="
kubectl run test-mutating-with-labels \
    --image=nginx:1.25 \
    --namespace="$NAMESPACE" \
    --restart=Never \
    --labels="app=myapp,env=test" \
    --overrides='{
        "spec": {
            "containers": [{
                "name": "test-mutating-with-labels",
                "image": "nginx:1.25",
                "resources": {
                    "limits": {"cpu": "100m", "memory": "128Mi"},
                    "requests": {"cpu": "50m", "memory": "64Mi"}
                }
            }]
        }
    }' 2>/dev/null

sleep 2

LABELS=$(kubectl get pod test-mutating-with-labels --namespace="$NAMESPACE" -o jsonpath='{.metadata.labels}')

if echo "$LABELS" | grep -q "app.*myapp"; then
    pass "Original label 'app=myapp' was preserved"
else
    fail "Original label 'app=myapp' was NOT preserved. Labels: $LABELS"
fi

if echo "$LABELS" | grep -q "env.*test"; then
    pass "Original label 'env=test' was preserved"
else
    fail "Original label 'env=test' was NOT preserved. Labels: $LABELS"
fi

if echo "$LABELS" | grep -q "injected-by"; then
    pass "Label 'injected-by' was also injected"
else
    fail "Label 'injected-by' was NOT injected. Labels: $LABELS"
fi

echo ""
echo "=== All tests passed! ==="
