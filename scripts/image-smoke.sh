#!/usr/bin/env bash
set -euo pipefail
[ "$#" -eq 3 ] || { echo "usage: scripts/image-smoke.sh IMAGE NAME VERSION" >&2; exit 2; }
image=$1
name=$2
version=$3
container="${name}-smoke"
created=0
cleanup() {
  if [ "$created" -eq 1 ]; then
    docker logs "$container" || true
    docker rm -f "$container" >/dev/null || true
  fi
}
trap cleanup EXIT
[ "$(docker image inspect -f '{{.Config.User}}' "$image")" = '65532:65532' ]
docker run --rm "$image" --version | grep -F -- "$version"
output=$(mktemp)
if docker run --rm "$image" >"$output" 2>&1; then
  cat "$output"
  rm -f "$output"
  echo "image accepted missing configuration" >&2
  exit 1
fi
grep -F 'AUTH_PROVIDER is required' "$output"
rm -f "$output"
docker compose up -d --wait postgres
runtime=(--network "${name}_default" -e APP_ENVIRONMENT=development -e AUTH_PROVIDER=none -e 'DATABASE_URL=postgres://app:app@postgres:5432/app?sslmode=disable')
docker run --rm "${runtime[@]}" "$image" db migrate up
docker run -d --name "$container" "${runtime[@]}" "$image" >/dev/null
created=1
for ((attempt=0; attempt<60; attempt++)); do
  status=$(docker inspect -f '{{.State.Health.Status}}' "$container")
  if [ "$status" = healthy ]; then
    echo "image-smoke: healthy"
    exit 0
  fi
  [ "$(docker inspect -f '{{.State.Running}}' "$container")" = true ] || break
  sleep 1
done
echo "image-smoke: image did not become healthy" >&2
exit 1
