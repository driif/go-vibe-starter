#!/usr/bin/env bash
set -euo pipefail
usage() {
  cat <<'EOF'
usage: scripts/init.sh [--module PATH] [--name NAME] [--keep-demo] [-y|--yes]
  --module PATH  required on the first run; defaults to the current module later
  --name NAME    binary, image and compose project name; defaults to module basename
  --keep-demo    keep the reference slice on the first run (manual removal later)
  -y, --yes      take flags and defaults without prompting
Exit codes: 0 done, 1 failed step, 2 invalid usage.
EOF
}
fail() { echo "init: $*" >&2; exit 1; }
invalid() { echo "init: $*" >&2; exit 2; }
module=
name=
yes=0
keep=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --module|--name)
      [ "$#" -ge 2 ] || invalid "$1 needs a value"
      if [ "$1" = --module ]; then module=$2; else name=$2; fi
      shift 2 ;;
    --keep-demo) keep=1; shift ;;
    -y|--yes) yes=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) invalid "unknown argument: $1" ;;
  esac
done
[ -f go.mod ] && [ -f Makefile ] && [ -f docker-compose.yaml ] || fail 'run from the repository root'
old=$(awk '$1 == "module" {print $2; exit}' go.mod)
old_name=$(sed -n 's/^APP_NAME := //p' Makefile)
first=0
[ ! -e scripts/template-smoke.sh ] || first=1
if [ "$yes" -eq 0 ]; then
  [ -t 0 ] || invalid 'not a terminal; pass --module, --name and --yes'
  read -r -p "Go module path [${module:-$old}]: " answer
  module=${answer:-${module:-$old}}
else
  [ "$first" -eq 0 ] || [ -n "$module" ] || invalid '--module is required on the first run'
fi
module=${module:-$old}
module_re='^[a-z0-9][a-z0-9.-]*\.[a-z][a-z0-9-]*(/[A-Za-z0-9_~-][A-Za-z0-9._~-]*)+$'
[[ "$module" =~ $module_re ]] || invalid 'invalid module path: use a domain followed by a path'
if [ -z "$name" ]; then
  base=${module##*/}
  if [[ "$base" =~ ^v[0-9]+$ ]]; then parent=${module%/*}; base=${parent##*/}; fi
  name=$(printf '%s' "$base" | tr '[:upper:]' '[:lower:]')
fi
if [ "$yes" -eq 0 ]; then
  read -r -p "Project name [$name]: " answer
  name=${answer:-$name}
  if [ "$first" -eq 1 ] && [ "$keep" -eq 0 ]; then
    read -r -p 'Keep the demo slice? [y/N] ' answer
    case "$answer" in y|Y|yes|YES) keep=1 ;; esac
  fi
fi
name_re='^[a-z0-9]+([_-][a-z0-9]+)*$'
[[ "$name" =~ $name_re ]] && [ "${#name}" -le 50 ] || invalid 'invalid name: use at most 50 lowercase letters, digits, hyphens or underscores'
for tool in git go make; do command -v "$tool" >/dev/null || fail "$tool is required"; done
[ "$(git rev-parse --show-toplevel)" = "$PWD" ] || fail 'run from the git work tree root'
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cp go.mod "$tmp/check.mod"
go mod edit -modfile="$tmp/check.mod" -module "$module" || invalid 'Go rejected the module path'
make tools-check
if [ "$yes" -eq 0 ]; then
  echo "Initialise $name ($module); keep reference slice: $keep."
  read -r -p 'Proceed? [y/N] ' answer
  case "$answer" in y|Y|yes|YES) ;; *) invalid 'cancelled' ;; esac
fi
rewrite() {
  target=$1
  expression=$2
  sed -E "$expression" "$target" >"$tmp/rewrite"
  if ! cmp -s "$target" "$tmp/rewrite"; then cat "$tmp/rewrite" >"$target"; fi
}
: >"$tmp/deleted"
if [ "$first" -eq 1 ]; then
  rm -f .github/workflows/template-smoke.yml SECURITY.md LICENSE
  rm -rf docs/designs
  find docs/plans docs/adr -type f -name '*.md' -delete
  cat >README.md <<EOF
# $name

