#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
module_dir="$(cd "${script_dir}/.." && pwd)"

cd "${module_dir}"

proto_files=()
while IFS= read -r proto_file; do
  proto_files+=("${proto_file}")
done < <(find app -path '*/pb/*.proto' -type f | sort)
if [[ ${#proto_files[@]} -eq 0 ]]; then
  echo "no protobuf definitions found" >&2
  exit 1
fi

protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  "${proto_files[@]}"

echo "generated protobuf code for ${#proto_files[@]} definitions"
