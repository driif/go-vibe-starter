#!/usr/bin/env bash
set -euo pipefail
module=
name=
keep=0
no_docker=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --module|--name)
      [ "$#" -ge 2 ] || exit 2
      if [ "$1" = --module ]; then module=$2; else name=$2; fi
      shift 2 ;;
    --keep-demo) keep=1; shift ;;
    --no-docker) no_docker=1; shift ;;
    *) echo "usage: scripts/template-smoke.sh --module PATH --name NAME [--keep-demo] [--no-docker]" >&2; exit 2 ;;
  esac
done
[ -n "$module" ] && [ -n "$name" ] && [ "$name" != app ] || { echo 'smoke: supply a module and a name different from app' >&2; exit 2; }
dest=$(mktemp -d)
# Only this temporary project may be removed by the cleanup trap.
export COMPOSE_PROJECT_NAME="template_smoke_$(basename "$dest" | tr -cd '[:alnum:]' | tr '[:upper:]' '[:lower:]')"
passed=0
compose_started=0
cleanup() {
  if [ "$compose_started" -eq 1 ]; then (cd "$dest" && docker compose down -v) || true; fi
  if [ "$passed" -eq 1 ]; then rm -rf "$dest"; else echo "smoke: failed copy retained at $dest" >&2; fi
}
trap cleanup EXIT
commit() { git -c user.name=template-smoke -c user.email=template-smoke@example.invalid -c commit.gpgsign=false commit -q -m "$1"; }
clean() { [ -z "$(git status --porcelain)" ] || { git status --short; echo 'smoke: dirty tree' >&2; exit 1; }; }
echo '1. Copy staged template'
git checkout-index --all --prefix="$dest/"
cd "$dest"
git init -q -b main
git add -A
commit 'Copy template'
echo '2. Install tools'
make tools
echo '3. Initialise'
args=(--module "$module" --name "$name" --yes)
[ "$keep" -eq 0 ] || args+=(--keep-demo)
"$BASH" scripts/init.sh "${args[@]}" </dev/null
git add -A
commit 'Initialise project'
echo '4. Independent residue checks'
if git grep -nIiF -e 'github.com/driif/go-vibe-starter' -e 'driif'; then exit 1; fi
for path in .github/workflows/template-smoke.yml scripts/template-smoke.sh SECURITY.md LICENSE docs/designs; do
  [ ! -e "$path" ] || { echo "smoke: residue $path" >&2; exit 1; }
done
if [ "$keep" -eq 0 ]; then
  if git grep -nIF -e 'DEMO:' -e 'internal/domain/notes' -e 'internal/rpc/notes' -e 'NotesService' -e 'notes.proto' -e 'notes.sql' -e 'routes_notes' -e 'TestNotes' -- . ':!scripts/init.sh'; then exit 1; fi
  for path in proto/api/v1/notes.proto internal/domain internal/rpc/notes internal/db/gen; do [ ! -e "$path" ] || exit 1; done
fi
if git grep -nIE -e '(^|[^A-Za-z])chi([^A-Za-z]|$)' -e 'oapi|[Oo]pen[Aa][Pp][Ii]' -e 'lib/pq|database/sql|go-pkgz/rest|go-chi' -e 'errs\.Write|RenderJSON|URLParam|Querier|PrincipalFromContext|TokenFromContext' -e 'NewWithConfig|InitDB|GetEnv|testing\.Short|test -short' -e 'sql\.(Tx|ErrNoRows|Null)' -e 'internal/api|internal/server/(auth|errs|test)([^_A-Za-z]|$)|(^|[^A-Za-z.])pkg/' -e '(^|[^/A-Za-z])migrations/|(^|[^A-Za-z0-9_])/v1/' -e 'db-up|db-down|db seed|owner_id|SERVER_[A-Z]' -e 'DEMO|go-vibe-starter|Go Vibe Starter|driif' -- .claude/skills .claude/agents docs AGENTS.md CLAUDE.md CONTEXT.md README.md; then exit 1; fi
if git grep -nIE -e 'make (test|lint)' -- .claude/skills .claude/agents; then exit 1; fi
echo '5. Check initialized tree'
if [ "$no_docker" -eq 1 ]; then
  make gen
  go mod tidy
  go build ./...
  go vet ./...
  make lint
else
  make check
fi
clean
echo '6. Repeat init'
"$BASH" scripts/init.sh "${args[@]}" </dev/null
clean
if [ "$no_docker" -eq 0 ]; then
  echo '7. Check image'
  compose_started=1
  make image-smoke
fi
passed=1
echo 'template-smoke: passed'
