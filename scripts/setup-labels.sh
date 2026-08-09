#!/usr/bin/env bash
# Create the GitHub issue label vocabulary the skill pack relies on.
# Idempotent: --force overwrites colour and description on labels that exist.
set -euo pipefail

if ! command -v gh >/dev/null 2>&1; then
  echo "setup-labels: the GitHub CLI (gh) is not installed — https://cli.github.com" >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "setup-labels: gh is not authenticated. Run: gh auth login" >&2
  exit 1
fi

if ! repo=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null); then
  echo "setup-labels: no GitHub repository detected for this directory." >&2
  echo "Run this from a clone with a GitHub remote, or set one with: gh repo set-default" >&2
  exit 1
fi

echo "setup-labels: applying labels to $repo"

# name|colour|description
labels=(
  "spec|0e8a16|A spec issue — the what and why, produced by /to-spec"
  "ticket|1d76db|A tracer-bullet slice of a spec, produced by /to-tickets"
  "decision|5319e7|An open decision from /wayfinder; closed when decided"
  "bug|d73a4a|Defect report"
  "needs-info|fbca04|Triage: reporter must supply more before work can start"
  "needs-grilling|d4c5f9|Triage: requirements too fuzzy; run /grill-me"
  "ready|0075ca|Triage: understood, specified, agent can pick it up"
  "blocked|b60205|Waiting on another issue (body says which)"
  "wontfix|cfd3d7|Triage: closed deliberately"
)

for entry in "${labels[@]}"; do
  IFS='|' read -r name colour description <<<"$entry"
  gh label create "$name" --color "$colour" --description "$description" --force >/dev/null
  echo "  $name"
done

echo "setup-labels: ${#labels[@]} labels applied"
