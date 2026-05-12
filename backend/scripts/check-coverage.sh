#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

profile="${1:-coverage.out}"
packages="$(go list ./internal/... | grep -v '/internal/testutil$')"

CGO_ENABLED=0 go test ${packages} -coverprofile="${profile}" -covermode=count
go tool cover -func="${profile}"

coverage="$(go tool cover -func="${profile}" | awk '/^total:/ {print $3}')"
if [ "${coverage}" != "100.0%" ]; then
  echo "backend coverage ${coverage}; expected 100.0%" >&2
  exit 1
fi
