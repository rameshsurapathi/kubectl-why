#!/bin/bash
set -e

echo "🚀 Building kubectl-why..."
go build -o kubectl-why .

echo "📦 Creating Kind cluster (if not exists)..."
if ! kind get clusters | grep -q "why-e2e"; then
  kind create cluster --name why-e2e
fi

echo "☸️ Switching context to kind-why-e2e..."
kubectl config use-context kind-why-e2e

echo "🗑️ Cleaning up previous test runs..."
helm uninstall test-env -n kubectl-why-tests 2>/dev/null || true
kubectl delete namespace kubectl-why-tests 2>/dev/null || true

echo "🏗️ Creating test namespace..."
kubectl create namespace kubectl-why-tests

echo "🚀 Installing intentionally broken Helm chart..."
helm install test-env ./tests/e2e/test-chart -n kubectl-why-tests

echo "⏳ Waiting 45 seconds for pods to enter failing states (image pulling takes time)..."
sleep 45

echo "========================================================="
echo "🧪 Running Automated Command Tests..."
echo "========================================================="

# Helper function to run a command and display its output clearly
run_demo() {
  local cmd="$1"
  echo -e "\n─────────────────────────────────────────────────────────"
  echo "🚀 Running: kubectl-why $cmd -n kubectl-why-tests"
  echo "─────────────────────────────────────────────────────────"
  ./kubectl-why $cmd -n kubectl-why-tests || true
}

run_demo "deployment bad-image-deploy"

POD_NAME=$(kubectl get pod -n kubectl-why-tests -l app=crashloop -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
if [ -n "$POD_NAME" ]; then
  run_demo "pod $POD_NAME"
else
  echo "⚠️ Could not find crashloop pod to test."
fi

run_demo "service orphaned-service"
run_demo "job failing-job"
run_demo "cronjob suspended-cronjob"
run_demo "hpa broken-hpa"
run_demo "statefulset pending-sts"
run_demo "daemonset bad-daemonset"
run_demo "tls expired-tls"
run_demo "networkpolicy default-deny-all"
run_demo "rolebinding dangling-rolebinding"
run_demo "pdb zero-disruption-pdb"

echo -e "\n========================================================="
echo "✅ All demonstrations complete!"
echo "Now run the interactive dashboard to explore further:"
echo "  ./kubectl-why view -n kubectl-why-tests"
echo "========================================================="

echo "✅ Test environment is ready for manual inspection!"
echo "To clean up: kind delete cluster --name why-e2e"
