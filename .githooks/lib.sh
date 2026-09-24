#!/usr/bin/env bash

if [[ -t 2 && "${TERM:-}" != dumb ]]; then
	hook_blue='\033[34m'
	hook_green='\033[32m'
	hook_red='\033[31m'
	hook_reset='\033[0m'
else
	hook_blue=''
	hook_green=''
	hook_red=''
	hook_reset=''
fi

hook_step() {
	printf '%b→%b %s\n' "$hook_blue" "$hook_reset" "$*" >&2
}

hook_success() {
	printf '%b✓%b %s\n' "$hook_green" "$hook_reset" "$*" >&2
}

hook_fail() {
	printf '%b✗%b %s\n' "$hook_red" "$hook_reset" "$*" >&2
	exit 1
}
