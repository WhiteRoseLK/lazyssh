#!/usr/bin/env python3
"""
Validates that all external distribution repositories (Homebrew tap, Scoop bucket)
configured in .goreleaser.yaml exist on GitHub and are reachable before release.
"""

import os
import subprocess
import sys
import yaml

def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    repo_root = os.path.abspath(os.path.join(script_dir, "..", ".."))
    config_path = os.path.join(repo_root, ".goreleaser.yaml")

    if not os.path.exists(config_path):
        print(f"Error: {config_path} not found.", file=sys.stderr)
        sys.exit(1)

    with open(config_path, "r", encoding="utf-8") as f:
        config = yaml.safe_load(f)

    distribution_repos = []

    for brew in config.get("brews", []):
        repo = brew.get("repository", {})
        owner = repo.get("owner")
        name = repo.get("name")
        if owner and name:
            distribution_repos.append(("Homebrew tap", f"{owner}/{name}"))

    for scoop in config.get("scoops", []):
        repo = scoop.get("repository", {})
        owner = repo.get("owner")
        name = repo.get("name")
        if owner and name:
            distribution_repos.append(("Scoop bucket", f"{owner}/{name}"))

    if not distribution_repos:
        print("No external distribution repositories configured in .goreleaser.yaml.")
        sys.exit(0)

    failed = False
    print(f"Verifying {len(distribution_repos)} distribution repository target(s)...")

    for kind, repo in distribution_repos:
        print(f"Checking {kind}: {repo}...")
        # Try gh api first, fall back to curl
        cmd = ["gh", "api", f"repos/{repo}", "--silent"]
        res = subprocess.run(cmd, capture_output=True, text=True)
        if res.returncode == 0:
            print(f"  ✓ {kind} repository '{repo}' exists and is accessible.")
        else:
            print(
                f"  ❌ ERROR: {kind} repository '{repo}' does not exist on GitHub or is not accessible (HTTP 404/failure)!",
                file=sys.stderr,
            )
            print(f"     Please ensure https://github.com/{repo} exists before merging this release.", file=sys.stderr)
            failed = True

    if failed:
        sys.exit(1)

    print("All release distribution repositories verified successfully.")

if __name__ == "__main__":
    main()
