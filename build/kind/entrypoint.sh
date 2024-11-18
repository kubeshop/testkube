#!/bin/bash

# Turn on bash's job control
set -m

# Logging function to make it easier to debug
log() {
  echo "[INFO] $1"
}

_detect_arch() {
    case $(uname -m) in
    amd64|x86_64) echo "x86_64"
    ;;
    arm64|aarch64) echo "arm64"
    ;;
    i386) echo "i386"
    ;;
    *) echo "Unsupported processor architecture";
    ;;
     esac
}

_detect_os(){
    case $(uname) in
    Linux) echo "Linux"
    ;;
    Darwin) echo "Darwin"
    ;;
    Windows) echo "Windows"
    ;;
     esac
}

_detect_version() {
  local tag

  tag="$(
    curl -s "https://api.github.com/repos/kubeshop/testkube/releases/latest" \
      2>/dev/null \
      | jq -r '.tag_name' \
  )"

  echo "${tag/#v/}" # remove leading v if present

}

_calculate_machine_id() {
  local hash

# Calculate hash using md5sum
  hash=$(echo -n "$(hostname)" | md5sum | awk '{print $1}')

  echo "$hash"
}

version="$(_detect_version)"
arch="$(_detect_arch)"
os="$(_detect_os)"
machine_id="$(_calculate_machine_id)"

# Function to send event message to Segment.io
send_event_to_segment() {
  local event="$1"  
  local error_code="$2"
  local error_type="$3"
  local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
  # Prepare the JSON payload
  local payload=$(cat <<EOF
{
  "userId":               "$machine_id",
  "event":                "$event",
  "properties": {
    "name":               "testkube-api-server",
    "version":            "$version",
    "arch":               "$arch",
    "os":                 "$os",
    "eventCategory":      "api",
    "contextType":        "agent",
    "machineId":          "$machine_id",
    "clusterType":        "kind",
    "errorType":          "$error_type",
    "errorCode":          "$error_code",
    "agentKey":           "$AGENT_KEY",
    "dockerImageVersion": "$DOCKER_IMAGE_VERSION"
  },
  "context": {
    "app": {
      "name":        "testkube-api-server",
      "version":     "$version",
      "build":       "cloud"
    }
  },
  "timestamp": "$timestamp"
}
EOF
)

  # Send the message to Segment via HTTP API
  curl -X POST -H "Content-Type: application/json" -u "$SEGMENTIO_KEY:" -d "$payload" https://api.segment.io/v1/track

  # Check if the curl command was successful
  if [ $? -eq 0 ]; then
      log "Message successfully sent to Segment."
  else
      log "Failed to send message to Segment."
  fi
}

# Function to send event message to GA
send_event_to_ga() {
  local event="$1"  
  local error_code="$2"
  local error_type="$3"
    
  # Prepare the JSON payload
  local payload=$(cat <<EOF
{
  "client_id":                 "$machine_id",
  "user_id":                   "$machine_id",
  "events": [{
    "name":                    "$event",
    "params": {
      "event_count":            1,
      "event_category":         "api",
      "app_version":            "$version",
      "app_name":               "testkube-api-server",
      "machine_id":             "$machine_id",
      "operating_system":       "$os",
      "architecture":           "$arch",
      "context": {
        "docker_image_version": "$DOCKER_IMAGE_VERSION",
        "type":                 "agent"
      },
      "cluster_type":           "kind",
      "error_type":             "$error_type",
      "error_code":             "$error_code",
      "agent_key":              "$AGENT_KEY"
    }
  }]
}
EOF
)

  # Send the message to GA via HTTP API
  curl -X POST -H "Content-Type: application/json" -d "$payload" "https://www.google-analytics.com/mp/collect?measurement_id=$GA_ID&api_secret=$GA_SECRET"

  # Check if the curl command was successful
  if [ $? -eq 0 ]; then
      log "Message successfully sent to GA."
  else
      log "Failed to send message to GA."
  fi

}

