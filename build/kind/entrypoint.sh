#!/bin/bash

# Turn on bash's job control
set -m

# Logging function to make it easier to debug
log() {
  echo "[INFO] $1"
}

# Step 1: Start docker service in background
/usr/local/bin/dockerd-entrypoint.sh &

# Step 2: Wait that the docker service is up
while ! docker info; do
  log "Waiting docker for 5 seconds..."
  sleep 5
done

# Step 3: Import pre-installed images
for file in /images/*.tar; do
  log "Load docker image $file..."
  docker load <$file
done

# Step 4: Create Kind cluster using a specific Kubernetes version
log "Creating Kubernetes cluster using Kind (Kubernetes v1.31.0)..."
kind create cluster --name testkube-cluster --image kindest/node:v1.31.0 --wait 5m
if [ $? -ne 0 ]; then
  log "Failed to create Kind cluster."
  exit 1
fi

# Step 5: Verify kubectl is connected to the cluster
log "Verifying cluster is up..."
kubectl cluster-info
if [ $? -ne 0 ]; then
  log "Failed to verify cluster."
  exit 1
fi

# Step 6: Add the Testkube Helm repository
log "Adding Testkube Helm repository..."
helm repo add testkube https://kubeshop.github.io/helm-charts
helm repo update

# Step 7: Install Testkube using Helm
log "Installing Testkube via Helm..."
helm install testkube testkube/testkube --namespace testkube --create-namespace
if [ $? -ne 0 ]; then
  log "Testkube installation failed."
  exit 1
fi

# Step 8: Wait that the Testkube is up
log "Waiting Testkube for 180 seconds..."
sleep 180

# Step 9: Verify Testkube is installed to the cluster
log "Verifying Testkube is up..."
kubectl get pods -n testkube
if [ $? -ne 0 ]; then
  log "Failed to verify Testkube."
  exit 1
fi

# Step 10: Create and Run Testkube k6 Test Workflow 
log "Creating and running Testkube k6 Test Workflow..."
kubectl apply -f /examples/k6.yaml -n testkube
kubectl testkube run testworkflow k6-workflow-smoke -f

log "Testkube installation successful!"
log "You can now use Testkube in your Kind Kubernetes cluster."

# Step 11: Bring docker service back to foreground
fg %1
