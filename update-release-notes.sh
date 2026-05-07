#!/bin/bash

# Script to update GitHub release notes
# This script uses the GitHub API to update an existing release with detailed release notes
# Usage: ./update-release-notes.sh [TAG] [NOTES_FILE]
#   TAG: Release tag (default: 2.6.1)
#   NOTES_FILE: Path to release notes file (default: docs/release-notes/2.6.1.md)

set -e

OWNER="kubeshop"
REPO="testkube"
TAG="${1:-2.6.1}"
RELEASE_NOTES_FILE="${2:-docs/release-notes/${TAG}.md}"

# Check if GITHUB_TOKEN is set
if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN environment variable is not set"
    echo "Please set it with: export GITHUB_TOKEN=your_token"
    exit 1
fi

# Read the release notes from file

if [ ! -f "$RELEASE_NOTES_FILE" ]; then
    echo "Error: Release notes file not found at $RELEASE_NOTES_FILE"
    exit 1
fi

# Read and prepare the release body
RELEASE_BODY=$(cat "$RELEASE_NOTES_FILE")

echo "Fetching release information for tag $TAG..."

# First, get the release ID from the tag
RELEASE_INFO=$(curl -L \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$OWNER/$REPO/releases/tags/$TAG" \
  2>&1)

RELEASE_ID=$(echo "$RELEASE_INFO" | jq -r '.id')

if [ "$RELEASE_ID" = "null" ] || [ -z "$RELEASE_ID" ]; then
    echo "Error: Could not find release for tag $TAG"
    echo "Response: $RELEASE_INFO"
    exit 1
fi

echo "Found release ID: $RELEASE_ID"

# Create a temporary JSON file
TMP_JSON=$(mktemp)
jq -n \
  --arg body "$RELEASE_BODY" \
  '{body: $body}' > "$TMP_JSON"

echo "Updating release $TAG..."

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