send_telenetry() {
  local event="$1"  
  local error_code="$2"
  local error_type="$3"

  send_event_to_segment "$event" "$error_code" "$error_type"
  send_event_to_ga "$event" "$error_code" "$error_type"
}

send_telenetry "docker_installation_started"

# Check if agent key is provided
if [ -z "$AGENT_KEY" ]; then
  log "Testkube installation failed. Please provide AGENT_KEY env var"
  send_telenetry "docker_installation_failed" "parameter_not_found" "agent key is empty"
  exit 1
fi

# Step 1: Start docker service in background
/usr/local/bin/dockerd-entrypoint.sh &

# Step 2: Wait that the docker service is up
while ! docker info; do
  log "Waiting docker for 5 seconds..."
  sleep 5
done

# Set image folder based on architecture
case "$arch" in
  x86_64)
    IMAGE_FOLDER="/images/amd"
    ;;
  arm64)
    IMAGE_FOLDER="/images/arm"
    ;;
  *)
    log "Unsupported architecture: $arch"
    exit 1
    ;;
esac

# Step 3: Import pre-installed images
for file in "$IMAGE_FOLDER"/*.tar; do
  log "Load docker image $file..."
  docker load < "$file"
done

# Get the list of kind clusters
EXISTING_CLUSTERS=$(kind get clusters)
# Check if the testkube-cluster exists in the list
if echo "$EXISTING_CLUSTERS" | grep -wq "testkube-cluster"; then
  log "Kind cluster already exists"
else
  # Step 4: Create Kind cluster using a specific Kubernetes version
  log "Creating Kubernetes cluster using Kind (Kubernetes v1.31.0)..."
  kind create cluster --name testkube-cluster --image kindest/node:v1.31.0 --wait 5m
  if [ $? -ne 0 ]; then
    log "Testkube installation failed. Couldn't create Kind cluster."
    send_telenetry "docker_installation_failed" "kind_error" "Kind cluster was not created"
    exit 1
  fi

  # Step 5: Verify kubectl is connected to the cluster
  log "Verifying cluster is up..."
  kubectl cluster-info
  if [ $? -ne 0 ]; then
    log "Testkube installation failed. Couldn't verify cluster."
    send_telenetry "docker_installation_failed" "kind_error" "Kind cluster is nor accessible"
    exit 1
  fi

  # Step 6: Add the Testkube Helm repository
  log "Adding Testkube Helm repository..."
  helm repo add testkube https://kubeshop.github.io/helm-charts
  helm repo update

  # Step 7: Install Testkube using Helm
  log "Installing Testkube via Helm..."
  helm install testkube testkube/testkube --namespace testkube --create-namespace  --set testkube-api.cloud.key=$AGENT_KEY --set testkube-api.minio.enabled=false --set mongodb.enabled=false --set testkube-dashboard.enabled=false --set testkube-api.cloud.url=$CLOUD_URL --set testkube-api.dockerImageVersion=$DOCKER_IMAGE_VERSION
  if [ $? -ne 0 ]; then
    log "Testkube installation failed."
    send_telenetry "docker_installation_failed" "helm_error" "Testkube installation failed"
    exit 1
  fi

  # Step 8: Verify Testkube is installed to the cluster
  log "Verifying Testkube is up..."
  counter=0
  log_pattern="starting Testkube API server"
  while [ $counter -lt 15 ]
  do
    # Get all pod statuses in the Testkube namespace
    pod_status=$(kubectl get pods -n testkube --no-headers)

    # Check if there are any pods in the Testkube namespace
    if [ -z "$pod_status" ]; then
      log "Testkube installation failed. No pods found in testkube namespace."
      send_telenetry "docker_installation_failed" "tetkube_error" "No pods found in testkube namespace"
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
    log "Testkube installation failed."
    send_telenetry "docker_installation_failed" "tetkube_error" "Testkube pods are not up and running"
    exit 1
  fi
  log "Testkube is up and running."

  log "Testkube installation succeed!"
  log "You can now use Testkube in your Kind Kubernetes cluster."
  send_telenetry "docker_installation_succeed"
fi

# Step 9: Bring docker service back to foreground
fg %1
