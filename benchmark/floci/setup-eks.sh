#!/bin/bash
set -euo pipefail

echo "======================================"
echo "  Floci EKS Simulation Setup (Go)"
echo "======================================"
echo ""

AWS_ENDPOINT="http://localhost:4566"
REGION="us-east-1"
CLUSTER_NAME="payment-eks"
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export AWS_DEFAULT_REGION="${REGION}"

echo "[1/8] Waiting for Floci to be healthy..."
for i in $(seq 1 30); do
  if curl -sf "${AWS_ENDPOINT}/_localstack/health" > /dev/null 2>&1; then
    echo "  Floci is healthy!"
    break
  fi
  echo "  Attempt ${i}/30... waiting 5s"
  sleep 5
done

echo ""
echo "[2/8] Creating EKS cluster: ${CLUSTER_NAME}..."
aws eks create-cluster \
  --endpoint-url "${AWS_ENDPOINT}" \
  --region "${REGION}" \
  --name "${CLUSTER_NAME}" \
  --role-arn "arn:aws:iam::000000000000:role/eks-role" \
  --resources-vpc-config "subnetIds=subnet-12345,subnet-67890,securityGroupIds=sg-12345" \
  --kubernetes-version "1.29" || true

echo ""
echo "[3/8] Building Go payment-service Docker image..."
docker build -t payment-service:latest -f ../Dockerfile ../
echo "  Image built: payment-service:latest"

echo ""
echo "[4/8] Loading image into Floci k3s registry..."
docker tag payment-service:latest localhost:5000/payment-service:latest
docker push localhost:5000/payment-service:latest || true

echo ""
echo "[5/8] Setting up kubeconfig..."
aws eks update-kubeconfig \
  --endpoint-url "${AWS_ENDPOINT}" \
  --region "${REGION}" \
  --name "${CLUSTER_NAME}" || true

if [ ! -f ~/.kube/config ]; then
  echo "  Creating kubeconfig for k3s..."
  kubectl config set-cluster floci-eks --server="https://localhost:6443" --insecure-skip-tls-verify=true
  kubectl config set-credentials floci-admin --token="floci-test-token"
  kubectl config set-context floci --cluster=floci-eks --user=floci-admin --namespace=payments
  kubectl config use-context floci
fi

echo ""
echo "[6/8] Deploying PostgreSQL + Payment Service..."
kubectl apply -f payment-service-elb.yaml

echo ""
echo "[7/8] Waiting for metrics server..."
for i in $(seq 1 20); do
  if kubectl get --raw "/apis/metrics.k8s.io/v1beta1/nodes" > /dev/null 2>&1; then
    echo "  Metrics server available!"
    break
  fi
  echo "  Attempt ${i}/20... waiting 5s"
  sleep 5
done

echo ""
echo "[8/8] Applying HPA..."
kubectl apply -f hpa.yaml

echo ""
echo "======================================"
echo "  Floci EKS Simulation Ready (Go)"
echo "======================================"
echo ""
echo "External IP:"
kubectl get svc payment-service -n payments -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || echo "  localhost:8080 (via port-forward)"
echo ""
echo "Usage:"
echo "  kubectl get pods -n payments"
echo "  kubectl get hpa -n payments"
echo "  kubectl -n payments port-forward svc/payment-service 8080:80"
echo ""
echo "Run benchmarks:"
echo "  BASE_URL=http://localhost:8080 k6 run benchmark/k6/payment-service-load-test.js"
