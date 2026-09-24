#!/usr/bin/env bash
set -euo pipefail

repository_root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	printf 'setup-hooks: not inside a Git repository\n' >&2
	exit 1
}
for hook_name in commit-msg pre-commit pre-merge-commit; do
	hook="$repository_root/.githooks/$hook_name"
	if [[ ! -x "$hook" ]]; then
		printf 'setup-hooks: %s is missing or not executable\n' "$hook" >&2
		exit 1
	fi
done

command -v go >/dev/null 2>&1 || {
	printf 'setup-hooks: go is not installed or not in PATH\n' >&2
	exit 1
}

tools_bin="$repository_root/.tools/bin"
export PATH="$tools_bin:$PATH"
required_tools='goimports=golang.org/x/tools/cmd/goimports@v0.34.0
shadow=golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@v0.34.0
golangci-lint=github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2'

mkdir -p "$tools_bin"
while IFS='=' read -r tool_name module; do
	if command -v "$tool_name" >/dev/null 2>&1; then
		if [[ "$tool_name" != golangci-lint ]] || golangci-lint version 2>/dev/null | grep -qE 'version v?2\.'; then
			printf 'setup-hooks: using %s from PATH\n' "$tool_name"
			continue
		fi
	fi
	printf 'setup-hooks: installing %s\n' "$tool_name"
	GOBIN="$tools_bin" go install "$module"
done <<<"$required_tools"

git -C "$repository_root" config core.hooksPath .githooks
printf 'setup-hooks: enabled .githooks with Go quality tools for %s\n' "$repository_root"
