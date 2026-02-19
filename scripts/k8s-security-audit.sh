#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-openoms}"
PASS=0
FAIL=0

check() {
    local name="$1" result="$2"
    if [ "$result" = "true" ]; then
        echo "PASS: $name"
        PASS=$((PASS + 1))
    else
        echo "FAIL: $name"
        FAIL=$((FAIL + 1))
    fi
}

echo "=== K8s Security Audit — namespace: $NAMESPACE ==="

# K8S-001: runAsNonRoot
NON_ROOT=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{range .items[*]}{.spec.containers[*].securityContext.runAsNonRoot}{"\n"}{end}' | grep -c "false" || true)
check "All pods runAsNonRoot" "$([ "$NON_ROOT" = "0" ] && echo true || echo false)"

# K8S-002: Drop ALL capabilities
DROP_ALL=$(kubectl get pods -n "$NAMESPACE" -o json | jq '[.items[].spec.containers[].securityContext.capabilities.drop // [] | contains(["ALL"])] | all')
check "All containers drop ALL capabilities" "$DROP_ALL"

# K8S-003: readOnlyRootFilesystem on API
API_ROFS=$(kubectl get deploy openoms-api -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].securityContext.readOnlyRootFilesystem}')
check "API readOnlyRootFilesystem" "$([ "$API_ROFS" = "true" ] && echo true || echo false)"

# K8S-004: Resource limits set
NO_LIMITS=$(kubectl get pods -n "$NAMESPACE" -o json | jq '[.items[].spec.containers[] | select(.resources.limits == null)] | length')
check "Resource limits on all containers" "$([ "$NO_LIMITS" = "0" ] && echo true || echo false)"

# K8S-005: Liveness probes configured
NO_LIVENESS=$(kubectl get pods -n "$NAMESPACE" -o json | jq '[.items[].spec.containers[] | select(.livenessProbe == null)] | length')
check "Liveness probes on all containers" "$([ "$NO_LIVENESS" = "0" ] && echo true || echo false)"

# K8S-006: No hostPath volumes
HOST_PATHS=$(kubectl get pods -n "$NAMESPACE" -o json | jq '[.items[].spec.volumes[]? | select(.hostPath != null)] | length')
check "No hostPath volumes" "$([ "$HOST_PATHS" = "0" ] && echo true || echo false)"

# K8S-007: Secrets are Opaque
NON_OPAQUE=$(kubectl get secrets -n "$NAMESPACE" -o json | jq '[.items[] | select(.type != "Opaque" and .type != "kubernetes.io/service-account-token" and .type != "helm.sh/release.v1")] | length')
check "Secrets are Opaque type" "$([ "$NON_OPAQUE" = "0" ] && echo true || echo false)"

# K8S-008: NetworkPolicy exists
NP_COUNT=$(kubectl get networkpolicies -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l | tr -d ' ')
check "NetworkPolicy exists" "$([ "$NP_COUNT" -gt 0 ] && echo true || echo false)"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit "$FAIL"
