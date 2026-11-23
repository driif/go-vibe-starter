#!/usr/bin/env bash
set -euo pipefail
src=.claude/skills
dst=agents/skills
[ -d "$src" ] || { echo "sync-skills: run from the repository root" >&2; exit 1; }
case "${1-}" in
  --check)
    [ "$#" -eq 1 ] || exit 2
    if ! diff -qr "$src" "$dst"; then
      echo "agents/skills is stale: run make sync-skills" >&2
      exit 1
    fi
    ;;
  "")
    [ "$#" -eq 0 ] || exit 2
    rm -rf "$dst"
    mkdir -p "$dst"
    cp -R "$src"/. "$dst"/
    ;;
  *) echo "usage: scripts/sync-skills.sh [--check]" >&2; exit 2 ;;
esac
