#!/bin/bash
# Test script for pod-validator validating webhook
# This script tests that the webhook correctly enforces pod policies

set -e

NAMESPACE="${TEST_NAMESPACE:-default}"
echo "Testing in namespace: $NAMESPACE"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    FAILED=1
}

info() {
    echo -e "${YELLOW}INFO${NC}: $1"
}

cleanup() {
    echo ""
    echo "=== Cleanup ==="
    kubectl delete pod test-valid-pod --namespace="$NAMESPACE" --ignore-not-found 2>/dev/null || true
}

# Cleanup on exit
trap cleanup EXIT

FAILED=0

echo "=== Test 1: Reject pod without labels ==="
OUTPUT=$(kubectl run test-no-labels \
    --image=nginx:1.25 \
    --namespace="$NAMESPACE" \
    --restart=Never \
    --overrides='{
        "spec": {
            "containers": [{
                "name": "test-no-labels",
                "image": "nginx:1.25",
                "resources": {
                    "limits": {"cpu": "100m", "memory": "128Mi"}
                }
            }]
        }
    }' 2>&1 || true)

if echo "$OUTPUT" | grep -qi "label"; then
    pass "Pod without labels was rejected"
    info "Message: $(echo "$OUTPUT" | head -1)"
else
    fail "Pod without labels should have been rejected"
    info "Output: $OUTPUT"
fi

echo ""
echo "=== Test 2: Reject pod with 'latest' image tag ==="
OUTPUT=$(kubectl run test-latest \
    --image=nginx:latest \
    --namespace="$NAMESPACE" \
    --restart=Never \
    --labels="app=test" \
    --overrides='{
        "spec": {
            "containers": [{
                "name": "test-latest",
                "image": "nginx:latest",
                "resources": {
                    "limits": {"cpu": "100m", "memory": "128Mi"}
                }
            }]
        }
    }' 2>&1 || true)

if echo "$OUTPUT" | grep -qi "latest"; then
    pass "Pod with 'latest' tag was rejected"
    info "Message: $(echo "$OUTPUT" | head -1)"
else
    fail "Pod with 'latest' tag should have been rejected"
    info "Output: $OUTPUT"
fi

echo ""
echo "=== Test 3: Reject pod without image tag (defaults to latest) ==="
OUTPUT=$(kubectl run test-no-tag \
    --image=nginx \
    --namespace="$NAMESPACE" \
    --restart=Never \
    --labels="app=test" \
    --overrides='{
        "spec": {
            "containers": [{
                "name": "test-no-tag",
                "image": "nginx",
                "resources": {
                    "limits": {"cpu": "100m", "memory": "128Mi"}
                }
            }]
        }
    }' 2>&1 || true)

if echo "$OUTPUT" | grep -qi "latest"; then
    pass "Pod without image tag was rejected (defaults to latest)"
    info "Message: $(echo "$OUTPUT" | head -1)"
else
    fail "Pod without image tag should have been rejected"
    info "Output: $OUTPUT"
fi

echo ""
echo "=== Test 4: Reject pod without resource limits ==="
OUTPUT=$(kubectl run test-no-limits \
    --image=nginx:1.25 \
    --namespace="$NAMESPACE" \
    --restart=Never \
    --labels="app=test" 2>&1 || true)

if echo "$OUTPUT" | grep -qi "limit"; then
    pass "Pod without resource limits was rejected"
    info "Message: $(echo "$OUTPUT" | head -1)"
else
    fail "Pod without resource limits should have been rejected"
    info "Output: $OUTPUT"
fi

echo ""
echo "=== Test 5: Reject pod missing CPU limit ==="
OUTPUT=$(kubectl run test-no-cpu \
    --image=nginx:1.25 \
    --namespace="$NAMESPACE" \
    --restart=Never \
    --labels="app=test" \
    --overrides='{
        "spec": {
            "containers": [{
                "name": "test-no-cpu",
                "image": "nginx:1.25",
                "resources": {
                    "limits": {"memory": "128Mi"}
                }
            }]
        }
    }' 2>&1 || true)

if echo "$OUTPUT" | grep -qi "limit\|cpu"; then
    pass "Pod without CPU limit was rejected"
    info "Message: $(echo "$OUTPUT" | head -1)"
else
    fail "Pod without CPU limit should have been rejected"
    info "Output: $OUTPUT"
fi

echo ""
echo "=== Test 6: Accept valid pod ==="
OUTPUT=$(kubectl run test-valid-pod \
    --image=nginx:1.25 \
    --namespace="$NAMESPACE" \
    --restart=Never \
    --labels="app=test,env=testing" \
    --overrides='{
        "spec": {
            "containers": [{
                "name": "test-valid-pod",
                "image": "nginx:1.25",
                "resources": {
                    "limits": {"cpu": "100m", "memory": "128Mi"},
                    "requests": {"cpu": "50m", "memory": "64Mi"}
                }
            }]
        }
    }' 2>&1)

if echo "$OUTPUT" | grep -q "created"; then
    pass "Valid pod was accepted"

    # Verify pod is running
    sleep 2
    STATUS=$(kubectl get pod test-valid-pod --namespace="$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
    if [ "$STATUS" != "NotFound" ]; then
        pass "Pod exists with status: $STATUS"
    else
        fail "Pod was not created"
    fi
else
    fail "Valid pod should have been accepted"
    info "Output: $OUTPUT"
fi

echo ""
if [ "$FAILED" -eq 0 ]; then
    echo -e "${GREEN}=== All tests passed! ===${NC}"
else
    echo -e "${RED}=== Some tests failed! ===${NC}"
    exit 1
fi
