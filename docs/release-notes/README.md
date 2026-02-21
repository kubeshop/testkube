# Release Notes for Testkube 2.6.1

This directory contains the detailed release notes for Testkube version 2.6.1.

## Release Highlights

This patch release includes important security updates and bug fixes for webhook event handling.

### Security Updates
- **CVE Fixes (February 2025)**: Updated Alpine base image to 3.23.3 and dependencies to address reported CVEs

### Bug Fixes
- **[TKC-4867] Webhook Event Issues**: Comprehensive fix for misleading logs and incorrect event types in webhook handling

## Full Release Notes

See [2.6.1.md](./2.6.1.md) for the complete, detailed release notes.

## Updating the GitHub Release

The detailed release notes in this directory can be published to the GitHub release using one of the following methods:

### Method 1: GitHub Actions Workflow (Recommended)

Use the provided workflow to update the release:

1. Go to: https://github.com/kubeshop/testkube/actions/workflows/update-release-notes.yaml
2. Click "Run workflow"
3. Enter the tag: `2.6.1`
4. Enter the notes file path: `docs/release-notes/2.6.1.md`
5. Click "Run workflow"

### Method 2: Python Script

```bash
export GITHUB_TOKEN=your_github_token
python3 update_release.py --tag 2.6.1 --notes-file docs/release-notes/2.6.1.md
```

### Method 3: Bash Script

```bash
export GITHUB_TOKEN=your_github_token
# Run with defaults (tag 2.6.1)
./update-release-notes.sh

# Or specify tag and notes file
./update-release-notes.sh 2.6.1 docs/release-notes/2.6.1.md
```

### Method 4: GitHub CLI

```bash
gh release edit 2.6.1 --notes-file docs/release-notes/2.6.1.md
```

## Release Structure

Each release note file should include:

1. **Title**: Release version
2. **Overview**: Brief description of the release
3. **Security Updates**: CVE fixes and security improvements
4. **New Features**: Major new capabilities (if applicable)
5. **Bug Fixes**: Important fixes with detailed descriptions
6. **Changelog**: Commit-level details
7. **Full Changelog Link**: Comparison link to previous version

## Related Files

- `docs/release-notes/2.6.1.md` - Detailed release notes
- `.github/workflows/update-release-notes.yaml` - GitHub Actions workflow for updating releases
- `update_release.py` - Python script for updating releases
- `update-release-notes.sh` - Bash script for updating releases
