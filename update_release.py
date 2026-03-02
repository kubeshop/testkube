#!/usr/bin/env python3
"""
Script to update GitHub release notes for Testkube releases.
Usage: python3 update_release.py --tag 2.6.1 --notes-file docs/release-notes/2.6.1.md
"""

import argparse
import json
import os
import sys
import urllib.request
from pathlib import Path


def update_release(owner: str, repo: str, tag: str, notes_file: Path, token: str):
    """Update a GitHub release with new notes."""
    
    # Read release notes
    if not notes_file.exists():
        print(f"Error: Release notes file not found: {notes_file}")
        sys.exit(1)
    
    with open(notes_file, 'r') as f:
        release_body = f.read()
    
    # First, get the release by tag
    api_url = f"https://api.github.com/repos/{owner}/{repo}/releases/tags/{tag}"
    
    headers = {
        'Accept': 'application/vnd.github+json',
        'Authorization': f'Bearer {token}',
        'X-GitHub-Api-Version': '2022-11-28'
    }
    
    print(f"Fetching release for tag {tag}...")
    req = urllib.request.Request(api_url, headers=headers)
    
    try:
        with urllib.request.urlopen(req) as response:
            release_data = json.loads(response.read())
            release_id = release_data['id']
            print(f"Found release ID: {release_id}")
    except urllib.error.HTTPError as e:
        print(f"Error fetching release: {e}")
        print(f"Response: {e.read().decode()}")
        sys.exit(1)
    
    # Update the release
    update_url = f"https://api.github.com/repos/{owner}/{repo}/releases/{release_id}"
    
    data = json.dumps({'body': release_body}).encode('utf-8')
    
    req = urllib.request.Request(
        update_url,
        data=data,
        headers={**headers, 'Content-Type': 'application/json'},
        method='PATCH'
    )
    
    print(f"Updating release notes...")
    
    try:
        with urllib.request.urlopen(req) as response:
            result = json.loads(response.read())
            print(f"✅ Release notes updated successfully!")
            print(f"View at: {result['html_url']}")
    except urllib.error.HTTPError as e:
        print(f"❌ Error updating release: {e}")
        print(f"Response: {e.read().decode()}")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description='Update GitHub release notes for Testkube'
    )
    parser.add_argument(
        '--tag',
        required=True,
        help='Release tag (e.g., 2.6.1)'
    )
    parser.add_argument(
        '--notes-file',
        required=True,
        type=Path,
        help='Path to release notes markdown file'
    )
    parser.add_argument(
        '--owner',
        default='kubeshop',
        help='Repository owner (default: kubeshop)'
    )
    parser.add_argument(
        '--repo',
        default='testkube',
        help='Repository name (default: testkube)'
    )
    
    args = parser.parse_args()
    
    # Get token from environment
    token = os.environ.get('GITHUB_TOKEN')
    if not token:
        print("Error: GITHUB_TOKEN environment variable not set")
        print("Please set it with: export GITHUB_TOKEN=your_token")
        sys.exit(1)
    
    update_release(
        args.owner,
        args.repo,
        args.tag,
        args.notes_file,
        token
    )


if __name__ == '__main__':
    main()
