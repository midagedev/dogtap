#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

find "${repo_root}/scripts" -name '*.sh' -type f -print0 | xargs -0 -n 1 bash -n
