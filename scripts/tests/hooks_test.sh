#!/usr/bin/env bash
set -euo pipefail

repository_root=$(git rev-parse --show-toplevel)
commit_msg_hook="$repository_root/.githooks/commit-msg"
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/wx-hooks-test.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT

assert_accepts() {
	local message=$1
	printf '%s\n' "$message" >"$temporary_directory/message"
	if ! "$commit_msg_hook" "$temporary_directory/message" >"$temporary_directory/output" 2>&1; then
		printf 'expected message to pass: %s\n' "$message" >&2
		cat "$temporary_directory/output" >&2
		exit 1
	fi
}

assert_rejects() {
	local message=$1
	printf '%s\n' "$message" >"$temporary_directory/message"
	if "$commit_msg_hook" "$temporary_directory/message" >"$temporary_directory/output" 2>&1; then
		printf 'expected message to fail: %s\n' "$message" >&2
		exit 1
	fi
}

assert_accepts 'feat(core): add cache refresh support'
assert_accepts 'fix!: correct request signature verification'
assert_accepts 'docs: 完善微信公众号接入说明'
assert_accepts "Merge branch 'topic'"
assert_accepts 'Revert "feat: add temporary API"'
assert_accepts 'fixup! feat(core): add cache refresh support'
assert_accepts 'amend! feat(core): add cache refresh support'

assert_rejects 'fix: fix'
assert_rejects 'update documentation'
assert_rejects 'feat(Invalid Scope): add cache refresh support'
assert_rejects 'feat: this subject is intentionally longer than seventy-two characters to verify the configured upper boundary'

printf 'hooks_test: all checks passed\n'
