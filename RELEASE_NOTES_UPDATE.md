# How to Update Release Notes for v2.6.1

## Overview

This document provides step-by-step instructions for updating the GitHub release notes for Testkube v2.6.1 with the comprehensive release notes created in this PR.

## Background

Tag `2.6.1` was created and a GitHub release was published, but the release notes were incomplete. Specifically, the important CVE fix ([#7036](https://github.com/kubeshop/testkube/pull/7036)) was missing from the changelog.

The comprehensive release notes have been prepared in `docs/release-notes/2.6.1.md` and include:

- **Security Updates**: CVE fixes with Alpine base image update to 3.23.3
- **Bug Fixes**: Detailed description of webhook event handling fixes ([#7032](https://github.com/kubeshop/testkube/pull/7032))
- **Complete Changelog**: All commits between v2.6.0 and v2.6.1

## Methods to Update the Release

Choose one of the following methods to update the release:

### Method 1: GitHub Actions Workflow (Recommended)

This is the easiest and most reliable method.

**Steps:**

1. Ensure this PR is merged to the main branch
2. Navigate to: https://github.com/kubeshop/testkube/actions/workflows/update-release-notes.yaml
3. Click the **"Run workflow"** button
4. Fill in the inputs:
   - **tag**: `2.6.1`
   - **release_notes_file**: `docs/release-notes/2.6.1.md`
5. Click **"Run workflow"** to start the job
6. Wait for the workflow to complete (should take < 1 minute)
7. Verify the updated release at: https://github.com/kubeshop/testkube/releases/tag/2.6.1

**Advantages:**
- No local setup required
- Uses GitHub's built-in authentication
- Audit trail in Actions history
- Can be run by anyone with write access to the repository

### Method 2: GitHub CLI

If you have the GitHub CLI (`gh`) installed and authenticated:

**Steps:**

```bash
# Clone the repository (if not already)
git clone https://github.com/kubeshop/testkube.git
cd testkube

# Checkout the branch with the release notes
git checkout main  # or the branch where this PR was merged

# Update the release
gh release edit 2.6.1 --notes-file docs/release-notes/2.6.1.md

# Verify
gh release view 2.6.1
```

**Advantages:**
- Simple one-line command
- Works from any machine with gh CLI
- Interactive verification

### Method 3: Python Script

Use the provided Python script for programmatic updates:

**Steps:**

```bash
# Clone the repository
git clone https://github.com/kubeshop/testkube.git
cd testkube

# Set your GitHub token (requires 'repo' scope)
export GITHUB_TOKEN=your_github_personal_access_token

# Run the update script
python3 update_release.py \
  --tag 2.6.1 \
  --notes-file docs/release-notes/2.6.1.md

# Verify the output
```

**Advantages:**
- Cross-platform compatibility
- No additional dependencies (uses Python stdlib)
- Good for automation and CI/CD

### Method 4: Bash Script

Use the provided bash script:

**Steps:**

```bash
# Clone the repository
git clone https://github.com/kubeshop/testkube.git
cd testkube

# Set your GitHub token
export GITHUB_TOKEN=your_github_personal_access_token

# Run the update script
./update-release-notes.sh

# The script will show success/failure status
```

**Advantages:**
- Simple shell script
- Uses curl and jq (commonly available)
- Good for Unix-like systems

## Verification

After updating the release, verify that it was successful:

1. **Visit the release page**: https://github.com/kubeshop/testkube/releases/tag/2.6.1

2. **Check for these sections**:
   - 🔒 Security Updates (CVE fixes)
   - 🐛 Bug Fixes (webhook event issues)
   - 📝 Changelog (with commit hashes)
   - Full Changelog link

3. **Verify the CVE fix is mentioned**:
   - Should see PR #7036 in the security section
   - Alpine base image update to 3.23.3 should be listed

4. **Verify the webhook fix details**:
   - Should see PR #7032 in bug fixes section
   - Should include detailed sub-sections for become events and execution queueing

## Troubleshooting

### GitHub CLI Not Authenticated

```bash
gh auth login
# Follow the prompts to authenticate
```

### Python Script Fails with "GITHUB_TOKEN not set"

```bash
# Create a GitHub Personal Access Token at:
# https://github.com/settings/tokens/new
# Required scopes: repo

export GITHUB_TOKEN=ghp_your_token_here
```

### Workflow Permission Denied

Ensure you have write access to the repository or ask a maintainer to run the workflow.

### Release Not Found

Ensure tag `2.6.1` exists:
```bash
git fetch --tags
git tag | grep 2.6.1
```

## Files Created/Modified

This PR adds the following files:

- `docs/release-notes/2.6.1.md` - Comprehensive release notes
- `docs/release-notes/README.md` - Documentation for release notes process
- `.github/workflows/update-release-notes.yaml` - GitHub Actions workflow
- `update_release.py` - Python script for updating releases
- `update-release-notes.sh` - Bash script for updating releases

## Next Steps

1. **Merge this PR** to include the release notes and tooling in the repository
2. **Run one of the update methods** above to publish the release notes to GitHub
3. **Verify** the release page shows the complete information
4. **Communicate** the release with updated notes to users/stakeholders

## Questions or Issues?

If you encounter any problems updating the release:

1. Check the GitHub Actions logs (for Method 1)
2. Verify your authentication and permissions
3. Ensure the tag `2.6.1` exists in the repository
4. Review the error messages carefully

For additional help, contact the Testkube maintainers or open an issue.
