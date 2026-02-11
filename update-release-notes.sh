#!/bin/bash

# Script to update GitHub release notes for tag 2.6.1
# This script uses the GitHub API to update an existing release with detailed release notes

set -e

OWNER="kubeshop"
REPO="testkube"
TAG="2.6.1"
RELEASE_ID="285366832"

# Check if GITHUB_TOKEN is set
if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN environment variable is not set"
    echo "Please set it with: export GITHUB_TOKEN=your_token"
    exit 1
fi

# Read the release notes from file
RELEASE_NOTES_FILE="/tmp/release-notes-2.6.1.md"

if [ ! -f "$RELEASE_NOTES_FILE" ]; then
    echo "Error: Release notes file not found at $RELEASE_NOTES_FILE"
    exit 1
fi

# Read and prepare the release body
RELEASE_BODY=$(cat "$RELEASE_NOTES_FILE")

# Create a temporary JSON file
TMP_JSON=$(mktemp)
jq -n \
  --arg body "$RELEASE_BODY" \
  '{body: $body}' > "$TMP_JSON"

echo "Updating release $TAG (ID: $RELEASE_ID)..."

# Update the release using GitHub API
RESPONSE=$(curl -L \
  -X PATCH \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$OWNER/$REPO/releases/$RELEASE_ID" \
  -d @"$TMP_JSON" \
  2>&1)

# Clean up
rm -f "$TMP_JSON"

# Check if the update was successful
if echo "$RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    echo "✅ Release notes updated successfully!"
    echo "View the release at: https://github.com/$OWNER/$REPO/releases/tag/$TAG"
else
    echo "❌ Failed to update release"
    echo "Response:"
    echo "$RESPONSE" | jq . || echo "$RESPONSE"
    exit 1
fi
