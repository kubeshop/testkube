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

# Step 8: Verify Testkube is installed to the cluster
log "Verifying Testkube is up..."
counter=0
log_pattern="testkube-api acquired lease"
while [ $counter -lt 15 ]
do
  # Get all pod statuses in the Testkube namespace
  pod_status=$(kubectl get pods -n testkube --no-headers)

  # Check if there are any pods in the Testkube namespace
  if [ -z "$pod_status" ]; then
    log "No pods found in Testkube namespace."
    exit 1
  fi

  # Iterate through each pod, check status and log pattern
  all_running=true
  found_pattern=false

  log "Checking pods in Testkube namespace..."
  while read -r line; do
    pod_name=$(echo "$line" | awk '{print $1}')
    status=$(echo "$line" | awk '{print $3}')
    
    if [ "$status" != "Running" ]; then
      log "Pod $pod_name is not running. Status: $status."
      all_running=false
      break
    else
      log "Pod $pod_name is running."
    fi

    if [[ $pod_name == *"testkube-api-server"* ]]; then
      pod_logs=$(kubectl logs "$pod_name" -n testkube)

      # Check if logs contain the desired pattern
      if echo "$pod_logs" | grep -q "$log_pattern"; then
        log "Log pattern found: $log_pattern."
        found_pattern=true
      else
        log "Log pattern not found: $log_pattern."
        break
      fi
    fi
  done <<< "$pod_status"

  if [ "$all_running" = true ] && [ "$found_pattern" = true ] ; then
    log "Waiting Testkube API for 30 seconds..."
    sleep 30
    break
  else
    log "Waiting Testkube for 30 seconds..."
    sleep 30
  fi

  counter=$(( counter + 1 ))
done

if [ $counter -eq 15 ]; then
  log "Testkube validation failed."
  exit 1
fi
log "Testkube is up and running."

# Step 9: Create and Run Testkube k6 Test Workflow 
log "Creating and running Testkube k6 Test Workflow..."
kubectl apply -f /examples/k6.yaml -n testkube
kubectl testkube run testworkflow k6-workflow-smoke -f

log "Testkube installation successful!"
log "You can now use Testkube in your Kind Kubernetes cluster."

# Step 10: Bring docker service back to foreground
fg %1
