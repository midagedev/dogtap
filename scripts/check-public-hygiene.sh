#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

private_regex="${DOGTAP_PUBLIC_HYGIENE_REGEX:-}"

if [ -z "${private_regex}" ]; then
  printf 'Public hygiene check skipped: DOGTAP_PUBLIC_HYGIENE_REGEX is not set.\n' >&2
  exit 0
fi

hits_file="$(mktemp)"
trap 'rm -f "${hits_file}"' EXIT

(
  cd "${repo_root}"
  rg -n -i --hidden "${private_regex}" \
    --glob '!.git/**' \
    --glob '!.private/**' \
    --glob '!scripts/check-public-hygiene.sh' \
    --glob '!dist/**' \
    --glob '!web/dist/**' \
    --glob '!node_modules/**' \
    --glob '!test-results/**' \
    --glob '!testdata/**' \
    --glob '!tmp/**' \
    --glob '!*.png' \
    --glob '!*.jpg' \
    --glob '!*.jpeg' \
    --glob '!*.gif' \
    --glob '!*.ico' \
    --glob '!*.lock' > "${hits_file}" || true
)

if [ -s "${hits_file}" ]; then
  printf 'Public hygiene check found private or company-specific terms:\n' >&2
  cat "${hits_file}" >&2
  printf '\nMove project-specific evidence to .private/ or rewrite public docs/examples generically.\n' >&2
  exit 1
fi

printf 'Public hygiene check passed.\n'
