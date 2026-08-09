#!/usr/bin/env bash
# Mirror the canonical skills in .claude/skills to the tool-neutral agents/skills
# tree so Codex / Pi / opencode have a non-Claude-branded path.
# EDIT SKILLS IN .claude/skills ONLY, then run: make sync-skills
set -euo pipefail

SRC=".claude/skills"
DST="agents/skills"

if [ ! -d "$SRC" ]; then
  echo "sync-skills: $SRC not found (run from repo root)" >&2
  exit 1
fi

rm -rf "$DST"
mkdir -p "$DST"
# copy every SKILL.md and any reference subdirs (e.g. code-review/reviewers)
cp -R "$SRC"/. "$DST"/

count=$(find "$DST" -name 'SKILL.md' | wc -l | tr -d ' ')
echo "sync-skills: mirrored $count skills to $DST"
