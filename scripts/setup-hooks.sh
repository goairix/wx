#!/usr/bin/env bash
set -euo pipefail

repository_root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	printf 'setup-hooks: not inside a Git repository\n' >&2
	exit 1
}
hook="$repository_root/.githooks/pre-commit"
if [[ ! -x "$hook" ]]; then
	printf 'setup-hooks: %s is missing or not executable\n' "$hook" >&2
	exit 1
fi

git -C "$repository_root" config core.hooksPath .githooks
printf 'setup-hooks: enabled .githooks for %s\n' "$repository_root"
