#!/usr/bin/env bash
# Bootstrap a project created from the go-vibe-starter template:
# rename the module, rename the binary, strip the DEMO slice, reset the docs.
# Safe to run more than once — every step is a no-op once it has been applied.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

DEMO_MARKER='DEMO: reference vertical slice'
DEMO_DOMAIN='notes'

NEW_MODULE=""
NEW_APP_NAME=""
KEEP_DEMO=0
ASSUME_YES=0

usage() {
  cat <<'EOF'
usage: scripts/init.sh [options]

  --module <path>   new Go module path (e.g. github.com/acme/billing)
  --name <name>     binary name written to bin/ (default: last module segment)
  --keep-demo       keep the notes reference slice instead of deleting it
  -y, --yes         take the defaults, never prompt
  -h, --help        show this message

With no flags the script asks for the module path and the app name.
EOF
}

say() { printf '%s\n' "$*"; }
warn() { printf 'init: %s\n' "$*" >&2; }
die() { printf 'init: %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    --module) [ $# -ge 2 ] || die "--module needs a value"; NEW_MODULE="$2"; shift 2 ;;
    --module=*) NEW_MODULE="${1#*=}"; shift ;;
    --name) [ $# -ge 2 ] || die "--name needs a value"; NEW_APP_NAME="$2"; shift 2 ;;
    --name=*) NEW_APP_NAME="${1#*=}"; shift ;;
    --keep-demo) KEEP_DEMO=1; shift ;;
    -y|--yes) ASSUME_YES=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; die "unknown option: $1" ;;
  esac
done

command -v go >/dev/null 2>&1 || die "go is not on PATH"
[ -f go.mod ] || die "go.mod not found — run this from the repository root"

CURRENT_MODULE="$(awk '$1 == "module" { print $2; exit }' go.mod)"
[ -n "$CURRENT_MODULE" ] || die "could not read the module path from go.mod"

# --- helpers ---------------------------------------------------------------

# escape a string so it is literal on both sides of a sed s|...|...| command
sed_escape() { printf '%s' "$1" | sed -e 's/[\\|&]/\\&/g'; }

# rewrite a file through a temp copy: BSD and GNU sed disagree about -i
rewrite_file() {
  local target="$1"; shift
  local tmp
  tmp="$(mktemp)"
  sed "$@" "$target" >"$tmp"
  if ! cmp -s "$target" "$tmp"; then
    cat "$tmp" >"$target"
  fi
  rm -f "$tmp"
}

ask() { # ask <question> <default> -> answer on stdout
  local question="$1" default="$2" answer=""
  if [ "$ASSUME_YES" -eq 1 ] || [ ! -t 0 ]; then
    printf '%s' "$default"
    return 0
  fi
  read -r -p "$question [$default]: " answer </dev/tty || answer=""
  printf '%s' "${answer:-$default}"
}

confirm() { # confirm <question> ; default no
  local answer
  if [ "$ASSUME_YES" -eq 1 ] || [ ! -t 0 ]; then
    return 1
  fi
  read -r -p "$1 [y/N]: " answer </dev/tty || answer=""
  case "$answer" in [yY]|[yY][eE][sS]) return 0 ;; *) return 1 ;; esac
}

capitalize() { printf '%s%s' "$(printf '%s' "${1:0:1}" | tr '[:lower:]' '[:upper:]')" "${1:1}"; }

# --- gather answers --------------------------------------------------------

if [ -z "$NEW_MODULE" ]; then
  NEW_MODULE="$(ask "New Go module path" "$CURRENT_MODULE")"
fi
case "$NEW_MODULE" in
  *[![:alnum:]._/~-]*|"") die "invalid module path: '$NEW_MODULE'" ;;
esac

if [ -z "$NEW_APP_NAME" ]; then
  NEW_APP_NAME="$(ask "Binary name" "${NEW_MODULE##*/}")"
fi
case "$NEW_APP_NAME" in
  *[![:alnum:]._-]*|"") die "invalid binary name: '$NEW_APP_NAME'" ;;
esac

