#!/usr/bin/env bash

set -euo pipefail

verification_root="$(mktemp -d)"
source_dir="${verification_root}/source"
archive_path="${verification_root}/source.tar"
direct_gopath="${verification_root}/gopath"
direct_gocache="${verification_root}/gocache"
trap 'rm -rf "${verification_root}"' EXIT

mkdir -p "${source_dir}"
git archive --format=tar --output="${archive_path}" HEAD
tar -xf "${archive_path}" -C "${source_dir}"
cd "${source_dir}"

export GIT_LFS_SKIP_SMUDGE=1
export GOFLAGS=-modcacherw
export GOWORK=off
export GOPROXY=direct
export GOPATH="${direct_gopath}"
export GOCACHE="${direct_gocache}"

# Direct Git fetches for unreleased commits can report a transient shallow-file
# conflict after populating the VCS cache. A fresh attempt uses that cache.
for attempt in 1 2 3; do
  if go mod download all; then
    break
  fi
  if [[ "${attempt}" -eq 3 ]]; then
    exit 1
  fi
done
go list -m all > /dev/null
go list -deps -test ./... > /dev/null