Go backend. Read [AGENTS.md](AGENTS.md) for the layout and conventions and
[CONTEXT.md](CONTEXT.md) for the domain language.

\`\`\`sh
make tools
make dev-up
make migrate-up
make run
\`\`\`

In another terminal:

\`\`\`sh
curl -fsS http://localhost:8080/readyz
curl -fsS -H 'Content-Type: application/json' -d '{}' http://localhost:8080/api.v1.WhoAmIService/WhoAmI
make check
\`\`\`

Start frontend work separately in \`frontend/\`. See [authentication](docs/auth.md)
and [deployment](docs/production.md). Add your own licence.
Run \`/setup-project\` in your agent to set up the project vocabulary and issue labels.
EOF
  if [ "$keep" -eq 0 ]; then
    while IFS= read -r -d '' file; do
      [ -f "$file" ] || continue
      case "$file" in internal/gen/*|internal/db/gen/*) continue ;; esac
      IFS= read -r line <"$file" || true
      if [[ "$line" =~ ^(//|--)\ DEMO:\  ]]; then
        printf '%s\n' "$file" >>"$tmp/deleted"
        rm -f "$file"
      elif [[ "$file" = *.go ]]; then
        rewrite "$file" '/\/\/ DEMO:line$/d'
      fi
    done < <(git ls-files -z --cached --others --exclude-standard -- internal proto sql)
    find internal proto -depth -type d -empty -delete
  else
    echo 'demo kept: init will not remove it later. Removal is manual, and its migration is version 1 of every database it was applied to.'
  fi
  rm -f scripts/template-smoke.sh
fi
if [ "$old" != "$module" ]; then
  escaped_old=$(printf '%s' "$old" | sed 's/[][\.^$*+?(){}|\\]/\\&/g')
  escaped_new=$(printf '%s' "$module" | sed 's/[&|\\]/\\&/g')
  while IFS= read -r -d '' file; do
    [ "$file" != go.mod ] || continue
    rewrite "$file" "s|${escaped_old}([^A-Za-z0-9_~-]|$)|${escaped_new}\\1|g"
  done < <(git grep -Iz --untracked -l -F -e "$old" -- . ':!internal/gen' ':!internal/db/gen' || true)
  go mod edit -module "$module"
fi
[ -n "$old_name" ] || fail 'APP_NAME anchor missing in Makefile'
grep -q '^name: ' docker-compose.yaml || fail 'name anchor missing in docker-compose.yaml'
rewrite Makefile "s/^APP_NAME := .*$/APP_NAME := $name/"
rewrite docker-compose.yaml "s/^name: .*$/name: $name/"
make gen
go mod tidy
make sync-skills
go build ./...
go vet ./...
make build
if [ "$old" != "$module" ] && git grep -nIF --untracked -e "$old" -- . ':!scripts/init.sh'; then
  fail 'old module path remains'
fi
present=0
while IFS= read -r -d '' file; do
  [ -f "$file" ] || continue
  case "$file" in internal/gen/*|internal/db/gen/*) continue ;; esac
  IFS= read -r line <"$file" || true
  if [[ "$line" =~ ^(//|--)\ DEMO:\  ]]; then present=1; break; fi
done < <(git ls-files -z --cached --others --exclude-standard -- internal proto sql)
if [ "$present" -eq 0 ]; then
  if git grep -nIF --untracked -e 'DEMO:' -- . ':!scripts/init.sh'; then fail 'reference marker remains'; fi
  while IFS= read -r file; do
    if git grep -nIF --untracked -e "$file" -- . ':!scripts/init.sh'; then fail 'deleted reference path remains'; fi
  done <"$tmp/deleted"
fi
printf 'Initialised %s (%s).\n' "$name" "$module"
[ "$first" -eq 0 ] || echo 'Removed LICENSE: add your own.'
cat <<'EOF'
Next:
  1. git add -A && git commit -m "Initialise project"
  2. make dev-up        Postgres on 127.0.0.1:5434
  3. make migrate-up
  4. make run           port 8080, local auth from .env.local
  5. make check         the gate CI runs
  6. make labels        once the repository exists on GitHub (needs gh)
Then run /setup-project in your agent.
EOF