say "init: module $CURRENT_MODULE -> $NEW_MODULE"
say "init: binary $NEW_APP_NAME"

# --- 1. module path --------------------------------------------------------

if [ "$NEW_MODULE" != "$CURRENT_MODULE" ]; then
  go mod edit -module "$NEW_MODULE"
  old_escaped="$(sed_escape "$CURRENT_MODULE")"
  new_escaped="$(sed_escape "$NEW_MODULE")"
  count=0
  while IFS= read -r file; do
    [ -n "$file" ] || continue
    rewrite_file "$file" -e "s|${old_escaped}|${new_escaped}|g"
    count=$((count + 1))
  done <<EOF
$(grep -rlF "$CURRENT_MODULE" --include='*.go' --include='*.md' \
    --exclude-dir=.git --exclude-dir=bin . 2>/dev/null || true)
EOF
  say "init: rewrote import paths in $count files"
else
  say "init: module path already set, skipping"
fi

# --- 2. binary name --------------------------------------------------------

rewrite_file Makefile -e "s|^APP_NAME := .*|APP_NAME := $(sed_escape "$NEW_APP_NAME")|"

# --- 3. DEMO slice ---------------------------------------------------------

strip_openapi_demo() {
  local spec="oapi/openapi.yaml"
  [ -f "$spec" ] || return 0
  grep -q "/v1/${DEMO_DOMAIN}" "$spec" || return 0
  if ! command -v python3 >/dev/null 2>&1; then
    warn "python3 not found — remove the /v1/${DEMO_DOMAIN} paths and schemas from $spec by hand"
    return 0
  fi
  if ! python3 - "$spec" "$DEMO_DOMAIN" "$DEMO_MARKER" "$NEW_APP_NAME" <<'PY'
import json, sys

path, domain, marker, app = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
with open(path) as fh:
    doc = json.load(fh)

info = doc.setdefault("info", {})
info["title"] = app + " API"
if domain in info.get("description", "").lower():
    del info["description"]

for key in [k for k in doc.get("paths", {}) if k.startswith("/v1/" + domain)]:
    del doc["paths"][key]

tag = domain[:1].upper() + domain[1:]
tags = [t for t in doc.get("tags", [])
        if t.get("name") != tag and marker not in t.get("description", "")]
if tags:
    doc["tags"] = tags
elif "tags" in doc:
    del doc["tags"]


def collect(node, out):
    """gather every $ref target under node."""
    if isinstance(node, dict):
        for key, value in node.items():
            if key == "$ref" and isinstance(value, str):
                out.add(value)
            else:
                collect(value, out)
    elif isinstance(node, list):
        for value in node:
            collect(value, out)


def resolve(ref):
    node = doc
    for part in ref.lstrip("#/").split("/"):
        if not isinstance(node, dict) or part not in node:
            return None
        node = node[part]
    return node


# walk out from the surviving paths; anything under components the walk never
# reaches only existed for the demo slice
reachable, frontier = set(), set()
collect(doc.get("paths", {}), frontier)
while frontier:
    ref = frontier.pop()
    if ref in reachable:
        continue
    reachable.add(ref)
    target = resolve(ref)
    if target is not None:
        found = set()
        collect(target, found)
        frontier |= found - reachable

components = doc.get("components", {})
for section in ("schemas", "responses", "parameters", "requestBodies", "headers", "examples"):
    entries = components.get(section)
    if not isinstance(entries, dict):
        continue
    for name in [n for n in entries
                 if "#/components/%s/%s" % (section, n) not in reachable]:
        del entries[name]
    if not entries:
        del components[section]

with open(path, "w") as fh:
    json.dump(doc, fh, indent=2)
    fh.write("\n")
PY
  then
    warn "could not rewrite $spec automatically — remove the /v1/${DEMO_DOMAIN} paths by hand"
  fi
}

if [ "$KEEP_DEMO" -eq 1 ]; then
  say "init: keeping the ${DEMO_DOMAIN} reference slice"
else
  # hand-written sources only: generated output carries the marker too, and
  # 'make gen' rewrites it from the stripped spec and queries
  demo_files="$(grep -rlF "$DEMO_MARKER" --include='*.go' --include='*.sql' \
    internal migrations sql 2>/dev/null \
    | grep -v -e '\.gen\.go$' -e '^internal/db/gen/' || true)"

  if [ -z "$demo_files" ]; then
    say "init: DEMO slice already removed, skipping"
  else
    # a deleted internal/api/router/routes_<domain>.go means routes.go still
    # calls register<Domain>; drop that call from the hub
    while IFS= read -r file; do
      [ -n "$file" ] || continue
      case "$file" in
        internal/api/router/routes_*.go)
          domain="${file##*/routes_}"
          domain="${domain%.go}"
          if [ -f internal/api/router/routes.go ]; then
            rewrite_file internal/api/router/routes.go -e "/register$(capitalize "$domain")(/d"
          fi
          ;;
      esac
      rm -f "$file"
      say "  removed $file"
    done <<EOF
$demo_files
EOF

    rm -f "internal/db/gen/${DEMO_DOMAIN}.sql.go"
    # sqlc refuses to run with no queries at all, so once the demo query file is
    # gone its stale output has to go too or the package stops compiling
    if [ -d internal/db/gen ] && ! ls sql/queries/*.sql >/dev/null 2>&1; then
      rm -f internal/db/gen/*.go
      say "  emptied internal/db/gen (sqlc regenerates it from your first query)"
    fi
    strip_openapi_demo
    say "init: DEMO slice stripped — run 'make gen-oapi' now, 'make sqlc' once you have queries"
  fi
fi

# --- 4. docs reset ---------------------------------------------------------

reset_docs() {
  cat >README.md <<EOF
# ${NEW_APP_NAME}

## Quick start

\`\`\`sh
make tools     # install the pinned codegen and lint tools
make db-up     # start postgres (and keycloak) via docker compose
make migrate-up
make run
\`\`\`

Run \`make help\` for the full target list.

## Working in this repo

Read [AGENTS.md](./AGENTS.md) first: stack, conventions, commands, and the
workflow agents follow. [CONTEXT.md](./CONTEXT.md) holds the domain language.
EOF

  cat >CONTEXT.md <<'EOF'
# Context

The domain language for this project. Keep it accurate — agents read it before
they touch code, and drift here shows up as drift in the code.

## Ubiquitous language

| Term | Means |
|---|---|
| | |

## Module map

| Package | Owns |
|---|---|
| `cmd` | CLI entry points (`run`, `db migrate`, `db seed`) |
| `internal/api/handlers` | HTTP handlers, one file per domain |
| `internal/api/router` | Route registration, one file per domain |
| `internal/server` | Server construction, config, auth, middleware |
| `internal/db/gen` | sqlc output — generated, never hand-edited |
| `pkg` | Reusable helpers with no project-specific knowledge |
EOF

  mkdir -p docs/plans/completed docs/adr
  # README.md in those directories documents the format, not the content
  find docs/plans docs/adr -type f -name '*.md' ! -name 'README.md' -delete
  touch docs/plans/.gitkeep docs/plans/completed/.gitkeep docs/adr/.gitkeep
}

if [ "$ASSUME_YES" -eq 1 ] || confirm "Reset README.md, CONTEXT.md, docs/plans and docs/adr?"; then
  reset_docs
  say "init: docs reset"
else
  say "init: left README.md, CONTEXT.md and docs/ alone"
fi

# --- 5. labels -------------------------------------------------------------

if confirm "Create the GitHub issue labels now (needs gh auth)?"; then
  ./scripts/setup-labels.sh
else
  say "init: skipping labels — run 'make labels' when the repo exists on GitHub"
fi

# --- 6. next steps ---------------------------------------------------------

cat <<EOF

Done. Next:

  1. make tools        install the pinned codegen and lint tools
  2. make gen          regenerate the API and sqlc layers
  3. make db-up        start postgres via docker compose
  4. make migrate-up   apply migrations
  5. make test         go test -race ./...
  6. cp .env.example .env.local and fill in the values you need

Then describe your first feature and let the skill pack take it from there.
EOF
